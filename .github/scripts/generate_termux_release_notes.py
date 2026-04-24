#!/usr/bin/env python3
import json
import os
import sys
import urllib.request


def fetch_release(repo: str, tag: str) -> dict:
    req = urllib.request.Request(
        f"https://api.github.com/repos/{repo}/releases/tags/{tag}",
        headers={"User-Agent": "ollama-termux-release-notes"},
    )
    with urllib.request.urlopen(req) as resp:
        return json.load(resp)


def main() -> int:
    out_path = sys.argv[1]
    version = os.environ["TERMUX_VERSION"]
    upstream_repo = os.environ.get("UPSTREAM_REPO", "ollama/ollama")
    upstream_tag = os.environ["UPSTREAM_TAG"]
    reuse_tag = os.environ.get("TERMUX_REUSE_RELEASE_TAG", "")

    release = fetch_release(upstream_repo, upstream_tag)
    upstream_name = release.get("name") or upstream_tag
    upstream_body = (release.get("body") or "").strip()

    lines = [
        f"ollama-termux {version}",
        "",
        f"Built from upstream {upstream_repo} {upstream_tag} and adapted for Termux-first Android ARM64 packaging.",
        "",
        "## Termux adaptation",
        "- Launcher CLI support on Termux stays limited to Codex (primary), Qwen Code (secondary), and Claude Code (frozen at 2.1.112).",
        "- Unsupported upstream coding CLIs remain in source for merge safety but are disabled on Termux runtime.",
        "- codex-vl is not included in this release.",
    ]
    if reuse_tag:
        lines.append(f"- Android ARM64 optimized libraries are reused from validated release assets: {reuse_tag}.")

    lines.extend(
        [
            "",
            f"Release assets:",
            f"- ollama-termux-{version}-android-arm64.tar.gz",
            f"- ollama-termux-{version}-android-arm64.tar.gz.sha256",
            "",
            f"## Upstream {upstream_name} notes",
            upstream_body,
            "",
        ]
    )

    with open(out_path, "w", encoding="utf-8") as fh:
        fh.write("\n".join(lines).rstrip() + "\n")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
