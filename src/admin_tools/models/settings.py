from pydantic import BaseModel


class Label(BaseModel):
    name: str
    color: str
    description: str = ""


class Collaborator(BaseModel):
    username: str
    permission: str = "push"


class RepositoryDefaults(BaseModel):
    default_branch: str = "main"
    visibility: str = "private"
    has_issues: bool = False
    has_projects: bool = False
    has_wiki: bool = False
    has_discussions: bool = False
    is_template: bool = False
    allow_squash_merge: bool = True
    allow_merge_commit: bool = True
    allow_rebase_merge: bool = True
    allow_auto_merge: bool = False
    delete_branch_on_merge: bool = True


class OrgSettings(BaseModel):
    repository: RepositoryDefaults = RepositoryDefaults()
    labels: list[Label] = []
    collaborators: list[Collaborator] = []
