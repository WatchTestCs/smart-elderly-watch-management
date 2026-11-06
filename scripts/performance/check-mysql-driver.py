"""Verify the bounded MySQL driver fork and scan its pinned upstream package."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import tempfile


def source_hash(path):
    return hashlib.sha256(path.read_bytes().replace(b"\r\n", b"\n")).hexdigest()


def verify_files(fork, metadata):
    expected = metadata["fork_files"]
    actual = {p.relative_to(fork).as_posix() for p in fork.rglob("*")
              if p.is_file() and p != fork / "UPSTREAM.json"}
    if actual != set(expected):
        raise ValueError("MySQL fork file inventory differs from UPSTREAM.json")
    if not set(metadata["upstream_files"]).issubset(expected):
        raise ValueError("MySQL fork is missing upstream files")
    for name, digest in expected.items():
        if source_hash(fork / name) != digest:
            raise ValueError(f"MySQL fork fingerprint mismatch: {name}")
    modified = {name for name, digest in metadata["upstream_files"].items()
                if expected.get(name) != digest}
    if modified != {"connection.go", "utils.go"}:
        raise ValueError("MySQL fork changes exceed the reviewed parameter scratch patch")
    return {"files": len(expected), "modified_upstream_files": sorted(modified)}


def module_info(directory):
    raw = subprocess.check_output(
        ["go", "list", "-m", "-json", "github.com/go-sql-driver/mysql"],
        cwd=directory, text=True, encoding="utf-8")
    return json.loads(raw), raw


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--evidence", required=True, type=Path)
    args = parser.parse_args()
    evidence = args.evidence.resolve()
    evidence.mkdir(parents=True, exist_ok=True)
    repo = Path(__file__).resolve().parents[2]
    backend = repo / "智慧养老手表管理系统" / "backend"
    fork = backend / "third_party" / "mysql"
    metadata = json.loads((fork / "UPSTREAM.json").read_text(encoding="utf-8"))
    integrity = verify_files(fork, metadata)
    module, raw = module_info(backend)
    if (module["Version"] != metadata["version"]
            or Path(module["Replace"]["Dir"]).resolve() != fork.resolve()):
        raise ValueError("Backend does not resolve the reviewed MySQL driver fork")
    (evidence / "backend-module.json").write_text(raw, encoding="utf-8")
    (evidence / "integrity.json").write_text(json.dumps(integrity, indent=2), encoding="utf-8")

    # A local replace changes govulncheck's module identity. Keep the ordinary
    # current-source scan and independently scan the real upstream package at
    # its authenticated version, conservatively including all package symbols.
    with tempfile.TemporaryDirectory(prefix="mysql-driver-upstream-") as directory:
        audit = Path(directory)
        (audit / "go.mod").write_text(
            "module mysql-driver-upstream-audit\n\ngo 1.26.0\n\n"
            f"require github.com/go-sql-driver/mysql {metadata['version']}\n", encoding="utf-8")
        (audit / "go.sum").write_bytes((backend / "go.sum").read_bytes())
        (audit / "main.go").write_text(
            'package main\nimport _ "github.com/go-sql-driver/mysql"\nfunc main() {}\n',
            encoding="utf-8")
        subprocess.run(["go", "mod", "tidy"], cwd=audit, check=True)
        upstream, raw = module_info(audit)
        if (upstream["Version"] != metadata["version"] or upstream.get("Replace")
                or upstream["Sum"] != metadata["sum"]
                or upstream["GoModSum"] != metadata["go_mod_sum"]):
            raise ValueError("Upstream MySQL vulnerability scan lost the pinned source identity")
        original = {p.name: source_hash(p) for p in Path(upstream["Dir"]).iterdir() if p.is_file()}
        if original != metadata["upstream_files"]:
            raise ValueError("MySQL fork provenance does not match the authenticated upstream source")
        (evidence / "upstream-module.json").write_text(raw, encoding="utf-8")
        subprocess.run(["go", "mod", "verify"], cwd=audit, check=True)
        with (evidence / "upstream-vulnerability-scan.jsonl").open("w", encoding="utf-8") as output:
            subprocess.run(["govulncheck", "-json", "-scan=package", "./..."], cwd=audit,
                           stdout=output, stderr=subprocess.STDOUT, check=True)
    print("MySQL fork integrity and pinned upstream package vulnerability scan passed")


if __name__ == "__main__":
    main()
