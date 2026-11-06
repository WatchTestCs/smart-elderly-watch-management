"""Repeat named watch-sim component cases in a reproducible shuffled order."""
import argparse
import hashlib
import json
import math
from pathlib import Path
import random
import subprocess


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=Path, required=True)
    parser.add_argument("--cases", required=True, help="Comma-separated case names")
    parser.add_argument("--repeats", type=int, default=5)
    parser.add_argument("--seconds", type=float, default=0.5)
    parser.add_argument("--seed", type=int, default=20261102)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    if args.repeats < 1 or not math.isfinite(args.seconds) or args.seconds <= 0:
        parser.error("repeats and seconds must be positive")
    cases = args.cases.split(",")
    if not all(cases) or len(set(cases)) != len(cases):
        parser.error("cases must be nonempty and unique")
    binary = args.binary.resolve(strict=True)
    randomizer = random.Random(args.seed)
    rows = []
    args.output.parent.mkdir(parents=True, exist_ok=True)
    for repeat in range(args.repeats):
        order = cases[:]
        randomizer.shuffle(order)
        for case in order:
            row = json.loads(subprocess.check_output(
                [str(binary), str(args.seconds), case], text=True))
            if row["case"] != case or row["iterations"] < 1:
                raise ValueError(f"invalid measurement for {case}")
            rows.append({**row, "repeat": repeat})
            args.output.write_text(json.dumps(rows, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({"binary_sha256": hashlib.sha256(binary.read_bytes()).hexdigest(),
                      "cases": cases, "repeats": args.repeats,
                      "seconds": args.seconds, "seed": args.seed,
                      "measurements": len(rows)}, indent=2))


if __name__ == "__main__":
    main()
