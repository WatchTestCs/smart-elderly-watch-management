"""Fail-closed scope classification regression tests."""

import unittest

from pr_change_scope import docs_only, is_documentation


class ChangeScopeTests(unittest.TestCase):
    def test_markdown_and_issue_forms_can_use_fast_lane(self):
        self.assertTrue(
            docs_only(
                [
                    "docs/ARCHITECTURE.md",
                    "README.en.md",
                    ".github/ISSUE_TEMPLATE/bug_report.yml",
                ]
            )
        )

    def test_empty_change_is_not_docs_only(self):
        self.assertFalse(docs_only([]))

    def test_runnable_yaml_and_source_force_full_validation(self):
        for path in [
            ".github/workflows/ci.yml",
            "backend/go.mod",
            "frontend/src/router.ts",
            "desktop/Cargo.toml",
            "scripts/reliability/pr_change_scope.py",
            "deploy/platform.compose.yml",
            ".github/ISSUE_TEMPLATE/README.sh",
        ]:
            with self.subTest(path=path):
                self.assertFalse(docs_only(["README.md", path]))

    def test_issue_templates_restrict_extension(self):
        self.assertTrue(is_documentation(".github/ISSUE_TEMPLATE/config.yml"))
        self.assertFalse(is_documentation(".github/ISSUE_TEMPLATE/deploy.ps1"))


if __name__ == "__main__":
    unittest.main()
