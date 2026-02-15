from pathlib import Path

import pytest

from admin_tools.models.overrides import RepoOverrideFile, RepositoryOverrides
from admin_tools.models.settings import OrgSettings
from admin_tools.services.settings_io import (
    list_existing_repo_files,
    load_org_settings,
    write_repo_override,
)


class TestLoadOrgSettings:
    def test_loads_real_format(self, settings_dir: Path) -> None:
        settings = load_org_settings(settings_dir)
        assert settings.repository.default_branch == "main"
        assert settings.repository.visibility == "private"
        assert settings.repository.has_issues is False
        assert settings.repository.has_projects is False
        assert settings.repository.has_wiki is False
        assert settings.repository.has_discussions is False
        assert settings.repository.is_template is False
        assert settings.repository.allow_squash_merge is True
        assert settings.repository.allow_merge_commit is True
        assert settings.repository.allow_rebase_merge is True
        assert settings.repository.allow_auto_merge is False
        assert settings.repository.delete_branch_on_merge is True

    def test_loads_labels(self, settings_dir: Path) -> None:
        settings = load_org_settings(settings_dir)
        assert len(settings.labels) == 1
        assert settings.labels[0].name == "bug"
        assert settings.labels[0].color == "d73a4a"

    def test_loads_collaborators(self, settings_dir: Path) -> None:
        settings = load_org_settings(settings_dir)
        assert len(settings.collaborators) == 1
        assert settings.collaborators[0].username == "nicerobot"
        assert settings.collaborators[0].permission == "admin"

    def test_missing_file_raises(self, tmp_path: Path) -> None:
        with pytest.raises(FileNotFoundError):
            load_org_settings(tmp_path / "nonexistent")

    def test_empty_settings_file(self, tmp_path: Path) -> None:
        settings_dir = tmp_path / ".github"
        settings_dir.mkdir()
        (settings_dir / "settings.yml").write_text("")
        settings = load_org_settings(settings_dir)
        assert isinstance(settings, OrgSettings)
        assert settings.repository.default_branch == "main"

    def test_minimal_settings_file(self, tmp_path: Path) -> None:
        settings_dir = tmp_path / ".github"
        settings_dir.mkdir()
        (settings_dir / "settings.yml").write_text(
            "repository:\n  visibility: public\n"
        )
        settings = load_org_settings(settings_dir)
        assert settings.repository.visibility == "public"
        assert settings.repository.default_branch == "main"  # default


