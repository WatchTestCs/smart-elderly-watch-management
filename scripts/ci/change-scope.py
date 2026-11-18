"""Select PR checks from changed paths. Pushes and uncertain changes run everything."""

import json
import os
import subprocess

CHECKS = ("frontend", "ui", "backend", "watch", "simulator", "windows", "deployment", "security")
PREFIX = "智慧养老手表管理系统/"
FRONTEND = PREFIX + "frontend/"
BACKEND = PREFIX + "backend/"
DESKTOP = PREFIX + "desktop/"
ROOT_DOCS = {"README.md", "README.en.md", "CONTRIBUTING.md"}


def select(paths):
    selected = set()
    for path in paths:
        if path in ROOT_DOCS or (path.startswith("docs/") and path.endswith(".md")):
            continue

        # Any non-documentation change can affect a dependency manifest.
        selected.add("security")
        if path.startswith(FRONTEND):
            selected.update(("frontend", "ui", "windows"))
        elif path.startswith(BACKEND):
            selected.update(("backend", "windows", "watch"))
        elif path.startswith(DESKTOP):
            selected.update(("windows", "frontend", "backend"))
        elif path.startswith("SuiningWatch/"):
            selected.add("watch")
        elif path.startswith(PREFIX + "watch-sim/"):
            selected.add("simulator")
        elif path.startswith("deploy/") or path == ".dockerignore":
            selected.add("deployment")
        elif path.startswith("docs/screenshots/"):
            selected.add("ui")
        else:
            return set(CHECKS)
    return selected


def main():
    selected = set(CHECKS)
    if os.environ["EVENT_NAME"] == "pull_request":
        comparison = f'{os.environ["BASE_SHA"]}...{os.environ["HEAD_SHA"]}'
        changes = subprocess.check_output(
            ["git", "diff", "--name-status", "--no-renames", "-z", comparison, "--"]
        )
        fields = changes.decode("utf-8").split("\0")[:-1]
        statuses, paths = fields[::2], fields[1::2]
        if paths and len(statuses) == len(paths) and set(statuses) <= {"A", "M"}:
            selected = select(paths)
        print(f"Selected checks for {len(paths)} changed paths: {sorted(selected)}")

    with open(os.environ["GITHUB_OUTPUT"], "a", encoding="utf-8") as file:
        for check in CHECKS:
            file.write(f"{check}={str(check in selected).lower()}\n")
        file.write(f"full={str(selected == set(CHECKS)).lower()}\n")
        matrix = []
        if "frontend" in selected or "ui" in selected:
            matrix.append({"language": "javascript-typescript", "build-mode": "none"})
        if "backend" in selected:
            matrix.append({"language": "go", "build-mode": "manual"})
        file.write(f"codeql={str(bool(matrix)).lower()}\n")
        file.write("codeql_matrix=" + json.dumps({"include": matrix}, separators=(",", ":")) + "\n")


if __name__ == "__main__":
    main()
