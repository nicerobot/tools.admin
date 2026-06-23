import subprocess

_BOT_EMAIL = (
    "41898282+github-actions[bot]@users.noreply.github.com"
)


def _run(
    args: list[str], check: bool = True,
) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        args, capture_output=True, text=True, check=check,
    )


def configure_bot_identity() -> None:
    _run(["git", "config", "user.name", "github-actions[bot]"])
    _run(["git", "config", "user.email", _BOT_EMAIL])


def checkout_branch(branch: str) -> None:
    _run(["git", "checkout", "-B", branch])


def stage_directory(path: str) -> None:
    _run(["git", "add", "--all", path])



def has_staged_changes() -> bool:
    result = _run(
        ["git", "diff", "--cached", "--quiet"], check=False,
    )
    return result.returncode != 0


def commit(message: str) -> None:
    _run(["git", "commit", "-m", message])


def force_push(branch: str) -> None:
    _run(["git", "push", "--force", "origin", branch])


def pr_exists(branch: str) -> bool:
    result = _run(
        [
            "gh", "pr", "list",
            "--head", branch,
            "--state", "open",
            "--json", "number",
            "--jq", ".[0].number",
        ],
        check=False,
    )
    return bool(result.stdout.strip())


def create_pr(
    *, title: str, body: str, head: str, base: str,
) -> None:
    _run([
        "gh", "pr", "create",
        "--title", title,
        "--body", body,
        "--head", head,
        "--base", base,
    ])
