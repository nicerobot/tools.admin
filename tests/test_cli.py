from pathlib import Path
from unittest.mock import patch

import pytest

from admin_tools.cli import main


class TestCliSnapshot:
    def test_snapshot_requires_owner(self) -> None:
        with pytest.raises(SystemExit, match="2"):
            main(["snapshot"])

    def test_snapshot_calls_run_snapshot(self) -> None:
        with patch("admin_tools.commands.snapshot.run_snapshot") as mock:
            main(["snapshot", "--owner", "nicerobot"])
        mock.assert_called_once_with("nicerobot", Path(".github"))

    def test_snapshot_custom_settings_path(self) -> None:
        with patch("admin_tools.commands.snapshot.run_snapshot") as mock:
            main(["snapshot", "--owner", "myorg", "--settings-path", "custom"])
        mock.assert_called_once_with("myorg", Path("custom"))


class TestCliCreatePr:
    def test_create_pr_defaults(self) -> None:
        with patch("admin_tools.commands.create_pr.run_create_pr") as mock:
            main(["create-pr"])
        mock.assert_called_once_with(
            settings_path=Path(".github"),
            branch="safe-settings/snapshot",
            base="main",
        )

    def test_create_pr_custom_args(self) -> None:
        with patch("admin_tools.commands.create_pr.run_create_pr") as mock:
            main([
                "create-pr",
                "--settings-path", "custom",
                "--branch", "my/branch",
                "--base", "develop",
            ])
        mock.assert_called_once_with(
            settings_path=Path("custom"),
            branch="my/branch",
            base="develop",
        )


class TestCliCleanupRuns:
    def test_cleanup_runs_requires_owner(self) -> None:
        with pytest.raises(SystemExit, match="2"):
            main(["cleanup-runs"])

    def test_cleanup_runs_defaults(self) -> None:
        with patch(
            "admin_tools.commands.cleanup_runs.run_cleanup_runs",
        ) as mock:
            main(["cleanup-runs", "--owner", "nicerobot"])
        mock.assert_called_once_with(
            owner="nicerobot",
            repo=None,
            days=30,
            keep=5,
            dry_run=False,
        )

    def test_cleanup_runs_custom_args(self) -> None:
        with patch(
            "admin_tools.commands.cleanup_runs.run_cleanup_runs",
        ) as mock:
            main([
                "cleanup-runs",
                "--owner", "myorg",
                "--repo", "myrepo",
                "--days", "7",
                "--keep", "3",
                "--dry-run",
            ])
        mock.assert_called_once_with(
            owner="myorg",
            repo="myrepo",
            days=7,
            keep=3,
            dry_run=True,
        )

    def test_cleanup_runs_dry_run_flag(self) -> None:
        with patch(
            "admin_tools.commands.cleanup_runs.run_cleanup_runs",
        ) as mock:
            main(["cleanup-runs", "--owner", "nicerobot", "--dry-run"])
        mock.assert_called_once()
        assert mock.call_args.kwargs["dry_run"] is True


class TestCliGeneral:
    def test_no_command_exits(self) -> None:
        with pytest.raises(SystemExit, match="2"):
            main([])

    def test_unknown_command_exits(self) -> None:
        with pytest.raises(SystemExit, match="2"):
            main(["unknown"])
