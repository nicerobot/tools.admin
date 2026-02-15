from pydantic import BaseModel, computed_field


class GitHubUser(BaseModel):
    login: str
    type: str


class WorkflowRun(BaseModel):
    id: int
    name: str | None = None
    status: str
    conclusion: str | None = None
    created_at: str
    workflow_id: int


class GitHubRepository(BaseModel):
    name: str
    description: str | None = None
    homepage: str | None = None
    private: bool = False
    default_branch: str = "main"
    has_issues: bool = True
    has_projects: bool = True
    has_wiki: bool = True
    has_discussions: bool = False
    is_template: bool = False
    allow_squash_merge: bool = True
    allow_merge_commit: bool = True
    allow_rebase_merge: bool = True
    allow_auto_merge: bool = False
    delete_branch_on_merge: bool = False
    archived: bool = False
    fork: bool = False

    @computed_field  # type: ignore[prop-decorator]
    @property
    def visibility(self) -> str:
        return "private" if self.private else "public"
