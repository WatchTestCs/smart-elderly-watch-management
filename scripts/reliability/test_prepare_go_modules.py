import importlib.util
from pathlib import Path
import subprocess
import tempfile
import unittest

spec = importlib.util.spec_from_file_location("prepare_go", Path(__file__).with_name("prepare-go-modules.py"))
prepare_go = importlib.util.module_from_spec(spec)
spec.loader.exec_module(prepare_go)


class ModulePreparationTests(unittest.TestCase):
    def exercise(self, failures):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        root = Path(temporary.name)
        (root / "go.mod").write_text("module fixture\n")
        (root / "go.sum").write_text("fixture checksum\n")
        calls = []

        def run(args, **kwargs):
            calls.append(args)
            index = len(calls) - 1
            message = failures[index] if index < len(failures) else ""
            return subprocess.CompletedProcess(args, int(bool(message)), "", message)

        return root, calls, run

    def test_network_retry_preserves_failures_and_never_runs_or_retries_tests(self):
        root, calls, run = self.exercise(["stream error: stream ID 41; INTERNAL_ERROR; received from peer"])
        prepare_go.prepare(root, root / "logs", run=run, sleep=lambda _: None)
        self.assertEqual(calls, [["go", "mod", "download"], ["go", "mod", "download"], ["go", "list", "-m", "-json", "all"], ["go", "mod", "verify"]])
        self.assertIn("INTERNAL_ERROR", (root / "logs/module-download-attempt-1.txt").read_text())

    def test_integrity_auth_and_permanent_failures_are_not_retried(self):
        for message in ("SECURITY ERROR checksum mismatch; stream error: INTERNAL_ERROR", "404 Not Found", "401 Unauthorized", "unknown revision", "unexpected local failure"):
            with self.subTest(message=message):
                root, calls, run = self.exercise([message])
                with self.assertRaises(RuntimeError):
                    prepare_go.prepare(root, root / "logs", run=run, sleep=lambda _: None)
                self.assertEqual(len(calls), 1)

    def test_network_attempts_are_bounded(self):
        root, calls, run = self.exercise(["TLS handshake timeout"] * 5)
        with self.assertRaises(RuntimeError):
            prepare_go.prepare(root, root / "logs", run=run, sleep=lambda _: None)
        self.assertEqual(len(calls), 3)

    def test_verification_failure_is_not_retried(self):
        root, calls, run = self.exercise(["", "", "checksum mismatch"])
        with self.assertRaises(RuntimeError):
            prepare_go.prepare(root, root / "logs", run=run, sleep=lambda _: None)
        self.assertEqual(len(calls), 3)

    def test_inventory_transport_failure_is_prepared_before_verification(self):
        root, calls, run = self.exercise(["", "503 Service Unavailable"])
        prepare_go.prepare(root, root / "logs", run=run, sleep=lambda _: None)
        self.assertEqual(calls[1:3], [["go", "list", "-m", "-json", "all"]] * 2)
        self.assertEqual(calls[-1], ["go", "mod", "verify"])

    def test_lockfile_changes_are_rejected(self):
        root, calls, run = self.exercise([])
        def mutate(args, **kwargs):
            result = run(args, **kwargs)
            (root / "go.sum").write_text("unreviewed change\n")
            return result
        with self.assertRaises(RuntimeError):
            prepare_go.prepare(root, root / "logs", run=mutate, sleep=lambda _: None)
        self.assertEqual(len(calls), 1)

    def test_timeout_preserves_partial_output_before_bounded_retry(self):
        root, calls, run = self.exercise([])
        def timeout_once(args, **kwargs):
            if not calls:
                calls.append(args)
                raise subprocess.TimeoutExpired(args, 180, output=b"partial download\n")
            return run(args, **kwargs)
        prepare_go.prepare(root, root / "logs", run=timeout_once, sleep=lambda _: None)
        self.assertEqual(len(calls), 4)
        self.assertIn("partial download", (root / "logs/module-download-attempt-1.txt").read_text())
