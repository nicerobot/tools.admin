from pydantic import BaseModel


class RepositoryOverrides(BaseModel):
    description: str | None = None
    homepage: str | None = None
    visibility: str | None = None
    default_branch: str | None = None
    has_issues: bool | None = None
    has_projects: bool | None = None
    has_wiki: bool | None = None
    has_discussions: bool | None = None
    is_template: bool | None = None
    allow_squash_merge: bool | None = None
    allow_merge_commit: bool | None = None
    allow_rebase_merge: bool | None = None
    allow_auto_merge: bool | None = None
    delete_branch_on_merge: bool | None = None
    archived: bool | None = None


class RepoOverrideFile(BaseModel):
    owner: str
    name: str
    comment_source: str
    is_fork: bool = False
    repository: RepositoryOverrides = RepositoryOverrides()
