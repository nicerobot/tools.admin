from admin_tools.models.github import GitHubRepository
from admin_tools.models.overrides import RepoOverrideFile, RepositoryOverrides
from admin_tools.models.settings import RepositoryDefaults


def compute_overrides(
    repo: GitHubRepository,
    defaults: RepositoryDefaults,
    owner: str,
    comment_source: str,
) -> RepoOverrideFile:
    overrides: dict[str, str | bool] = {}

    # description and homepage are always included if non-empty (no default)
    if repo.description:
        overrides["description"] = repo.description
    if repo.homepage:
        overrides["homepage"] = repo.homepage

    if repo.visibility != defaults.visibility:
        overrides["visibility"] = repo.visibility
    if repo.default_branch != defaults.default_branch:
        overrides["default_branch"] = repo.default_branch
    if repo.has_issues != defaults.has_issues:
        overrides["has_issues"] = repo.has_issues
    if repo.has_projects != defaults.has_projects:
        overrides["has_projects"] = repo.has_projects
    if repo.has_wiki != defaults.has_wiki:
        overrides["has_wiki"] = repo.has_wiki
    if repo.has_discussions != defaults.has_discussions:
        overrides["has_discussions"] = repo.has_discussions
    if repo.is_template != defaults.is_template:
        overrides["is_template"] = repo.is_template
    if repo.allow_squash_merge != defaults.allow_squash_merge:
        overrides["allow_squash_merge"] = repo.allow_squash_merge
    if repo.allow_merge_commit != defaults.allow_merge_commit:
        overrides["allow_merge_commit"] = repo.allow_merge_commit
    if repo.allow_rebase_merge != defaults.allow_rebase_merge:
        overrides["allow_rebase_merge"] = repo.allow_rebase_merge
    if repo.allow_auto_merge != defaults.allow_auto_merge:
        overrides["allow_auto_merge"] = repo.allow_auto_merge
    if repo.delete_branch_on_merge != defaults.delete_branch_on_merge:
        overrides["delete_branch_on_merge"] = repo.delete_branch_on_merge
    if repo.archived:
        overrides["archived"] = True

    return RepoOverrideFile(
        owner=owner,
        name=repo.name,
        comment_source=comment_source,
        is_fork=repo.fork,
        repository=RepositoryOverrides(**overrides),
    )
