"""Select PR checks from changed paths. Pushes and uncertain changes run everything."""

import json
import os
import subprocess


CHECKS = ("frontend", "ui", "backend", "watch", "simulator", "windows", "deployment", "security")
PREFIX = "智慧养老手表管理系统/"
FRONTEND = PREFIX + "frontend/"
BACKEND = PREFIX + "backend/"
DESKTOP = PREFIX + "desktop/"


def select(paths):
    selected = set()
    for path in paths:
        if path.endswith((".md", ".mdx")):
            continue
        if path.startswith(FRONTEND):
            selected.update(("frontend", "ui", "windows"))
        elif path.startswith(BACKEND):
            selected.update(("backend", "windows", "watch", "security"))
        elif path.startswith(DESKTOP):
            selected.update(("windows", "frontend", "backend"))
        elif path.startswith("SuiningWatch/"):
            selected.add("watch")
        elif path.startswith(PREFIX + "watch-sim/"):
            selected.add("simulator")
        elif path.startswith(("deploy/", ".dockerignore")):
            selected.add("deployment")
        elif path.startswith("docs/screenshots/"):
            selected.add("ui")
        else:
            return set(CHECKS)
    return selected


def main():
    full = os.environ["EVENT_NAME"] != "pull_request"
    if full:
        selected = set(CHECKS)
    else:
        changes = subprocess.check_output(
            ["git", "diff", "--name-only", "-z", os.environ["BASE_SHA"], os.environ["HEAD_SHA"]]
        )
        paths = [p.decode("utf-8") for p in changes.split(b"\0") if p]
        selected = select(paths) if paths else set(CHECKS)
        print(f"Selected checks for {len(paths)} changed paths: {sorted(selected)}")
    output = os.environ["GITHUB_OUTPUT"]
    with open(output, "a", encoding="utf-8") as file:
        for check in CHECKS:
            file.write(f"{check}={str(check in selected).lower()}\n")
        file.write(f"full={str(selected == set(CHECKS)).lower()}\n")
        # CodeQL only supports Go and JS/TS. Other isolated modules need no CodeQL matrix.
        matrix = []
        if "frontend" in selected or "ui" in selected:
            matrix.append({"language": "javascript-typescript", "build-mode": "none"})
        if "backend" in selected or "security" in selected:
            matrix.append({"language": "go", "build-mode": "manual"})
        file.write(f"codeql={str(bool(matrix)).lower()}\n")
        file.write("codeql_matrix=" + json.dumps({"include": matrix}, separators=(",", ":")) + "\n")


if __name__ == "__main__":
    main()
