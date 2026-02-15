
from admin_tools.models.github import GitHubRepository, GitHubUser
from admin_tools.models.overrides import RepoOverrideFile, RepositoryOverrides
from admin_tools.models.settings import (
    Collaborator,
    Label,
    OrgSettings,
    RepositoryDefaults,
)


class TestRepositoryDefaults:
    def test_defaults(self) -> None:
        d = RepositoryDefaults()
        assert d.default_branch == "main"
        assert d.visibility == "private"
        assert d.has_issues is False
        assert d.has_projects is False
        assert d.has_wiki is False
        assert d.has_discussions is False
        assert d.is_template is False
        assert d.allow_squash_merge is True
        assert d.allow_merge_commit is True
        assert d.allow_rebase_merge is True
        assert d.allow_auto_merge is False
        assert d.delete_branch_on_merge is True

    def test_custom_values(self) -> None:
        d = RepositoryDefaults(default_branch="develop", visibility="public")
        assert d.default_branch == "develop"
        assert d.visibility == "public"


class TestLabel:
    def test_label(self) -> None:
        label = Label(name="bug", color="d73a4a", description="Something isn't working")
        assert label.name == "bug"
        assert label.color == "d73a4a"

    def test_label_default_description(self) -> None:
        label = Label(name="bug", color="d73a4a")
        assert label.description == ""


class TestCollaborator:
    def test_collaborator(self) -> None:
        c = Collaborator(username="nicerobot", permission="admin")
        assert c.username == "nicerobot"
        assert c.permission == "admin"

    def test_default_permission(self) -> None:
        c = Collaborator(username="someone")
        assert c.permission == "push"


class TestOrgSettings:
    def test_defaults(self) -> None:
        s = OrgSettings()
        assert isinstance(s.repository, RepositoryDefaults)
        assert s.labels == []
        assert s.collaborators == []

    def test_full_settings(self) -> None:
        s = OrgSettings(
            repository=RepositoryDefaults(visibility="public"),
            labels=[Label(name="bug", color="d73a4a")],
            collaborators=[Collaborator(username="user1")],
        )
        assert s.repository.visibility == "public"
        assert len(s.labels) == 1
        assert len(s.collaborators) == 1

    def test_from_yaml_dict(self) -> None:
        data = {
            "repository": {
                "default_branch": "main",
                "visibility": "private",
                "has_issues": False,
            },
            "labels": [{"name": "bug", "color": "d73a4a"}],
            "collaborators": [{"username": "nicerobot", "permission": "admin"}],
        }
        s = OrgSettings.model_validate(data)
        assert s.repository.default_branch == "main"
        assert s.repository.has_issues is False
        assert s.labels[0].name == "bug"
        assert s.collaborators[0].username == "nicerobot"


class TestGitHubUser:
    def test_user(self) -> None:
        u = GitHubUser(login="nicerobot", type="User")
        assert u.login == "nicerobot"
        assert u.type == "User"

    def test_org(self) -> None:
        u = GitHubUser(login="myorg", type="Organization")
        assert u.type == "Organization"


class TestGitHubRepository:
    def test_public_repo(self) -> None:
        r = GitHubRepository(name="test", private=False)
        assert r.visibility == "public"

    def test_private_repo(self) -> None:
        r = GitHubRepository(name="test", private=True)
        assert r.visibility == "private"

    def test_defaults(self) -> None:
        r = GitHubRepository(name="test")
        assert r.description is None
        assert r.homepage is None
        assert r.private is False
        assert r.default_branch == "main"
        assert r.has_issues is True
        assert r.has_projects is True
        assert r.has_wiki is True
        assert r.has_discussions is False
        assert r.is_template is False
        assert r.allow_squash_merge is True
        assert r.allow_merge_commit is True
        assert r.allow_rebase_merge is True
        assert r.allow_auto_merge is False
        assert r.delete_branch_on_merge is False
        assert r.archived is False
        assert r.fork is False

    def test_from_api_response(self) -> None:
        data = {
            "name": "admin",
            "description": "Admin repo",
            "homepage": None,
            "private": True,
            "default_branch": "main",
            "has_issues": True,
            "has_projects": True,
            "has_wiki": True,
            "has_discussions": False,
            "is_template": False,
            "allow_squash_merge": True,
            "allow_merge_commit": True,
            "allow_rebase_merge": True,
            "allow_auto_merge": False,
            "delete_branch_on_merge": False,
            "archived": False,
            "fork": False,
        }
        r = GitHubRepository.model_validate(data)
        assert r.name == "admin"
        assert r.visibility == "private"
        assert r.description == "Admin repo"

    def test_fork_repo(self) -> None:
        r = GitHubRepository(name="forked", fork=True)
        assert r.fork is True

    def test_archived_repo(self) -> None:
        r = GitHubRepository(name="old", archived=True)
        assert r.archived is True

    def test_api_response_extra_fields_ignored(self) -> None:
        """GitHub API returns many extra fields; Pydantic should ignore them."""
        data = {
            "name": "test",
            "id": 12345,
            "full_name": "owner/test",
            "html_url": "https://github.com/owner/test",
            "private": False,
            "owner": {"login": "owner"},
            "default_branch": "main",
        }
        r = GitHubRepository.model_validate(data)
        assert r.name == "test"


class TestRepositoryOverrides:
    def test_all_none(self) -> None:
        o = RepositoryOverrides()
        assert o.model_dump(exclude_none=True) == {}

    def test_some_overrides(self) -> None:
        o = RepositoryOverrides(description="test", has_issues=True)
        dumped = o.model_dump(exclude_none=True)
        assert dumped == {"description": "test", "has_issues": True}

    def test_all_fields_settable(self) -> None:
        o = RepositoryOverrides(
            description="desc",
            homepage="https://example.com",
            visibility="public",
            default_branch="master",
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
        )
        dumped = o.model_dump(exclude_none=True)
        assert len(dumped) == 15


class TestRepoOverrideFile:
    def test_basic(self) -> None:
        f = RepoOverrideFile(owner="nicerobot", name="admin", comment_source="account")
        assert f.owner == "nicerobot"
        assert f.name == "admin"
        assert f.comment_source == "account"
        assert f.is_fork is False
        assert f.repository == RepositoryOverrides()

    def test_fork(self) -> None:
        f = RepoOverrideFile(
            owner="nicerobot", name="forked",
            comment_source="account", is_fork=True,
        )
        assert f.is_fork is True
