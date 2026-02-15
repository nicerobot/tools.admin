"""Validate generated YAML against the official safe-settings JSON schema.

The schema is pinned to safe-settings 2.1.18 and stored at
schema/safe-settings-2.1.18.json. When upgrading safe-settings,
download the new dereferenced schema and update SCHEMA_PATH.
"""

import json
from pathlib import Path

import jsonschema
import yaml

from admin_tools.models.github import GitHubRepository
from admin_tools.models.overrides import RepoOverrideFile, RepositoryOverrides
from admin_tools.models.settings import RepositoryDefaults
from admin_tools.services.settings_io import write_repo_override
from admin_tools.util.diff import compute_overrides

SCHEMA_PATH = Path(__file__).parent.parent / "schema" / "safe-settings-2.1.18.json"


def _load_schema() -> dict[str, object]:
    with open(SCHEMA_PATH) as f:
        return json.load(f)  # type: ignore[no-any-return]


def _extract_repo_schema(schema: dict[str, object]) -> dict[str, object]:
    """Extract the repositories sub-schema.

    Merge allOf into a single properties dict.
    """
    repos = schema["properties"]["repositories"]  # type: ignore[index]
    all_props: dict[str, object] = {}
    for item in repos.get("allOf", []):  # type: ignore[union-attr]
        all_props.update(item.get("properties", {}))  # type: ignore[union-attr]
    return {
        "type": "object",
        "properties": all_props,
        "additionalProperties": False,
    }


def _validate_repo_section(
    repo_dict: dict[str, object],
    repo_schema: dict[str, object],
) -> list[str]:
    """Validate a repository section against the schema. Returns list of errors."""
    validator = jsonschema.Draft202012Validator(repo_schema)
    return [e.message for e in validator.iter_errors(repo_dict)]


class TestSchemaExists:
    def test_schema_file_present(self) -> None:
        assert SCHEMA_PATH.exists(), f"Schema not found at {SCHEMA_PATH}"

    def test_schema_is_valid_json(self) -> None:
        schema = _load_schema()
        assert "properties" in schema
        assert "repositories" in schema["properties"]  # type: ignore[operator]


class TestOverrideFieldsInSchema:
    """Every RepositoryOverrides field must exist in the schema."""

    def setup_method(self) -> None:
        self.schema = _load_schema()
        self.repo_schema = _extract_repo_schema(self.schema)
        self.schema_fields = set(
            self.repo_schema["properties"].keys()  # type: ignore[union-attr]
        )

    def test_all_override_fields_in_schema(self) -> None:
        """Check that every field we generate is recognized by safe-settings.

        has_discussions is a GitHub API field not in the safe-settings schema.
        Safe-settings passes unknown fields through to the API, but we document
        this as an intentional extension.
        """
        override_fields = set(RepositoryOverrides.model_fields.keys())
        not_in_schema = override_fields - self.schema_fields
        # has_discussions is a known extension not in the safe-settings schema
        # but supported by the GitHub API and passed through by safe-settings
        expected_extensions = {"has_discussions"}
        unexpected = not_in_schema - expected_extensions
        assert unexpected == set(), f"Fields not in safe-settings schema: {unexpected}"

    def test_known_extensions_documented(self) -> None:
        """Ensure has_discussions is the ONLY extension we use."""
        override_fields = set(RepositoryOverrides.model_fields.keys())
        not_in_schema = override_fields - self.schema_fields
        assert not_in_schema == {"has_discussions"}, (
            f"Extension fields changed: {not_in_schema}. "
            "Update this test if intentionally adding new extensions."
        )


