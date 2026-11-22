"""Classify a PR conservatively: only Markdown and Issue forms use docs-only CI."""

from __future__ import annotations

import os
from pathlib import PurePosixPath
import re
import subprocess


_SHA = re.compile(r"[0-9a-f]{40}\Z")


def is_documentation(path: str) -> bool:
    if path.startswith(".github/ISSUE_TEMPLATE/"):
        return PurePosixPath(path).suffix.lower() in {".yml", ".yaml"}
    return PurePosixPath(path).suffix.lower() in {".md", ".mdx", ".markdown"}


def docs_only(paths: list[str]) -> bool:
    return bool(paths) and all(is_documentation(path) for path in paths)


def changed_paths(base: str, head: str) -> list[str]:
    if not _SHA.fullmatch(base) or not _SHA.fullmatch(head):
        raise ValueError("PR base and head must be full commit SHAs")
    result = subprocess.run(
        ["git", "diff", "--name-only", "--no-renames", "-z", base, head, "--"],
        check=True,
        capture_output=True,
    )
    return [os.fsdecode(p) for p in result.stdout.split(b"\0") if p]


def main() -> None:
    base = os.environ.get("BASE_SHA", "")
    head = os.environ.get("HEAD_SHA", "")
    paths = changed_paths(base, head) if base else []
    print("docs_only=" + ("true" if docs_only(paths) else "false"))


if __name__ == "__main__":
    main()
