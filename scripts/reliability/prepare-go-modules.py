#!/usr/bin/env python3
"""Prepare checksum-locked modules; retry transport failures, never tests."""
import argparse
import hashlib
from pathlib import Path
import re
import subprocess
import time


TRANSIENT = re.compile(
    r"stream error:.*INTERNAL_ERROR|TLS handshake timeout|connection reset|"
    r"unexpected EOF|i/o timeout|context deadline exceeded|download timed out|"
    r"connectex:.*connection attempt failed|"
    r"(?:502 Bad Gateway|503 Service Unavailable|504 Gateway Timeout)", re.I
)
PERMANENT = re.compile(
    r"checksum mismatch|SECURITY ERROR|unknown revision|invalid version|"
    r"(?:401 Unauthorized|403 Forbidden|404 Not Found)", re.I
)


def prepare(backend, output, run=subprocess.run, sleep=time.sleep):
    backend, output = Path(backend).resolve(), Path(output).resolve()
    locked = {name: hashlib.sha256((backend / name).read_bytes()).digest()
              for name in ("go.mod", "go.sum")}
    output.mkdir(parents=True, exist_ok=True)

    def command(args, log):
        try:
            result = run(args, cwd=backend, capture_output=True, text=True,
                         encoding="utf-8", errors="replace", timeout=180, check=False)
            code, text = result.returncode, result.stdout + result.stderr
        except subprocess.TimeoutExpired as error:
            text = "".join(value.decode("utf-8", errors="replace") if isinstance(value, bytes)
                           else value or "" for value in (error.stdout, error.stderr))
            code, text = 124, text + "\ndownload timed out\n"
        with (output / log).open("x", encoding="utf-8") as file:
            file.write(text)
        if any(hashlib.sha256((backend / name).read_bytes()).digest() != digest
               for name, digest in locked.items()):
            raise RuntimeError("Go preparation changed a lock file; review the dependency change")
        return code, text

    # Download code pinned by go.mod, then resolve the full inventory metadata.
    # Neither phase executes tests or fetches source for dependencies' own tests.
    for phase, args in (("module-download", ["go", "mod", "download"]),
                        ("module-inventory", ["go", "list", "-m", "-json", "all"])):
        for attempt in range(1, 4):
            code, text = command(args, f"{phase}-attempt-{attempt}.txt")
            if code == 0:
                break
            print(f"{phase} failed on attempt {attempt}; original output is preserved", flush=True)
            if PERMANENT.search(text) or not TRANSIENT.search(text) or attempt == 3:
                raise RuntimeError("Go module preparation failed; tests have not run")
            sleep(attempt * 2)
    code, _ = command(["go", "mod", "verify"], "module-verify.txt")
    if code:
        raise RuntimeError("Go module integrity verification failed; tests have not run")
    print("Checksum-locked Go modules ready; no tests were retried", flush=True)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--backend", required=True)
    parser.add_argument("--output", required=True)
    args = parser.parse_args()
    prepare(args.backend, args.output)
