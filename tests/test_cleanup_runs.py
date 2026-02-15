from unittest.mock import AsyncMock, call, patch

from admin_tools.commands.cleanup_runs import run_cleanup_runs
from admin_tools.models.github import GitHubRepository, WorkflowRun


def _repo(name: str) -> GitHubRepository:
    return GitHubRepository.model_validate({
        "name": name,
        "private": False,
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
    })


def _run(
    run_id: int,
    workflow_id: int = 1,
    created_at: str = "2025-01-01T00:00:00Z",
    name: str = "CI",
) -> WorkflowRun:
    return WorkflowRun(
        id=run_id,
        name=name,
        status="completed",
        conclusion="success",
        created_at=created_at,
        workflow_id=workflow_id,
    )


def _mock_client(
    repos: list[GitHubRepository] | None = None,
    runs_by_repo: dict[str, list[WorkflowRun]] | None = None,
) -> AsyncMock:
    client = AsyncMock()
    client.list_repos = AsyncMock(return_value=repos or [])

    async def _list_runs(
        owner: str,
        repo: str,
        *,
        created_before: str | None = None,
    ) -> list[WorkflowRun]:
        if runs_by_repo and repo in runs_by_repo:
            return runs_by_repo[repo]
        return []

    client.list_workflow_runs = AsyncMock(side_effect=_list_runs)
    client.delete_workflow_run = AsyncMock()
    client.__aenter__ = AsyncMock(return_value=client)
    client.__aexit__ = AsyncMock(return_value=None)
    return client


class TestCleanupRuns:
    def test_deletes_old_runs(self) -> None:
        runs = [
            _run(1, created_at="2025-01-01T00:00:00Z"),
            _run(2, created_at="2025-01-02T00:00:00Z"),
            _run(3, created_at="2025-01-03T00:00:00Z"),
        ]
        mock = _mock_client(runs_by_repo={"repo1": runs})
        with patch(
            "admin_tools.commands.cleanup_runs.GitHubClient",
            return_value=mock,
        ):
            run_cleanup_runs("nicerobot", "repo1", days=30, keep=0,
                             dry_run=False)

        assert mock.delete_workflow_run.call_count == 3

    def test_keeps_minimum_per_workflow(self) -> None:
        runs = [
            _run(1, workflow_id=10, created_at="2025-01-01T00:00:00Z"),
            _run(2, workflow_id=10, created_at="2025-01-02T00:00:00Z"),
            _run(3, workflow_id=10, created_at="2025-01-03T00:00:00Z"),
            _run(4, workflow_id=20, created_at="2025-01-01T00:00:00Z"),
            _run(5, workflow_id=20, created_at="2025-01-02T00:00:00Z"),
        ]
        mock = _mock_client(runs_by_repo={"repo1": runs})
        with patch(
            "admin_tools.commands.cleanup_runs.GitHubClient",
            return_value=mock,
        ):
            run_cleanup_runs("nicerobot", "repo1", days=30, keep=2,
                             dry_run=False)

        # workflow 10: keep 2, delete 1  |  workflow 20: keep 2, delete 0
        assert mock.delete_workflow_run.call_count == 1
        mock.delete_workflow_run.assert_called_once_with(
            "nicerobot", "repo1", 1,
        )

    def test_dry_run_does_not_delete(self) -> None:
        runs = [
            _run(1, created_at="2025-01-01T00:00:00Z"),
            _run(2, created_at="2025-01-02T00:00:00Z"),
        ]
        mock = _mock_client(runs_by_repo={"repo1": runs})
        with patch(
            "admin_tools.commands.cleanup_runs.GitHubClient",
            return_value=mock,
        ):
            run_cleanup_runs("nicerobot", "repo1", days=30, keep=0,
                             dry_run=True)

        mock.delete_workflow_run.assert_not_called()

    def test_single_repo_mode(self) -> None:
        runs = [_run(1)]
        mock = _mock_client(runs_by_repo={"target": runs})
        with patch(
            "admin_tools.commands.cleanup_runs.GitHubClient",
            return_value=mock,
        ):
            run_cleanup_runs("nicerobot", "target", days=30, keep=0,
                             dry_run=False)

        mock.list_repos.assert_not_called()
        mock.delete_workflow_run.assert_called_once_with(
            "nicerobot", "target", 1,
        )

    def test_all_repos_mode(self) -> None:
        repos = [_repo("repo1"), _repo("repo2")]
        runs_map = {
            "repo1": [_run(1)],
            "repo2": [_run(2)],
        }
        mock = _mock_client(repos=repos, runs_by_repo=runs_map)
        with patch(
            "admin_tools.commands.cleanup_runs.GitHubClient",
            return_value=mock,
        ):
            run_cleanup_runs("nicerobot", None, days=30, keep=0,
                             dry_run=False)

        mock.list_repos.assert_called_once_with("nicerobot")
        assert mock.delete_workflow_run.call_count == 2
        mock.delete_workflow_run.assert_has_calls([
            call("nicerobot", "repo1", 1),
            call("nicerobot", "repo2", 2),
        ])

    def test_no_old_runs_nothing_deleted(self) -> None:
        mock = _mock_client(runs_by_repo={"repo1": []})
        with patch(
            "admin_tools.commands.cleanup_runs.GitHubClient",
            return_value=mock,
        ):
            run_cleanup_runs("nicerobot", "repo1", days=30, keep=5,
                             dry_run=False)

        mock.delete_workflow_run.assert_not_called()

    def test_empty_repo_skipped(self) -> None:
        repos = [_repo("empty")]
        mock = _mock_client(repos=repos, runs_by_repo={})
        with patch(
            "admin_tools.commands.cleanup_runs.GitHubClient",
            return_value=mock,
        ):
            run_cleanup_runs("nicerobot", None, days=30, keep=5,
                             dry_run=False)

        mock.delete_workflow_run.assert_not_called()
