#!/usr/bin/env python3
"""Verify current branch SHA, latest CI runs and immutable artifact digests."""
import argparse
import datetime
import json
import os
import re
import urllib.parse
import urllib.request
from pathlib import Path


def get_api(path):
    request = urllib.request.Request("https://api.github.com/"+path, headers={
        "Accept": "application/vnd.github+json", "X-GitHub-Api-Version": "2022-11-28",
        "User-Agent": "suining-exact-source-release-gate",
    })
    token = os.getenv("GITHUB_TOKEN")
    if token:
        request.add_header("Authorization", "Bearer "+token)
    with urllib.request.urlopen(request, timeout=30) as response:
        return json.load(response)


def verify(repo, branch, sha, api=get_api):
    if not re.fullmatch(r"[0-9a-f]{40}", sha):
        raise ValueError("an explicit full lowercase commit SHA is required")
    base = "repos/"+repo
    ref = base+"/branches/"+urllib.parse.quote(branch, safe="")
    if api(ref)["commit"]["sha"] != sha:
        raise ValueError("branch head differs from requested release source")
    # Query all event types. The PR-only connector helper cannot prove that a
    # push-triggered run is absent or successful.
    runs = api(base+"/actions/runs?head_sha="+sha+"&per_page=100")["workflow_runs"]
    selected = []
    for workflow in (".github/workflows/ci.yml", ".github/workflows/security.yml"):
        matching = [run for run in runs if run["head_sha"] == sha and run["path"] == workflow]
        if not matching:
            raise ValueError("no exact-source run: "+workflow)
        run = max(matching, key=lambda item: (item["id"], item.get("run_attempt", 1)))
        if run["status"] != "completed" or run["conclusion"] != "success":
            raise ValueError("latest exact-source workflow is not successful: "+workflow)
        selected.append(run)
    ci = selected[0]
    jobs = api(base+"/actions/runs/"+str(ci["id"])+"/jobs?per_page=100")["jobs"]
    if not any(job["name"] == "Release gate / required" and job["conclusion"] == "success" for job in jobs):
        raise ValueError("current release aggregator did not pass (historical CI is insufficient)")
    artifacts = api(base+"/actions/runs/"+str(ci["id"])+"/artifacts?per_page=100")["artifacts"]
    required = ["backend-1.26.10-", "backend-1.27.2-", "frontend-", "windows-sidecar-", "security-"]
    evidence = []
    for prefix in required:
        matches = [a for a in artifacts if a["name"] == prefix+sha and not a["expired"]]
        if len(matches) != 1 or not re.fullmatch(r"sha256:[0-9a-f]{64}", matches[0].get("digest", "")):
            raise ValueError("missing, ambiguous, expired or unhashed evidence artifact: "+prefix+sha)
        evidence.append({key: matches[0][key] for key in ("id", "name", "digest", "size_in_bytes")})
    if api(ref)["commit"]["sha"] != sha:
        raise ValueError("branch moved during verification; rerun on the new head")
    return {"source_sha": sha, "branch": branch, "repository": repo,
            "verified_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
            "workflows": [{key: r[key] for key in ("id", "head_sha", "html_url", "conclusion")} for r in selected],
            "artifacts": evidence,
            "scope": "Build/test evidence only. Human/provider/hardware/DR and repository protection remain independent production gates."}


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo", required=True)
    parser.add_argument("--branch", default="preview")
    parser.add_argument("--sha", required=True)
    parser.add_argument("--output", required=True)
    args = parser.parse_args()
    result = verify(args.repo, args.branch, args.sha)
    Path(args.output).write_text(json.dumps(result, ensure_ascii=False, indent=2)+"\n", encoding="utf-8")
    print("Verified exact source and evidence: "+args.sha)
