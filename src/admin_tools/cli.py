import argparse
import sys
from pathlib import Path


def main(argv: list[str] | None = None) -> None:
    parser = argparse.ArgumentParser(
        prog="admin-tools",
        description="GitHub admin automation tools",
    )
    sub = parser.add_subparsers(dest="command", required=True)

    snap = sub.add_parser("snapshot", help="Snapshot live repo settings")
    snap.add_argument(
        "--owner", required=True, help="GitHub user or organization",
    )
    snap.add_argument(
        "--settings-path", default=".github",
        help="Path to settings directory",
    )

    pr = sub.add_parser("create-pr", help="Create PR from snapshot")
    pr.add_argument(
        "--settings-path", default=".github",
        help="Path to settings directory",
    )
    pr.add_argument(
        "--branch", default="safe-settings/snapshot",
        help="Branch name",
    )
    pr.add_argument("--base", default="main", help="Base branch")

    cr = sub.add_parser("cleanup-runs", help="Delete old workflow runs")
    cr.add_argument(
        "--owner", required=True, help="GitHub user or organization",
    )
    cr.add_argument(
        "--repo", default=None, help="Single repo (omit for all repos)",
    )
    cr.add_argument(
        "--days", type=int, default=30,
        help="Delete runs older than N days",
    )
    cr.add_argument(
        "--keep", type=int, default=5,
        help="Keep at least N runs per workflow",
    )
    cr.add_argument(
        "--dry-run", action="store_true", default=False,
        help="Print what would be deleted without deleting",
    )

    args = parser.parse_args(argv)

    match args.command:
        case "snapshot":
            from admin_tools.commands.snapshot import run_snapshot

            run_snapshot(args.owner, Path(args.settings_path))
        case "create-pr":
            from admin_tools.commands.create_pr import run_create_pr

            run_create_pr(
                settings_path=Path(args.settings_path),
                branch=args.branch,
                base=args.base,
            )
        case "cleanup-runs":
            from admin_tools.commands.cleanup_runs import run_cleanup_runs

            run_cleanup_runs(
                owner=args.owner,
                repo=args.repo,
                days=args.days,
                keep=args.keep,
                dry_run=args.dry_run,
            )
        case _:
            parser.print_help()
            sys.exit(1)