class TestGeneratedYamlValidatesAgainstSchema:
    """Validate that YAML from write_repo_override() is schema-compliant."""

    def setup_method(self) -> None:
        self.schema = _load_schema()
        self.repo_schema = _extract_repo_schema(self.schema)
        # Allow has_discussions as an extension
        props = dict(self.repo_schema["properties"])  # type: ignore[arg-type]
        props["has_discussions"] = {"type": "boolean"}
        self.repo_schema = {
            "type": "object",
            "properties": props,
            "additionalProperties": False,
        }
        self.defaults = RepositoryDefaults(
            default_branch="main",
            visibility="private",
            has_issues=False,
            has_projects=False,
            has_wiki=False,
            has_discussions=False,
            is_template=False,
            allow_squash_merge=True,
            allow_merge_commit=True,
            allow_rebase_merge=True,
            allow_auto_merge=False,
            delete_branch_on_merge=True,
        )

    def _generate_and_validate(
        self, repo: GitHubRepository, tmp_path: Path
    ) -> list[str]:
        override = compute_overrides(repo, self.defaults, "testowner", "account")
        outfile = write_repo_override(override, tmp_path / "repos")
        parsed = yaml.safe_load(outfile.read_text())
        repo_section = parsed.get("repository", {})
        if isinstance(repo_section, dict):
            return _validate_repo_section(repo_section, self.repo_schema)
        return []

    def test_empty_overrides_valid(self, tmp_path: Path) -> None:
        repo = GitHubRepository(
            name="boring",
            private=True,
            default_branch="main",
            has_issues=False,
            has_projects=False,
            has_wiki=False,
            has_discussions=False,
            is_template=False,
            allow_squash_merge=True,
            allow_merge_commit=True,
            allow_rebase_merge=True,
            allow_auto_merge=False,
            delete_branch_on_merge=True,
        )
        errors = self._generate_and_validate(repo, tmp_path)
        assert errors == []

    def test_description_override_valid(self, tmp_path: Path) -> None:
        repo = GitHubRepository(
            name="described",
            description="A test repo",
            private=True,
        )
        errors = self._generate_and_validate(repo, tmp_path)
        assert errors == []

    def test_homepage_override_valid(self, tmp_path: Path) -> None:
        repo = GitHubRepository(
            name="with-home",
            homepage="https://example.com",
            private=True,
        )
        errors = self._generate_and_validate(repo, tmp_path)
        assert errors == []

    def test_visibility_override_valid(self, tmp_path: Path) -> None:
        repo = GitHubRepository(name="public", private=False)
        errors = self._generate_and_validate(repo, tmp_path)
        assert errors == []

    def test_boolean_overrides_valid(self, tmp_path: Path) -> None:
        repo = GitHubRepository(
            name="bools",
            private=True,
            has_issues=True,
            has_projects=True,
            has_wiki=True,
            has_discussions=True,
            allow_squash_merge=False,
            allow_merge_commit=False,
            allow_rebase_merge=False,
            allow_auto_merge=True,
            delete_branch_on_merge=False,
        )
        errors = self._generate_and_validate(repo, tmp_path)
        assert errors == []

    def test_archived_override_valid(self, tmp_path: Path) -> None:
        repo = GitHubRepository(name="old", private=True, archived=True)
        errors = self._generate_and_validate(repo, tmp_path)
        assert errors == []

    def test_default_branch_override_valid(self, tmp_path: Path) -> None:
        repo = GitHubRepository(name="branched", private=True, default_branch="master")
        errors = self._generate_and_validate(repo, tmp_path)
        assert errors == []

    def test_full_override_valid(self, tmp_path: Path) -> None:
        """A repo with every possible override field set."""
        repo = GitHubRepository(
            name="full",
            description="Full override test",
            homepage="https://example.com",
            private=False,
            default_branch="develop",
            has_issues=True,
            has_projects=True,
            has_wiki=True,
            has_discussions=True,
            is_template=True,
            allow_squash_merge=False,
            allow_merge_commit=False,
            allow_rebase_merge=False,
            allow_auto_merge=True,
            delete_branch_on_merge=False,
            archived=True,
            fork=True,
        )
        errors = self._generate_and_validate(repo, tmp_path)
        assert errors == []

    def test_real_admin_repo_valid(self, tmp_path: Path) -> None:
        """The real admin.yml override format must validate."""
        override = RepoOverrideFile(
            owner="nicerobot",
            name="admin",
            comment_source="account",
            repository=RepositoryOverrides(
                default_branch="main",
                has_issues=True,
                has_projects=True,
                has_wiki=True,
                delete_branch_on_merge=False,
            ),
        )
        outfile = write_repo_override(override, tmp_path / "repos")
        parsed = yaml.safe_load(outfile.read_text())
        errors = _validate_repo_section(parsed["repository"], self.repo_schema)
        assert errors == []

    def test_real_nicerobot_profile_valid(self, tmp_path: Path) -> None:
        override = RepoOverrideFile(
            owner="nicerobot",
            name="nicerobot",
            comment_source="account",
            repository=RepositoryOverrides(
                description="About nicerobot",
                visibility="public",
                default_branch="master",
            ),
        )
        outfile = write_repo_override(override, tmp_path / "repos")
        parsed = yaml.safe_load(outfile.read_text())
        errors = _validate_repo_section(parsed["repository"], self.repo_schema)
        assert errors == []
