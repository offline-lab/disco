#!/usr/bin/env -S uv run python
# /// script
# dependencies = ["zensical"]
# ///
"""Build the disco documentation site using Zensical."""

import http.server
import functools
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

MEDIA_REPO = "https://github.com/offline-lab/media.git"
MEDIA_BRANCH = "main"


def fetch_images():
    dest = Path("docs/public/images")
    with tempfile.TemporaryDirectory() as tmp:
        result = subprocess.run(
            ["git", "clone", "--depth=1", "--branch", MEDIA_BRANCH, MEDIA_REPO, tmp],
            capture_output=True,
            text=True,
        )

        if result.returncode != 0:
            raise SystemExit(f"Failed to clone {MEDIA_REPO}: {result.stderr.strip()}")
        src = Path(tmp) / "images"

        if not src.is_dir():
            raise SystemExit(f"No images/ directory found in {MEDIA_REPO}")

        if dest.exists():
            shutil.rmtree(dest)
        shutil.copytree(src, dest)


def build():
    subprocess.run(["zensical", "build"], check=True)
    fetch_images()


def serve(bind="127.0.0.1", port=8000):
    try:
        build()

        handler = functools.partial(
            http.server.SimpleHTTPRequestHandler, directory="docs/public"
        )

        with http.server.HTTPServer((bind, port), handler) as httpd:
            print(f"Serving public_html at http://{bind}:{port}/")
            httpd.serve_forever()

    except KeyboardInterrupt:
        exit(1)


def main():
    if len(sys.argv) > 1 and sys.argv[1] == "serve":
        serve()
    else:
        build()


if __name__ == "__main__":
    main()
