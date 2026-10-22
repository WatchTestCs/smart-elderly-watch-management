"""Run: python -m unittest scripts/test_verify_health_scale.py"""
import importlib.util
import pathlib
import unittest

script = pathlib.Path(__file__).with_name('verify-health-scale.py')
spec = importlib.util.spec_from_file_location('scale_verifier', script)
verifier = importlib.util.module_from_spec(spec)
spec.loader.exec_module(verifier)


class ScaleVerificationTests(unittest.TestCase):
    def setUp(self):
        self.report = {'mode':'health', 'run_id':'abcdefgh1234', 'errors':0,
                       'acknowledged_samples':200000, 'elapsed_seconds':1.0}
        self.evidence = {'run_id':'abcdefgh1234', 'scope':'recordId prefix abcdefgh1234-',
                         'persisted_samples':200000,'persisted_changes':200000}

    def test_target_reached_when_db_counts_match(self):
        self.assertTrue(verifier.verify(self.report,self.evidence,200000)['passed'])

    def test_rejects_missing_database_rows(self):
        self.evidence['persisted_samples'] -= 1
        with self.assertRaises(ValueError):verifier.verify(self.report,self.evidence,200000)

    def test_rejects_wrong_run(self):
        self.evidence['run_id'] = 'other'
        with self.assertRaises(ValueError):verifier.verify(self.report,self.evidence,200000)

    def test_does_not_count_http_rate_as_durable_rate(self):
        self.evidence['persisted_changes'] = 0
        with self.assertRaises(ValueError):verifier.verify(self.report,self.evidence,200000)

    def test_rejects_unscoped_database_evidence(self):
        self.evidence['scope'] = 'all records'
        with self.assertRaises(ValueError):verifier.verify(self.report,self.evidence,200000)

    def test_fails_below_target(self):
        self.report['elapsed_seconds'] = 2
        self.assertFalse(verifier.verify(self.report,self.evidence,200000)['passed'])

    def test_rejects_errors(self):
        self.report['errors'] = 1
        with self.assertRaises(ValueError):verifier.verify(self.report,self.evidence,200000)

if __name__ == '__main__':unittest.main()