class TestWriteRepoOverride:
    def test_empty_overrides(self, tmp_path: Path) -> None:
        repos_dir = tmp_path / "repos"
        override = RepoOverrideFile(
            owner="nicerobot",
            name="boring",
            comment_source="account",
        )
        outfile = write_repo_override(override, repos_dir)
        content = outfile.read_text()
        assert content == (
            "# nicerobot/boring — overrides from account defaults\n"
            "\n"
            "repository: {}\n"
        )

    def test_with_overrides(self, tmp_path: Path) -> None:
        repos_dir = tmp_path / "repos"
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
        outfile = write_repo_override(override, repos_dir)
        content = outfile.read_text()
        assert content == (
            "# nicerobot/admin — overrides from account defaults\n"
            "\n"
            "repository:\n"
            "  default_branch: main\n"
            "  has_issues: true\n"
            "  has_projects: true\n"
            "  has_wiki: true\n"
            "  delete_branch_on_merge: false\n"
        )

    def test_description_double_quoted(self, tmp_path: Path) -> None:
        repos_dir = tmp_path / "repos"
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
        outfile = write_repo_override(override, repos_dir)
        content = outfile.read_text()
        assert '  description: "About nicerobot"' in content
        assert "  visibility: public\n" in content
        assert "  default_branch: master\n" in content

    def test_homepage_double_quoted(self, tmp_path: Path) -> None:
        repos_dir = tmp_path / "repos"
        override = RepoOverrideFile(
            owner="nicerobot",
            name="year",
            comment_source="account",
            repository=RepositoryOverrides(
                homepage="https://nicerobot.github.io/year?i=2024",
            ),
        )
        outfile = write_repo_override(override, repos_dir)
        content = outfile.read_text()
        assert '  homepage: "https://nicerobot.github.io/year?i=2024"' in content

    def test_fork_flag(self, tmp_path: Path) -> None:
        repos_dir = tmp_path / "repos"
        override = RepoOverrideFile(
            owner="nicerobot",
            name="forked",
            comment_source="account",
            is_fork=True,
        )
        outfile = write_repo_override(override, repos_dir)
        content = outfile.read_text()
        assert content == (
            "# nicerobot/forked — overrides from account defaults\n"
            "\n"
            "_fork: true\n"
            "\n"
            "repository: {}\n"
        )

    def test_fork_with_overrides(self, tmp_path: Path) -> None:
        repos_dir = tmp_path / "repos"
        override = RepoOverrideFile(
            owner="nicerobot",
            name="forked",
            comment_source="account",
            is_fork=True,
            repository=RepositoryOverrides(has_issues=True),
        )
        outfile = write_repo_override(override, repos_dir)
        content = outfile.read_text()
        lines = content.split("\n")
        assert lines[0] == "# nicerobot/forked — overrides from account defaults"
        assert lines[1] == ""
        assert lines[2] == "_fork: true"
        assert lines[3] == ""
        assert lines[4] == "repository:"
        assert lines[5] == "  has_issues: true"

    def test_org_comment_source(self, tmp_path: Path) -> None:
        repos_dir = tmp_path / "repos"
        override = RepoOverrideFile(
            owner="myorg",
            name="project",
            comment_source="org",
        )
        outfile = write_repo_override(override, repos_dir)
        content = outfile.read_text()
        assert content.startswith("# myorg/project — overrides from org defaults\n")

    def test_creates_repos_dir(self, tmp_path: Path) -> None:
        repos_dir = tmp_path / "nested" / "repos"
        assert not repos_dir.exists()
        override = RepoOverrideFile(owner="o", name="r", comment_source="account")
        write_repo_override(override, repos_dir)
        assert repos_dir.exists()

    def test_overwrites_existing_file(self, tmp_path: Path) -> None:
        repos_dir = tmp_path / "repos"
        repos_dir.mkdir()
        (repos_dir / "test.yml").write_text("old content")
        override = RepoOverrideFile(owner="o", name="test", comment_source="account")
        outfile = write_repo_override(override, repos_dir)
        assert "old content" not in outfile.read_text()
        assert "# o/test" in outfile.read_text()

    def test_booleans_bare_true_false(self, tmp_path: Path) -> None:
        repos_dir = tmp_path / "repos"
        override = RepoOverrideFile(
            owner="o",
            name="r",
            comment_source="account",
            repository=RepositoryOverrides(
                has_issues=True,
                allow_squash_merge=False,
            ),
        )
        outfile = write_repo_override(override, repos_dir)
        content = outfile.read_text()
        assert "  has_issues: true\n" in content
        assert "  allow_squash_merge: false\n" in content
        # booleans should NOT be quoted
        assert '"true"' not in content
        assert '"false"' not in content

    def test_archived_true(self, tmp_path: Path) -> None:
        repos_dir = tmp_path / "repos"
        override = RepoOverrideFile(
            owner="o",
            name="old",
            comment_source="account",
            repository=RepositoryOverrides(archived=True),
        )
        outfile = write_repo_override(override, repos_dir)
        content = outfile.read_text()
        assert "  archived: true\n" in content

    def test_file_ends_with_newline(self, tmp_path: Path) -> None:
        repos_dir = tmp_path / "repos"
        override = RepoOverrideFile(owner="o", name="r", comment_source="account")
        outfile = write_repo_override(override, repos_dir)
        content = outfile.read_text()
        assert content.endswith("\n")
        assert not content.endswith("\n\n")

    def test_no_trailing_document_marker(self, tmp_path: Path) -> None:
        repos_dir = tmp_path / "repos"
        override = RepoOverrideFile(
            owner="o",
            name="r",
            comment_source="account",
            repository=RepositoryOverrides(has_issues=True),
        )
        outfile = write_repo_override(override, repos_dir)
        content = outfile.read_text()
        assert "---" not in content

    def test_real_admin_repo_format(self, tmp_path: Path) -> None:
        """Should produce output identical to the real admin.yml file."""
        repos_dir = tmp_path / "repos"
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
        outfile = write_repo_override(override, repos_dir)
        expected = (
            "# nicerobot/admin — overrides from account defaults\n"
            "\n"
            "repository:\n"
            "  default_branch: main\n"
            "  has_issues: true\n"
            "  has_projects: true\n"
            "  has_wiki: true\n"
            "  delete_branch_on_merge: false\n"
        )
        assert outfile.read_text() == expected


class TestListExistingRepoFiles:
    def test_empty_dir(self, tmp_path: Path) -> None:
        repos_dir = tmp_path / "repos"
        repos_dir.mkdir()
        assert list_existing_repo_files(repos_dir) == set()

    def test_nonexistent_dir(self, tmp_path: Path) -> None:
        assert list_existing_repo_files(tmp_path / "nope") == set()

    def test_finds_yml_files(self, tmp_path: Path) -> None:
        repos_dir = tmp_path / "repos"
        repos_dir.mkdir()
        (repos_dir / "admin.yml").write_text("test")
        (repos_dir / "site-example.com.yml").write_text("test")
        (repos_dir / "other.txt").write_text("ignored")
        result = list_existing_repo_files(repos_dir)
        assert result == {"admin", "site-example.com"}

    def test_strips_yml_extension(self, tmp_path: Path) -> None:
        repos_dir = tmp_path / "repos"
        repos_dir.mkdir()
        (repos_dir / "my-repo.yml").write_text("")
        result = list_existing_repo_files(repos_dir)
        assert "my-repo" in result
        assert "my-repo.yml" not in result
