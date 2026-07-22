#!/usr/bin/env python3
import json
import os
import sys
import urllib.error
import urllib.request


def fetch_release(repo: str, tag: str) -> dict:
    headers = {"User-Agent": "ollama-termux-release-notes"}
    # Authenticate to use the 5000/hour limit instead of the 60/hour anonymous
    # limit, which shared GitHub Actions runner IPs routinely exhaust (HTTP 403).
    token = os.environ.get("GITHUB_TOKEN") or os.environ.get("GH_TOKEN")
    if token:
        headers["Authorization"] = f"Bearer {token}"
    req = urllib.request.Request(
        f"https://api.github.com/repos/{repo}/releases/tags/{tag}",
        headers=headers,
    )
    try:
        with urllib.request.urlopen(req) as resp:
            return json.load(resp)
    except (urllib.error.HTTPError, urllib.error.URLError, TimeoutError) as exc:
        # Upstream notes are best-effort; never let an API hiccup block a release.
        print(
            f"warning: could not fetch upstream release {repo}@{tag}: {exc}",
            file=sys.stderr,
        )
        return {}


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
        "- Launcher CLI support on Termux stays limited to Codex (Termux fork), Codex VL, Qwen Code (Termux fork), and Pi.",
        "- Unsupported upstream coding CLIs are disabled on Termux runtime.",
        "- Android's system Vulkan loader now precedes Termux Mesa, preventing silent llvmpipe fallback.",
        "- The mobile memory budget never inflates MemAvailable and backs off under Android zram/swap pressure.",
        "- The npm installer requires a valid SHA-256 file and rejects unsafe or incomplete archives before installation.",
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
            "## Device compatibility",
            (
                "Vulkan behavior depends on the Android vendor driver. If model "
                "loading is unstable, restart the server with `OLLAMA_VULKAN=0` "
                "for CPU fallback."
            ),
            (
                "The Adreno 650 investigation remains open in "
                "[issue #1](https://github.com/DioNanos/ollama-termux/issues/1)."
            ),
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
