"""Compare signing builds on a verified Windows processor affinity mask."""
import argparse
import ctypes
from ctypes import wintypes
import hashlib
import json
import math
from pathlib import Path
import random
import subprocess
import sys


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--before", type=Path, required=True)
    parser.add_argument("--after", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--mode", choices=["component", "http"], default="component")
    parser.add_argument("--mask", type=lambda value: int(value, 0), default=4)
    parser.add_argument("--seconds", type=float, default=0.5)
    parser.add_argument("--repeats", type=int, default=5)
    args = parser.parse_args()
    if sys.platform != "win32":
        parser.error("this comparison uses Windows process affinity")
    if args.mask <= 0 or args.repeats < 1 or not math.isfinite(args.seconds) or args.seconds <= 0:
        parser.error("mask, repeats and seconds must be positive")
    kernel = ctypes.WinDLL("kernel32", use_last_error=True)
    kernel.GetCurrentProcess.restype = wintypes.HANDLE
    kernel.SetProcessAffinityMask.argtypes = [wintypes.HANDLE, ctypes.c_size_t]
    kernel.GetProcessAffinityMask.argtypes = [wintypes.HANDLE,
                                             ctypes.POINTER(ctypes.c_size_t),
                                             ctypes.POINTER(ctypes.c_size_t)]
    if not kernel.SetProcessAffinityMask(kernel.GetCurrentProcess(), args.mask):
        raise ctypes.WinError(ctypes.get_last_error())
    binaries = {"baseline": args.before.resolve(strict=True),
                "prepared": args.after.resolve(strict=True)}
    if args.mode == "component":
        jobs = [(build, f"signature_{size}{suffix}")
                for size in [512, 65536]
                for build, suffix in [("baseline", ""), ("prepared", ""),
                                      ("prepared", "_prepared")]]
    else:
        binaries["control"] = binaries["baseline"]
        jobs = [(build, case) for build in binaries
                for case in ["http_report", "http_report_signed"]]
    rows = []
    randomizer = random.Random(20261102)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    for repeat in range(args.repeats):
        order = jobs[:]
        randomizer.shuffle(order)
        for build, case in order:
            with subprocess.Popen([str(binaries[build]), str(args.seconds), case],
                                  stdout=subprocess.PIPE, text=True) as process:
                affinity, system = ctypes.c_size_t(), ctypes.c_size_t()
                if not kernel.GetProcessAffinityMask(wintypes.HANDLE(int(process._handle)),
                                                     ctypes.byref(affinity), ctypes.byref(system)):
                    raise ctypes.WinError(ctypes.get_last_error())
                if affinity.value != args.mask:
                    raise RuntimeError(f"child affinity is {affinity.value}, expected {args.mask}")
                stdout, _ = process.communicate()
                if process.returncode:
                    raise subprocess.CalledProcessError(process.returncode, process.args)
            row = json.loads(stdout)
            if row["case"] != case:
                raise ValueError(f"unexpected measurement: {row['case']}")
            rows.append({**row, "build": build, "repeat": repeat, "affinity_mask": args.mask})
            args.output.write_text(json.dumps(rows, indent=2) + "\n", encoding="utf-8", newline="\n")
    print(json.dumps({"measurements": len(rows), "affinity_mask": args.mask,
                      "sha256": {name: hashlib.sha256(path.read_bytes()).hexdigest()
                                 for name, path in binaries.items()}}, indent=2))


if __name__ == "__main__":
    main()
