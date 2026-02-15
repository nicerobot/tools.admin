from pathlib import Path
from unittest.mock import AsyncMock, patch

import httpx
import pytest

from admin_tools.commands.snapshot import run_snapshot
from admin_tools.models.github import GitHubRepository


def _repo(**kwargs: object) -> GitHubRepository:
    defaults: dict[str, object] = {
        "name": "test",
        "private": True,
        "default_branch": "main",
        "has_issues": False,
        "has_projects": False,
        "has_wiki": False,
        "has_discussions": False,
        "is_template": False,
        "allow_squash_merge": True,
        "allow_merge_commit": True,
        "allow_rebase_merge": True,
        "allow_auto_merge": False,
        "delete_branch_on_merge": True,
        "archived": False,
        "fork": False,
    }
    defaults.update(kwargs)
    return GitHubRepository.model_validate(defaults)


def _mock_client(
    repos: list[GitHubRepository],
    account_type: str = "User",
    *,
    repo_exists_side_effect: list[bool] | None = None,
) -> AsyncMock:
    client = AsyncMock()
    client.get_account_type = AsyncMock(return_value=account_type)
    client.list_repos = AsyncMock(return_value=repos)
    if repo_exists_side_effect is not None:
        client.repo_exists = AsyncMock(side_effect=repo_exists_side_effect)
    else:
        client.repo_exists = AsyncMock(return_value=False)
    client.__aenter__ = AsyncMock(return_value=client)
    client.__aexit__ = AsyncMock(return_value=None)
    return client


