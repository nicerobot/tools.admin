from pathlib import Path

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


def run_create_pr(
    *,
    settings_path: Path,
    branch: str = "safe-settings/snapshot",
    base: str = "main",
) -> None:
    configure_bot_identity()
    checkout_branch(branch)
    stage_directory(str(settings_path / "repos"))

    if not has_staged_changes():
        print("No changes to commit.")
        return

    commit("chore: snapshot live repo settings")
    force_push(branch)

    if not pr_exists(branch):
        create_pr(
            title="chore: snapshot live repo settings",
            body=(
                "Auto-generated snapshot of current"
                " GitHub repo settings vs"
                " org/account defaults."
            ),
            head=branch,
            base=base,
        )
