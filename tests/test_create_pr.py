from pathlib import Path
from unittest.mock import patch

from admin_tools.commands.create_pr import run_create_pr


class TestRunCreatePr:
    def test_full_flow_with_changes(self, tmp_path: Path) -> None:
        settings_path = tmp_path / ".github"
        with (
            patch("admin_tools.commands.create_pr.configure_bot_identity") as mock_id,
            patch("admin_tools.commands.create_pr.checkout_branch") as mock_co,
            patch("admin_tools.commands.create_pr.stage_directory") as mock_stage,
            patch(
                "admin_tools.commands.create_pr.has_staged_changes",
                return_value=True,
            ) as mock_has,
            patch("admin_tools.commands.create_pr.commit") as mock_commit,
            patch("admin_tools.commands.create_pr.force_push") as mock_push,
            patch(
                "admin_tools.commands.create_pr.pr_exists",
                return_value=False,
            ) as mock_pr_exists,
            patch("admin_tools.commands.create_pr.create_pr") as mock_create_pr,
        ):
            run_create_pr(settings_path=settings_path)

        mock_id.assert_called_once()
        mock_co.assert_called_once_with("safe-settings/snapshot")
        mock_stage.assert_called_once_with(str(settings_path / "repos"))
        mock_has.assert_called_once()
        mock_commit.assert_called_once_with("chore: snapshot live repo settings")
        mock_push.assert_called_once_with("safe-settings/snapshot")
        mock_pr_exists.assert_called_once_with("safe-settings/snapshot")
        mock_create_pr.assert_called_once_with(
            title="chore: snapshot live repo settings",
            body=(
                "Auto-generated snapshot of current"
                " GitHub repo settings vs org/account"
                " defaults."
            ),
            head="safe-settings/snapshot",
            base="main",
        )

    def test_no_changes_exits_early(self, tmp_path: Path, capsys: object) -> None:
        settings_path = tmp_path / ".github"
        with (
            patch("admin_tools.commands.create_pr.configure_bot_identity"),
            patch("admin_tools.commands.create_pr.checkout_branch"),
            patch("admin_tools.commands.create_pr.stage_directory"),
            patch(
                "admin_tools.commands.create_pr.has_staged_changes",
                return_value=False,
            ),
            patch("admin_tools.commands.create_pr.commit") as mock_commit,
            patch("admin_tools.commands.create_pr.force_push") as mock_push,
            patch("admin_tools.commands.create_pr.create_pr") as mock_create_pr,
        ):
            run_create_pr(settings_path=settings_path)

        mock_commit.assert_not_called()
        mock_push.assert_not_called()
        mock_create_pr.assert_not_called()

    def test_existing_pr_skips_creation(self, tmp_path: Path) -> None:
        settings_path = tmp_path / ".github"
        with (
            patch("admin_tools.commands.create_pr.configure_bot_identity"),
            patch("admin_tools.commands.create_pr.checkout_branch"),
            patch("admin_tools.commands.create_pr.stage_directory"),
            patch(
                "admin_tools.commands.create_pr.has_staged_changes",
                return_value=True,
            ),
            patch("admin_tools.commands.create_pr.commit"),
            patch("admin_tools.commands.create_pr.force_push"),
            patch(
                "admin_tools.commands.create_pr.pr_exists",
                return_value=True,
            ),
            patch("admin_tools.commands.create_pr.create_pr") as mock_create_pr,
        ):
            run_create_pr(settings_path=settings_path)

        mock_create_pr.assert_not_called()

    def test_custom_branch_and_base(self, tmp_path: Path) -> None:
        settings_path = tmp_path / ".github"
        with (
            patch("admin_tools.commands.create_pr.configure_bot_identity"),
            patch("admin_tools.commands.create_pr.checkout_branch") as mock_co,
            patch("admin_tools.commands.create_pr.stage_directory"),
            patch(
                "admin_tools.commands.create_pr.has_staged_changes",
                return_value=True,
            ),
            patch("admin_tools.commands.create_pr.commit"),
            patch("admin_tools.commands.create_pr.force_push") as mock_push,
            patch(
                "admin_tools.commands.create_pr.pr_exists",
                return_value=False,
            ),
            patch("admin_tools.commands.create_pr.create_pr") as mock_create_pr,
        ):
            run_create_pr(
                settings_path=settings_path,
                branch="custom/branch",
                base="develop",
            )

        mock_co.assert_called_once_with("custom/branch")
        mock_push.assert_called_once_with("custom/branch")
        mock_create_pr.assert_called_once()
        assert mock_create_pr.call_args.kwargs["head"] == "custom/branch"
        assert mock_create_pr.call_args.kwargs["base"] == "develop"

    def test_custom_settings_path(self, tmp_path: Path) -> None:
        settings_path = tmp_path / "custom-settings"
        with (
            patch("admin_tools.commands.create_pr.configure_bot_identity"),
            patch("admin_tools.commands.create_pr.checkout_branch"),
            patch(
                "admin_tools.commands.create_pr.stage_directory",
            ) as mock_stage,
            patch(
                "admin_tools.commands.create_pr.has_staged_changes",
                return_value=False,
            ),
        ):
            run_create_pr(settings_path=settings_path)

        mock_stage.assert_called_once_with(
            str(settings_path / "repos")
        )