class TestRunSnapshot:
    def test_writes_override_files(self, settings_dir: Path) -> None:
        repos = [
            _repo(name="repo1", has_issues=True),
            _repo(name="repo2"),
        ]
        with patch(
            "admin_tools.commands.snapshot.GitHubClient",
            return_value=_mock_client(repos),
        ):
            run_snapshot("nicerobot", settings_dir)

        repos_dir = settings_dir / "repos"
        assert (repos_dir / "repo1.yml").exists()
        assert (repos_dir / "repo2.yml").exists()

        content1 = (repos_dir / "repo1.yml").read_text()
        assert "has_issues: true" in content1

        content2 = (repos_dir / "repo2.yml").read_text()
        assert "repository: {}" in content2

    def test_removes_stale_files_verified_404(
        self, settings_dir: Path,
    ) -> None:
        repos_dir = settings_dir / "repos"
        repos_dir.mkdir()
        (repos_dir / "alive.yml").write_text("old")
        (repos_dir / "dead.yml").write_text("old")

        mock = _mock_client(
            [_repo(name="alive")],
            repo_exists_side_effect=[False],
        )
        with patch(
            "admin_tools.commands.snapshot.GitHubClient",
            return_value=mock,
        ):
            run_snapshot("nicerobot", settings_dir)

        assert (repos_dir / "alive.yml").exists()
        assert not (repos_dir / "dead.yml").exists()
        mock.repo_exists.assert_called_once_with("nicerobot", "dead")

    def test_aborts_when_stale_repo_exists(
        self, settings_dir: Path,
    ) -> None:
        repos_dir = settings_dir / "repos"
        repos_dir.mkdir()
        (repos_dir / "visible.yml").write_text("old content")
        (repos_dir / "hidden.yml").write_text("old content")

        mock = _mock_client(
            [_repo(name="visible")],
            repo_exists_side_effect=[True],
        )
        with (
            patch(
                "admin_tools.commands.snapshot.GitHubClient",
                return_value=mock,
            ),
            pytest.raises(SystemExit, match="1"),
        ):
            run_snapshot("nicerobot", settings_dir)

        # no files should have been modified
        assert (repos_dir / "hidden.yml").read_text() == "old content"
        assert (repos_dir / "visible.yml").read_text() == "old content"

    def test_aborts_on_verification_api_error(
        self, settings_dir: Path,
    ) -> None:
        repos_dir = settings_dir / "repos"
        repos_dir.mkdir()
        (repos_dir / "alive.yml").write_text("old content")
        (repos_dir / "maybe.yml").write_text("old content")

        mock = _mock_client(
            [_repo(name="alive")],
            repo_exists_side_effect=[
                httpx.HTTPStatusError(
                    "Server Error",
                    request=httpx.Request("GET", "https://api.github.com"),
                    response=httpx.Response(500),
                ),
            ],
        )
        with (
            patch(
                "admin_tools.commands.snapshot.GitHubClient",
                return_value=mock,
            ),
            pytest.raises(httpx.HTTPStatusError),
        ):
            run_snapshot("nicerobot", settings_dir)

        # no files should have been modified
        assert (repos_dir / "maybe.yml").read_text() == "old content"
        assert (repos_dir / "alive.yml").read_text() == "old content"

    def test_no_stale_skips_verification(
        self, settings_dir: Path,
    ) -> None:
        repos_dir = settings_dir / "repos"
        repos_dir.mkdir()
        (repos_dir / "repo1.yml").write_text("old")

        mock = _mock_client([_repo(name="repo1")])
        with patch(
            "admin_tools.commands.snapshot.GitHubClient",
            return_value=mock,
        ):
            run_snapshot("nicerobot", settings_dir)

        assert (repos_dir / "repo1.yml").exists()
        mock.repo_exists.assert_not_called()

    def test_no_stale_removal_when_all_exist(
        self, settings_dir: Path,
    ) -> None:
        repos_dir = settings_dir / "repos"
        repos_dir.mkdir()
        (repos_dir / "repo1.yml").write_text("old")

        with patch(
            "admin_tools.commands.snapshot.GitHubClient",
            return_value=_mock_client([_repo(name="repo1")]),
        ):
            run_snapshot("nicerobot", settings_dir)

        assert (repos_dir / "repo1.yml").exists()

    def test_org_comment_source(self, settings_dir: Path) -> None:
        with patch(
            "admin_tools.commands.snapshot.GitHubClient",
            return_value=_mock_client(
                [_repo(name="orgproject")],
                account_type="Organization",
            ),
        ):
            run_snapshot("myorg", settings_dir)

        content = (settings_dir / "repos" / "orgproject.yml").read_text()
        assert "overrides from org defaults" in content

    def test_user_comment_source(self, settings_dir: Path) -> None:
        with patch(
            "admin_tools.commands.snapshot.GitHubClient",
            return_value=_mock_client([_repo(name="myrepo")]),
        ):
            run_snapshot("nicerobot", settings_dir)

        content = (settings_dir / "repos" / "myrepo.yml").read_text()
        assert "overrides from account defaults" in content

    def test_fork_repos_marked(self, settings_dir: Path) -> None:
        with patch(
            "admin_tools.commands.snapshot.GitHubClient",
            return_value=_mock_client(
                [_repo(name="forked", fork=True)],
            ),
        ):
            run_snapshot("nicerobot", settings_dir)

        content = (settings_dir / "repos" / "forked.yml").read_text()
        assert "_fork: true" in content

    def test_creates_repos_dir_if_missing(
        self, settings_dir: Path,
    ) -> None:
        repos_dir = settings_dir / "repos"
        assert not repos_dir.exists()

        with patch(
            "admin_tools.commands.snapshot.GitHubClient",
            return_value=_mock_client([_repo(name="first")]),
        ):
            run_snapshot("nicerobot", settings_dir)

        assert repos_dir.exists()
        assert (repos_dir / "first.yml").exists()

    def test_empty_repos_list_cleans_verified_stale(
        self, settings_dir: Path,
    ) -> None:
        repos_dir = settings_dir / "repos"
        repos_dir.mkdir()
        (repos_dir / "orphan.yml").write_text("old")

        mock = _mock_client([], repo_exists_side_effect=[False])
        with patch(
            "admin_tools.commands.snapshot.GitHubClient",
            return_value=mock,
        ):
            run_snapshot("nicerobot", settings_dir)

        assert not (repos_dir / "orphan.yml").exists()
