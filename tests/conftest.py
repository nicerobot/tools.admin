from pathlib import Path

import pytest

from admin_tools.models.settings import (
    Collaborator,
    Label,
    OrgSettings,
    RepositoryDefaults,
)


@pytest.fixture
def default_settings() -> OrgSettings:
    return OrgSettings(
        repository=RepositoryDefaults(
            default_branch="main",
            visibility="private",
            has_issues=False,
            has_projects=False,
            has_wiki=False,
            has_discussions=False,
            is_template=False,
            allow_squash_merge=True,
            allow_merge_commit=True,
            allow_rebase_merge=True,
            allow_auto_merge=False,
            delete_branch_on_merge=True,
        ),
        labels=[
            Label(name="bug", color="d73a4a", description="Something isn't working"),
        ],
        collaborators=[
            Collaborator(username="nicerobot", permission="admin"),
        ],
    )


@pytest.fixture
def settings_dir(tmp_path: Path, default_settings: OrgSettings) -> Path:
    """Create a .github directory with settings.yml matching the real format."""
    github_dir = tmp_path / ".github"
    github_dir.mkdir()
    settings_file = github_dir / "settings.yml"
    settings_file.write_text(
        "repository:\n"
        "  default_branch: main\n"
        "  visibility: private\n"
        "  has_issues: false\n"
        "  has_projects: false\n"
        "  has_wiki: false\n"
        "  has_discussions: false\n"
        "  is_template: false\n"
        "  allow_squash_merge: true\n"
        "  allow_merge_commit: true\n"
        "  allow_rebase_merge: true\n"
        "  allow_auto_merge: false\n"
        "  delete_branch_on_merge: true\n"
        "\n"
        "labels:\n"
        '  - name: bug\n'
        '    color: d73a4a\n'
        '    description: "Something isn\'t working"\n'
        "\n"
        "collaborators:\n"
        "  - username: nicerobot\n"
        "    permission: admin\n"
    )
    return github_dir
