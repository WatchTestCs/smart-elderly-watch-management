#!/usr/bin/env python3
"""Do not treat govulncheck JSON-mode exit zero as no reachable findings."""
import argparse
import json
from pathlib import Path


def audit(path):
    text = Path(path).read_text(encoding="utf-8")
    decoder = json.JSONDecoder()
    events, offset = [], 0
    while offset < len(text):
        if text[offset].isspace():
            offset += 1
            continue
        event, offset = decoder.raw_decode(text, offset)
        events.append(event)
    configs = [e["config"] for e in events if "config" in e]
    if len(configs) != 1 or configs[0].get("scan_level") != "symbol" or configs[0].get("scan_mode") != "source":
        raise ValueError("one complete symbol-level source scan configuration is required")
    if not any("progress" in e for e in events):
        raise ValueError("scan progress is missing")
    findings = [e["finding"] for e in events if "finding" in e]
    reachable = [f for f in findings if f.get("trace") and f["trace"][0].get("function")]
    return {"accepted": not reachable, "config": configs[0], "reachable": reachable,
            "module_or_package_findings": [f for f in findings if f not in reachable]}


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("log")
    parser.add_argument("--output", required=True)
    args = parser.parse_args()
    try:
        result = audit(args.log)
    except (ValueError, OSError) as error:
        result = {"accepted": False, "error": str(error)}
    Path(args.output).write_text(json.dumps(result, indent=2)+"\n", encoding="utf-8")
    print(json.dumps({"accepted": result["accepted"], "reachable": len(result.get("reachable", [])), "error": result.get("error")}))
    raise SystemExit(0 if result["accepted"] else 1)
