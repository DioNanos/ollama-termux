#!/usr/bin/env python3
"""Fail-closed validation for ollama-termux Android release archives."""

from __future__ import annotations

import argparse
import hashlib
from pathlib import Path, PurePosixPath
import re
import sys
import tarfile


MAX_MEMBERS = 4096
MAX_UNPACKED_BYTES = 4 * 1024 * 1024 * 1024
SHA256_LINE = re.compile(r"^([0-9a-fA-F]{64})\s+\*?([^\r\n]+)\s*$")


def normalized_member_name(raw: str) -> str:
    while raw.startswith("./"):
        raw = raw[2:]
    raw = raw.rstrip("/")
    if not raw or raw == ".":
        return ""
    path = PurePosixPath(raw)
    if path.is_absolute() or ".." in path.parts or "\x00" in raw:
        raise ValueError(f"unsafe archive path: {raw!r}")
    return path.as_posix()


def allowed_member(name: str) -> bool:
    return (
        name == "install.js"
        or name == "bin"
        or name.startswith("bin/")
        or name == "lib"
        or name == "lib/ollama"
        or name.startswith("lib/ollama/")
    )


def read_expected_sha(checksum: Path, archive: Path) -> str:
    match = SHA256_LINE.fullmatch(checksum.read_text(encoding="utf-8"))
    if not match:
        raise ValueError(f"invalid SHA256 file: {checksum}")
    if match.group(2) != archive.name:
        raise ValueError(
            f"checksum names {match.group(2)!r}, expected {archive.name!r}"
        )
    return match.group(1).lower()


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def verify_archive(archive: Path, require_vulkan: bool) -> tuple[int, int]:
    names: set[str] = set()
    modes: dict[str, int] = {}
    unpacked_bytes = 0

    with tarfile.open(archive, mode="r:gz") as payload:
        members = payload.getmembers()
        if len(members) > MAX_MEMBERS:
            raise ValueError(
                f"archive contains {len(members)} members; maximum is {MAX_MEMBERS}"
            )

        for member in members:
            name = normalized_member_name(member.name)
            if not name:
                continue
            if name in names:
                raise ValueError(f"duplicate archive member: {name}")
            if not allowed_member(name):
                raise ValueError(f"unexpected archive member: {name}")
            if not (member.isdir() or member.isreg()):
                raise ValueError(f"link or special archive member rejected: {name}")
            if member.mode & 0o7000:
                raise ValueError(f"privileged archive mode rejected: {name}")

            names.add(name)
            modes[name] = member.mode
            if member.isreg():
                unpacked_bytes += member.size
                if unpacked_bytes > MAX_UNPACKED_BYTES:
                    raise ValueError(
                        f"archive expands beyond {MAX_UNPACKED_BYTES} bytes"
                    )

    required = {
        "install.js",
        "bin/ollama",
        "lib/ollama/llama-server",
        "lib/ollama/libggml-base.so",
    }
    missing = sorted(required - names)
    if missing:
        raise ValueError(f"archive is missing required members: {', '.join(missing)}")
    if not any(re.fullmatch(r"lib/ollama/libggml-cpu.*\.so", name) for name in names):
        raise ValueError("archive is missing a ggml CPU backend")
    if require_vulkan and not any(
        re.fullmatch(r"lib/ollama/vulkan/libggml-vulkan.*\.so", name)
        for name in names
    ):
        raise ValueError("archive is missing the required Vulkan backend")

    for executable in ("bin/ollama", "lib/ollama/llama-server"):
        if modes[executable] & 0o111 == 0:
            raise ValueError(f"required executable has no execute bit: {executable}")

    return len(names), unpacked_bytes


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("archive", type=Path)
    parser.add_argument("checksum", type=Path)
    parser.add_argument("--require-vulkan", action="store_true")
    args = parser.parse_args()

    expected = read_expected_sha(args.checksum, args.archive)
    actual = sha256_file(args.archive)
    if actual != expected:
        raise ValueError(f"SHA256 mismatch: expected {expected}, got {actual}")

    members, unpacked = verify_archive(args.archive, args.require_vulkan)
    print(
        f"archive verified: sha256={actual} members={members} "
        f"unpacked_bytes={unpacked}"
    )
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (OSError, ValueError, tarfile.TarError) as exc:
        print(f"archive verification failed: {exc}", file=sys.stderr)
        raise SystemExit(1)
