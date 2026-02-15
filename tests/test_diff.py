from admin_tools.models.github import GitHubRepository
from admin_tools.models.settings import RepositoryDefaults
from admin_tools.util.diff import compute_overrides


class TestComputeOverrides:
    """Test every possible override field individually and in combination."""

    def setup_method(self) -> None:
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

    def test_no_overrides_when_matching_defaults(self) -> None:
        """Repo matching all defaults produces empty overrides."""
        repo = GitHubRepository(
            name="boring",
            description=None,
            homepage=None,
            private=True,  # visibility=private matches default
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
            archived=False,
            fork=False,
        )
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        overrides = result.repository.model_dump(exclude_none=True)
        assert overrides == {}
        assert result.is_fork is False

    def test_description_included_when_nonempty(self) -> None:
        repo = GitHubRepository(
            name="test", description="About nicerobot",
            private=True,
        )
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        assert result.repository.description == "About nicerobot"

    def test_description_excluded_when_empty(self) -> None:
        repo = GitHubRepository(name="test", description="", private=True)
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        assert result.repository.description is None

    def test_description_excluded_when_none(self) -> None:
        repo = GitHubRepository(name="test", description=None, private=True)
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        assert result.repository.description is None

    def test_homepage_included_when_nonempty(self) -> None:
        repo = GitHubRepository(
            name="test", homepage="https://example.com",
            private=True,
        )
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        assert result.repository.homepage == "https://example.com"

    def test_homepage_excluded_when_empty(self) -> None:
        repo = GitHubRepository(name="test", homepage="", private=True)
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        assert result.repository.homepage is None

    def test_homepage_excluded_when_none(self) -> None:
        repo = GitHubRepository(name="test", homepage=None, private=True)
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        assert result.repository.homepage is None

    def test_visibility_override_public(self) -> None:
        repo = GitHubRepository(name="test", private=False)
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        assert result.repository.visibility == "public"

    def test_visibility_matches_default_not_included(self) -> None:
        repo = GitHubRepository(name="test", private=True)
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        assert result.repository.visibility is None

    def test_default_branch_override(self) -> None:
        repo = GitHubRepository(name="test", private=True, default_branch="master")
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        assert result.repository.default_branch == "master"

    def test_default_branch_matches_not_included(self) -> None:
        repo = GitHubRepository(name="test", private=True, default_branch="main")
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        assert result.repository.default_branch is None

    def test_has_issues_override(self) -> None:
        repo = GitHubRepository(name="test", private=True, has_issues=True)
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        assert result.repository.has_issues is True

    def test_has_issues_matches_not_included(self) -> None:
        repo = GitHubRepository(name="test", private=True, has_issues=False)
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        assert result.repository.has_issues is None

    def test_has_projects_override(self) -> None:
        repo = GitHubRepository(name="test", private=True, has_projects=True)
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        assert result.repository.has_projects is True

    def test_has_wiki_override(self) -> None:
        repo = GitHubRepository(name="test", private=True, has_wiki=True)
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        assert result.repository.has_wiki is True

    def test_has_discussions_override(self) -> None:
        repo = GitHubRepository(name="test", private=True, has_discussions=True)
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        assert result.repository.has_discussions is True

    def test_is_template_override(self) -> None:
        repo = GitHubRepository(name="test", private=True, is_template=True)
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        assert result.repository.is_template is True

    def test_allow_squash_merge_override(self) -> None:
        repo = GitHubRepository(name="test", private=True, allow_squash_merge=False)
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        assert result.repository.allow_squash_merge is False

    def test_allow_merge_commit_override(self) -> None:
        repo = GitHubRepository(name="test", private=True, allow_merge_commit=False)
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        assert result.repository.allow_merge_commit is False

    def test_allow_rebase_merge_override(self) -> None:
        repo = GitHubRepository(name="test", private=True, allow_rebase_merge=False)
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        assert result.repository.allow_rebase_merge is False

    def test_allow_auto_merge_override(self) -> None:
        repo = GitHubRepository(name="test", private=True, allow_auto_merge=True)
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        assert result.repository.allow_auto_merge is True

    def test_delete_branch_on_merge_override(self) -> None:
        repo = GitHubRepository(name="test", private=True, delete_branch_on_merge=False)
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        assert result.repository.delete_branch_on_merge is False

    def test_archived_true_included(self) -> None:
        repo = GitHubRepository(name="test", private=True, archived=True)
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        assert result.repository.archived is True

    def test_archived_false_not_included(self) -> None:
        repo = GitHubRepository(name="test", private=True, archived=False)
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        assert result.repository.archived is None

    def test_fork_flag(self) -> None:
        repo = GitHubRepository(name="forked", private=True, fork=True)
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        assert result.is_fork is True

    def test_not_fork(self) -> None:
        repo = GitHubRepository(name="owned", private=True, fork=False)
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        assert result.is_fork is False

    def test_comment_source_org(self) -> None:
        repo = GitHubRepository(name="test", private=True)
        result = compute_overrides(repo, self.defaults, "myorg", "org")
        assert result.comment_source == "org"

    def test_comment_source_account(self) -> None:
        repo = GitHubRepository(name="test", private=True)
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        assert result.comment_source == "account"

    def test_owner_and_name_preserved(self) -> None:
        repo = GitHubRepository(name="admin", private=True)
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        assert result.owner == "nicerobot"
        assert result.name == "admin"

    def test_multiple_overrides(self) -> None:
        """Repo like the real admin repo: several fields differ from defaults."""
        repo = GitHubRepository(
            name="admin",
            private=True,
            default_branch="main",
            has_issues=True,
            has_projects=True,
            has_wiki=True,
            delete_branch_on_merge=False,
        )
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        overrides = result.repository.model_dump(exclude_none=True)
        assert overrides == {
            "has_issues": True,
            "has_projects": True,
            "has_wiki": True,
            "delete_branch_on_merge": False,
        }

    def test_real_nicerobot_profile_repo(self) -> None:
        """Matches the real nicerobot/nicerobot override file.

        Must set all fields to match what the GitHub API actually returns
        so only the expected overrides appear.
        """
        repo = GitHubRepository(
            name="nicerobot",
            description="About nicerobot",
            private=False,
            default_branch="master",
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
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        overrides = result.repository.model_dump(exclude_none=True)
        assert overrides == {
            "description": "About nicerobot",
            "visibility": "public",
            "default_branch": "master",
        }

    def test_real_site_repo(self) -> None:
        """Matches site-nicerobot.com override file."""
        repo = GitHubRepository(
            name="site-nicerobot.com",
            homepage="http://nicerobot.com",
            private=True,
            default_branch="master",
            has_issues=False,
            has_projects=False,
            has_wiki=False,
            has_discussions=False,
            is_template=False,
            allow_squash_merge=True,
            allow_merge_commit=False,
            allow_rebase_merge=False,
            allow_auto_merge=False,
            delete_branch_on_merge=True,
        )
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        overrides = result.repository.model_dump(exclude_none=True)
        assert overrides == {
            "homepage": "http://nicerobot.com",
            "default_branch": "master",
            "allow_merge_commit": False,
            "allow_rebase_merge": False,
        }

    def test_field_ordering_matches_override_model(self) -> None:
        """Ensure model_dump preserves field declaration order."""
        repo = GitHubRepository(
            name="test",
            description="desc",
            homepage="https://example.com",
            private=False,
            default_branch="master",
            has_issues=True,
            has_projects=False,
            has_wiki=False,
            has_discussions=False,
            is_template=False,
            allow_squash_merge=True,
            allow_merge_commit=True,
            allow_rebase_merge=True,
            allow_auto_merge=False,
            delete_branch_on_merge=True,
            archived=True,
        )
        result = compute_overrides(repo, self.defaults, "nicerobot", "account")
        overrides = result.repository.model_dump(exclude_none=True)
        keys = list(overrides.keys())
        assert keys == [
            "description", "homepage", "visibility",
            "default_branch", "has_issues", "archived",
        ]
