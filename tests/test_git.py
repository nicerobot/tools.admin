import subprocess
from unittest.mock import patch

from admin_tools.services.git import (
    checkout_branch,
    commit,
    configure_bot_identity,
    create_pr,
    force_push,
    has_staged_changes,
    pr_exists,
    stage_directory,
)


class TestConfigureBotIdentity:
    @patch("admin_tools.services.git._run")
    def test_sets_name_and_email(self, mock_run: object) -> None:
        configure_bot_identity()
        assert mock_run.call_count == 2  # type: ignore[union-attr]
        mock_run.assert_any_call(  # type: ignore[union-attr]
            ["git", "config", "user.name", "github-actions[bot]"]
        )
        mock_run.assert_any_call(  # type: ignore[union-attr]
            [
                "git",
                "config",
                "user.email",
                "41898282+github-actions[bot]@users.noreply.github.com",
            ]
        )


class TestCheckoutBranch:
    @patch("admin_tools.services.git._run")
    def test_checkout_branch(self, mock_run: object) -> None:
        checkout_branch("safe-settings/snapshot")
        mock_run.assert_called_once_with(  # type: ignore[union-attr]
            ["git", "checkout", "-B", "safe-settings/snapshot"]
        )


class TestStageDirectory:
    @patch("admin_tools.services.git._run")
    def test_stage_directory(self, mock_run: object) -> None:
        stage_directory(".github/repos")
        mock_run.assert_called_once_with(  # type: ignore[union-attr]
            ["git", "add", "--all", ".github/repos"]
        )



class TestHasStagedChanges:
    @patch("admin_tools.services.git._run")
    def test_has_changes(self, mock_run: object) -> None:
        mock_run.return_value = subprocess.CompletedProcess(  # type: ignore[union-attr]
            args=[], returncode=1
        )
        assert has_staged_changes() is True

    @patch("admin_tools.services.git._run")
    def test_no_changes(self, mock_run: object) -> None:
        mock_run.return_value = subprocess.CompletedProcess(  # type: ignore[union-attr]
            args=[], returncode=0
        )
        assert has_staged_changes() is False


class TestCommit:
    @patch("admin_tools.services.git._run")
    def test_commit(self, mock_run: object) -> None:
        commit("chore: snapshot live repo settings")
        mock_run.assert_called_once_with(  # type: ignore[union-attr]
            ["git", "commit", "-m", "chore: snapshot live repo settings"]
        )


class TestForcePush:
    @patch("admin_tools.services.git._run")
    def test_force_push(self, mock_run: object) -> None:
        force_push("safe-settings/snapshot")
        mock_run.assert_called_once_with(  # type: ignore[union-attr]
            ["git", "push", "--force", "origin", "safe-settings/snapshot"]
        )


class TestPrExists:
    @patch("admin_tools.services.git._run")
    def test_pr_exists(self, mock_run: object) -> None:
        mock_run.return_value = subprocess.CompletedProcess(  # type: ignore[union-attr]
            args=[], returncode=0, stdout="42\n", stderr=""
        )
        assert pr_exists("my-branch") is True

    @patch("admin_tools.services.git._run")
    def test_pr_not_exists(self, mock_run: object) -> None:
        mock_run.return_value = subprocess.CompletedProcess(  # type: ignore[union-attr]
            args=[], returncode=0, stdout="", stderr=""
        )
        assert pr_exists("my-branch") is False

    @patch("admin_tools.services.git._run")
    def test_pr_check_uses_correct_args(self, mock_run: object) -> None:
        mock_run.return_value = subprocess.CompletedProcess(  # type: ignore[union-attr]
            args=[], returncode=0, stdout="", stderr=""
        )
        pr_exists("safe-settings/snapshot")
        mock_run.assert_called_once_with(  # type: ignore[union-attr]
            [
                "gh", "pr", "list",
                "--head", "safe-settings/snapshot",
                "--state", "open",
                "--json", "number",
                "--jq", ".[0].number",
            ],
            check=False,
        )


class TestCreatePr:
    @patch("admin_tools.services.git._run")
    def test_create_pr(self, mock_run: object) -> None:
        create_pr(
            title="chore: snapshot",
            body="Auto-generated.",
            head="my-branch",
            base="main",
        )
        mock_run.assert_called_once_with([  # type: ignore[union-attr]
            "gh", "pr", "create",
            "--title", "chore: snapshot",
            "--body", "Auto-generated.",
            "--head", "my-branch",
            "--base", "main",
        ])
