#!/usr/bin/env python3
"""Prepare the product compiler and the pinned JSON v2 self-copy backport.

The installed toolchain is read-only. Product builds and tests consume the same
Go overlay. Downloaded toolchains need an owned copy outside GOMODCACHE because
the go command rejects overlays beneath that cache.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess

BEFORE = "6978277353d629e9fa02a3a351df55d647a067918c33fff6617ccaf307656fc8"
AFTER = "70ccd0db45093bed636d00a015ed26f165c0400b2a3b0f0e64ccfd35e99cc456"
RELATIVE = Path("src/encoding/json/jsontext/decode.go")
OLD = "\t\tn := copy(d.buf[:cap(d.buf)], d.buf[d.prevStart:])\n\t\td.buf = d.buf[:n]"
NEW = "\t\tif d.prevStart != 0 {\n\t\t\tn := copy(d.buf[:cap(d.buf)], d.buf[d.prevStart:])\n\t\t\td.buf = d.buf[:n]\n\t\t}"


def digest(data):
    return hashlib.sha256(data).hexdigest()


def go_quote(argument):
    # cmd/internal/quoted.Split does not unescape quoted fields.
    if not any(c in argument for c in " \t\r\n'\""):
        return argument
    for quote in ("'", '"'):
        if quote not in argument:
            return quote + argument + quote
    raise ValueError("Go flags cannot represent a path containing both quote types")


def prepare(args):
    cfg = json.loads(subprocess.check_output([args.go, "env", "-json", "GOVERSION", "GOROOT", "GOMODCACHE", "GOFLAGS", "GOEXPERIMENT"]).decode("utf-8"))
    match = re.fullmatch(r"go1\.(\d+)\.(\d+)", cfg["GOVERSION"])
    if not match:
        raise ValueError("a released Go compiler is required")
    version = tuple(map(int, match.groups()))
    if args.compatibility:
        if version not in ((26, 10), (27, 2), (27, 3)):
            raise ValueError("unverified compatibility compiler")
    elif version < (27, 3):
        raise ValueError("product artifacts require Go 1.27.3 or newer")
    root = Path(cfg["GOROOT"]).resolve()
    if (root/"VERSION").read_text(encoding="utf-8").splitlines()[0] != "go1." + ".".join(map(str, version)):
        raise ValueError("GOROOT and compiler versions do not match")
    output = Path(args.output).resolve()
    if output.is_relative_to(root):
        raise ValueError("output must not be inside the installed toolchain")
    if "-overlay" in cfg["GOFLAGS"]:
        raise ValueError("pass caller overlays with --overlay so the JSON backport cannot be replaced")
    output.mkdir(parents=True, exist_ok=True)
    replacement = {}
    metadata = {"compiler": cfg["GOVERSION"], "compatibility": args.compatibility, "installed_goroot": str(root), "toolchain_source_modified": False}
    if version >= (27, 0):
        original = (root/RELATIVE).read_bytes().replace(b"\r\n", b"\n")
        if digest(original) not in (BEFORE, AFTER):
            raise ValueError("unreviewed JSON source; review the backport before changing compiler versions")
        if root.is_relative_to(Path(cfg["GOMODCACHE"]).resolve()):
            owned = output/"toolchain"
            if not owned.resolve().is_relative_to(output):
                raise ValueError("owned toolchain path escapes output")
            identity = {"source": str(root), "version": cfg["GOVERSION"]}
            owner_path = output/"toolchain-owner.json"
            if owner_path.exists():
                if json.loads(owner_path.read_text(encoding="utf-8")) != identity:
                    raise ValueError("output belongs to a different toolchain")
            else:
                if owned.exists():
                    raise ValueError("refusing to adopt an unowned toolchain directory")
                owner_path.write_text(json.dumps(identity, indent=2), encoding="utf-8")
            complete = output/"toolchain-copy-complete"
            if not complete.exists():
                shutil.copytree(root, owned, dirs_exist_ok=True)
                complete.write_text(cfg["GOVERSION"], encoding="utf-8")
            for relative in (RELATIVE, Path("bin/go.exe" if os.name == "nt" else "bin/go"), Path("src/crypto/x509/verify.go")):
                if digest((root/relative).read_bytes()) != digest((owned/relative).read_bytes()):
                    raise ValueError("owned toolchain copy identity mismatch")
            root = owned
        patched = original.decode("utf-8")
        if digest(original) == BEFORE:
            if patched.count(OLD) != 1:
                raise ValueError("JSON backport anchor is not unique")
            patched = patched.replace(OLD, NEW)
        data = patched.encode("utf-8")
        if digest(data) != AFTER:
            raise ValueError("JSON backport result hash mismatch")
        source = output/"jsontext-decode.go"
        source.write_bytes(data)
        replacement[str(root/RELATIVE)] = str(source)
        metadata.update(json_before_sha256=digest(original), json_after_sha256=digest(data), json_engine=2)
    else:
        metadata["json_engine"] = 1
    if args.overlay:
        caller = json.loads(Path(args.overlay).read_text(encoding="utf-8-sig"))["Replace"]
        if set(caller) & set(replacement):
            raise ValueError("caller overlay would replace the managed JSON backport")
        replacement.update(caller)
    flags = cfg["GOFLAGS"].strip()
    if replacement:
        overlay = output/"overlay.json"
        overlay.write_text(json.dumps({"Replace": replacement}), encoding="utf-8")
        flags = (flags + " " + go_quote("-overlay="+str(overlay))).strip()
        metadata["overlay_sha256"] = digest(overlay.read_bytes())
    experiments = [x for x in cfg["GOEXPERIMENT"].split(",") if x and x not in ("jsonv2", "nojsonv2")]
    experiments.append("jsonv2" if version >= (27, 0) else "nojsonv2")
    env = {"GOROOT": str(root), "GOFLAGS": flags, "GOEXPERIMENT": ",".join(experiments)}
    if any("\n" in value or "\r" in value for value in env.values()):
        raise ValueError("multiline build environment is not supported")
    metadata["environment"] = env
    metadata["crypto_verify_sha256"] = digest((root/"src/crypto/x509/verify.go").read_bytes())
    manifest = output/"manifest.json"
    manifest.write_text(json.dumps(metadata, indent=2)+"\n", encoding="utf-8")
    license_data = (root/"LICENSE").read_bytes()
    license_path = output/"GO-LICENSE"
    if license_path.exists():
        if license_path.read_bytes() != license_data:
            raise ValueError("output license does not match the selected toolchain")
    else:
        license_path.write_bytes(license_data)
    if args.github_env:
        with Path(args.github_env).open("a", encoding="utf-8") as stream:
            for key, value in env.items():
                stream.write(key+"="+value+"\n")
    return env, str(manifest)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--go", default="go")
    parser.add_argument("--output", required=True)
    parser.add_argument("--compatibility", action="store_true")
    parser.add_argument("--overlay")
    parser.add_argument("--github-env")
    parser.add_argument("--run", nargs=argparse.REMAINDER)
    args = parser.parse_args()
    environment, manifest = prepare(args)
    if args.run:
        raise SystemExit(subprocess.run([args.go, *args.run], env={**os.environ, **environment}).returncode)
    else:
        print(json.dumps({"environment": environment, "manifest": manifest}))
