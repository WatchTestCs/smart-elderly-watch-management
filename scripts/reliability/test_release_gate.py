import copy
import importlib.util
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location("release_gate", Path(__file__).with_name("release-gate.py"))
gate = importlib.util.module_from_spec(spec)
spec.loader.exec_module(gate)
SHA = "a"*40


class ReleaseGateTests(unittest.TestCase):
    def fixture(self):
        return {
            "branch": SHA,
            "runs": [{"id": i, "head_sha": SHA, "path": path, "status": "completed", "conclusion": "success", "html_url": "https://github.com/example/run"}
                     for i,path in enumerate((".github/workflows/ci.yml", ".github/workflows/security.yml"), 1)],
            "jobs": [{"name": "Release gate / required", "conclusion": "success"}],
            "artifacts": [{"id": i, "name": prefix+SHA, "expired": False, "digest": "sha256:"+"b"*64, "size_in_bytes": 42}
                          for i,prefix in enumerate(("backend-1.26.10-", "backend-1.27.2-", "frontend-", "watch-", "windows-sidecar-", "security-"),1)],
        }

    def api(self, data):
        def get(path):
            if "/branches/" in path:
                return {"commit": {"sha": data["branch"]}}
            if "/jobs?" in path:
                return {"jobs": data["jobs"]}
            if "/artifacts?" in path:
                return {"artifacts": data["artifacts"]}
            return {"workflow_runs": data["runs"]}
        return get

    def test_accepts_exact_source_evidence(self):
        result = gate.verify("owner/repo", "preview", SHA, self.api(self.fixture()))
        self.assertEqual(result["source_sha"], SHA)
        self.assertEqual(len(result["artifacts"]), 6)

    def test_watch_protocol_evidence_is_required(self):
        data=self.fixture()
        data["artifacts"]=[item for item in data["artifacts"] if not item["name"].startswith("watch-")]
        with self.assertRaises(ValueError):
            gate.verify("owner/repo", "preview", SHA, self.api(data))

    def test_rejects_stale_source_missing_runs_and_partial_gate(self):
        for field,value in (("branch", "c"*40), ("runs", []), ("jobs", []), ("artifacts", [])):
            with self.subTest(field=field):
                data=self.fixture(); data[field]=value
                with self.assertRaises(ValueError):
                    gate.verify("owner/repo", "preview", SHA, self.api(data))

    def test_latest_attempt_cannot_reuse_previous_green(self):
        for status,conclusion in (("in_progress",None),("completed","failure"),("completed","cancelled")):
            data=self.fixture()
            newer=copy.deepcopy(data["runs"][0]); newer.update(id=99,status=status,conclusion=conclusion)
            data["runs"].append(newer)
            with self.assertRaises(ValueError):
                gate.verify("owner/repo", "preview", SHA, self.api(data))

    def test_rejects_expired_or_unhashed_artifact(self):
        for update in ({"expired":True},{"digest":None},{"name":"wrong-source"}):
            data=self.fixture(); data["artifacts"][0].update(update)
            with self.assertRaises(ValueError):
                gate.verify("owner/repo", "preview", SHA, self.api(data))

    def test_branch_move_during_verification_fails(self):
        data=self.fixture(); original=self.api(data); count=0
        def api(path):
            nonlocal count
            if "/branches/" in path:
                count+=1
                if count>1: data["branch"]="c"*40
            return original(path)
        with self.assertRaises(ValueError):
            gate.verify("owner/repo", "preview", SHA, api)


if __name__ == "__main__":
    unittest.main()
