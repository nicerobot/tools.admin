import asyncio
import os
from collections import defaultdict
from datetime import UTC, datetime, timedelta

from admin_tools.models.github import WorkflowRun
from admin_tools.services.github_api import GitHubClient


def _resolve_target(
    owner: str | None, repo: str | None,
) -> tuple[str, str | None]:
    """Resolve the owner and repo to clean.

    When ``owner`` is omitted, auto-detect the current repository from the
    ``GITHUB_REPOSITORY`` environment variable (``owner/name``) that GitHub
    Actions injects into every workflow run.  This lets the tool clean
    "the repo I'm running in" with no configuration.  An explicit ``owner``
    with no ``repo`` keeps the all-repos-under-owner behavior.
    """
    if owner is not None:
        return owner, repo
    detected_owner, _, detected_repo = os.environ.get(
        "GITHUB_REPOSITORY", "",
    ).partition("/")
    if not detected_owner or not detected_repo:
        raise RuntimeError(
            "--owner is required when GITHUB_REPOSITORY is unset "
            "(not running in GitHub Actions?)",
        )
    return detected_owner, detected_repo


async def _run_cleanup_runs_async(
    owner: str | None,
    repo: str | None,
    days: int,
    keep: int,
    dry_run: bool,
) -> None:
    owner, repo = _resolve_target(owner, repo)
    cutoff = datetime.now(tz=UTC) - timedelta(days=days)
    cutoff_date = cutoff.strftime("%Y-%m-%d")

    async with GitHubClient() as client:
        if repo is not None:
            repo_names = [repo]
        else:
            repos = await client.list_repos(owner)
            repo_names = [r.name for r in repos]

        total_deleted = 0
        total_kept = 0

        for repo_name in sorted(repo_names):
            runs = await client.list_workflow_runs(
                owner, repo_name, created_before=cutoff_date,
            )
            if not runs:
                continue

            by_workflow: dict[int, list[WorkflowRun]] = defaultdict(list)
            for run in runs:
                by_workflow[run.workflow_id].append(run)

            to_delete: list[WorkflowRun] = []
            kept = 0
            for _wf_id, wf_runs in by_workflow.items():
                # newest first so we keep the most recent
                wf_runs.sort(key=lambda r: r.created_at, reverse=True)
                to_delete.extend(wf_runs[keep:])
                kept += min(len(wf_runs), keep)

            if not to_delete:
                continue

            print(f"{owner}/{repo_name}: deleting {len(to_delete)}, "
                  f"keeping {kept}")

            for run in to_delete:
                label = f"  run {run.id} ({run.name}, {run.created_at})"
                if dry_run:
                    print(f"  [dry-run] would delete {label}")
                else:
                    await client.delete_workflow_run(owner, repo_name, run.id)
                    print(f"  deleted {label}")

            total_deleted += len(to_delete)
            total_kept += kept

    action = "would delete" if dry_run else "deleted"
    print(f"\nSummary: {len(repo_names)} repos scanned, "
          f"{total_deleted} runs {action}, {total_kept} runs kept")


def run_cleanup_runs(
    owner: str | None,
    repo: str | None,
    days: int,
    keep: int,
    dry_run: bool,
) -> None:
    asyncio.run(_run_cleanup_runs_async(owner, repo, days, keep, dry_run))
