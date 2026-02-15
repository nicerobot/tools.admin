import asyncio
import sys
from pathlib import Path

from admin_tools.services.github_api import GitHubClient
from admin_tools.services.settings_io import (
    list_existing_repo_files,
    load_org_settings,
    write_repo_override,
)
from admin_tools.util.diff import compute_overrides


async def _run_snapshot_async(owner: str, settings_path: Path) -> None:
    settings = load_org_settings(settings_path)
    repos_dir = settings_path / "repos"

    async with GitHubClient() as client:
        account_type = await client.get_account_type(owner)
        comment_source = "org" if account_type == "Organization" else "account"
        repos = await client.list_repos(owner)

        live_names: set[str] = set()
        overrides = []
        for repo in repos:
            live_names.add(repo.name)
            overrides.append(
                compute_overrides(repo, settings.repository, owner, comment_source)
            )

        # identify stale candidates — files without a matching live repo
        existing = list_existing_repo_files(repos_dir)
        stale = existing - live_names

        # verify each stale candidate before any writes or deletes
        confirmed_gone: list[str] = []
        for name in sorted(stale):
            exists = await client.repo_exists(owner, name)
            if exists:
                print(
                    f"ERROR: Repo {owner}/{name} exists but was not "
                    "returned by list_repos().\n"
                    "This likely means the API token does not have "
                    "access to all repos.\n"
                    "Aborting to prevent data loss. No files were modified.",
                    file=sys.stderr,
                )
                sys.exit(1)
            confirmed_gone.append(name)

    # all verification passed — now apply changes
    for override in overrides:
        outfile = write_repo_override(override, repos_dir)
        print(f"  wrote {outfile}")

    for name in confirmed_gone:
        filepath = repos_dir / f"{name}.yml"
        print(f"  removing {filepath} (repo no longer exists)")
        filepath.unlink()


def run_snapshot(owner: str, settings_path: Path) -> None:
    asyncio.run(_run_snapshot_async(owner, settings_path))
