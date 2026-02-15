import os
import re
from types import TracebackType

import httpx

from admin_tools.models.github import GitHubRepository, GitHubUser, WorkflowRun

_LINK_NEXT_RE = re.compile(r'<([^>]+)>;\s*rel="next"')


class GitHubClient:
    def __init__(self, token: str | None = None) -> None:
        tok = token or os.environ.get("GH_TOKEN", "")
        if not tok:
            raise RuntimeError("GH_TOKEN environment variable must be set")
        self._client = httpx.AsyncClient(
            base_url="https://api.github.com",
            headers={
                "Authorization": f"Bearer {tok}",
                "Accept": "application/vnd.github+json",
                "X-GitHub-Api-Version": "2022-11-28",
            },
            timeout=30.0,
        )

    async def __aenter__(self) -> "GitHubClient":
        return self

    async def __aexit__(
        self,
        exc_type: type[BaseException] | None,
        exc_val: BaseException | None,
        exc_tb: TracebackType | None,
    ) -> None:
        await self._client.aclose()

    async def _paginate(
        self,
        url: str,
        params: dict[str, str] | None = None,
        *,
        items_key: str | None = None,
    ) -> list[dict[str, object]]:
        results: list[dict[str, object]] = []
        next_url: str | None = url
        query = params or {}
        while next_url is not None:
            resp = await self._client.get(next_url, params=query)
            resp.raise_for_status()
            data = resp.json()
            if items_key is not None:
                results.extend(data[items_key])
            elif isinstance(data, list):
                results.extend(data)
            else:
                results.append(data)
            # follow Link: <url>; rel="next"
            link = resp.headers.get("link", "")
            match = _LINK_NEXT_RE.search(link)
            if match:
                next_url = match.group(1)
                query = {}  # params are embedded in the next URL
            else:
                next_url = None
        return results

    async def get_account_type(self, owner: str) -> str:
        resp = await self._client.get(f"/users/{owner}")
        resp.raise_for_status()
        user = GitHubUser.model_validate(resp.json())
        return user.type

    async def repo_exists(self, owner: str, name: str) -> bool:
        """Check whether a repository exists at the exact owner/name path.

        Returns True on 200, False on 404 or 301.  A 301 means the repo
        was renamed or transferred — it no longer lives at owner/name.
        Raises on any other HTTP status so the caller can abort safely.
        """
        resp = await self._client.get(f"/repos/{owner}/{name}")
        if resp.status_code in (404, 301):
            return False
        resp.raise_for_status()
        return True

    async def list_repos(self, owner: str) -> list[GitHubRepository]:
        account_type = await self.get_account_type(owner)
        if account_type == "Organization":
            url = f"/orgs/{owner}/repos"
            params = {"type": "all", "per_page": "100"}
            raw = await self._paginate(url, params)
        else:
            # Use the authenticated /user/repos endpoint when the owner
            # matches the token holder.  The /users/{owner}/repos endpoint
            # only returns public repos for OAuth tokens, even with the
            # "repo" scope.  /user/repos?affiliation=owner returns all
            # repos — public and private — that the authenticated user owns.
            authenticated_user = await self._get_authenticated_login()
            if authenticated_user and authenticated_user == owner:
                url = "/user/repos"
                params = {"affiliation": "owner", "per_page": "100"}
                raw = await self._paginate(url, params)
            else:
                # For GitHub App installation tokens, GET /user returns 403
                # so authenticated_user is None.  Try the installation
                # repositories endpoint which lists all repos the App has
                # access to, then filter to the requested owner.
                install_repos = await self._list_installation_repos(owner)
                if install_repos is not None:
                    raw = install_repos
                else:
                    # Not an App token — fall back to the public endpoint.
                    url = f"/users/{owner}/repos"
                    params = {"type": "owner", "per_page": "100"}
                    raw = await self._paginate(url, params)
        return [GitHubRepository.model_validate(r) for r in raw]

    async def _list_installation_repos(
        self, owner: str,
    ) -> list[dict[str, object]] | None:
        """List repos via the App installation endpoint, filtered by owner.

        Returns None if the token is not a GitHub App installation token
        (i.e. the endpoint returns 403 or 404).
        """
        try:
            raw = await self._paginate(
                "/installation/repositories",
                {"per_page": "100"},
                items_key="repositories",
            )
        except httpx.HTTPStatusError:
            return None
        filtered: list[dict[str, object]] = []
        for repo in raw:
            repo_owner = repo.get("owner")
            if isinstance(repo_owner, dict) and repo_owner.get("login") == owner:
                filtered.append(repo)
        return filtered

    async def _get_authenticated_login(self) -> str | None:
        """Return the login of the authenticated user, or None on failure."""
        try:
            resp = await self._client.get("/user")
            resp.raise_for_status()
            login: str | None = resp.json().get("login")
            return login
        except httpx.HTTPStatusError:
            return None

    async def list_workflow_runs(
        self,
        owner: str,
        repo: str,
        *,
        status: str = "completed",
        created_before: str | None = None,
    ) -> list[WorkflowRun]:
        """List workflow runs for a repository.

        ``created_before`` uses the GitHub ``created`` query parameter with
        ``<YYYY-MM-DD`` syntax to filter runs created before that date.
        """
        params: dict[str, str] = {"per_page": "100", "status": status}
        if created_before is not None:
            params["created"] = f"<{created_before}"
        url = f"/repos/{owner}/{repo}/actions/runs"
        raw = await self._paginate(url, params, items_key="workflow_runs")
        return [WorkflowRun.model_validate(r) for r in raw]

    async def delete_workflow_run(
        self, owner: str, repo: str, run_id: int,
    ) -> None:
        resp = await self._client.delete(
            f"/repos/{owner}/{repo}/actions/runs/{run_id}",
        )
        resp.raise_for_status()
