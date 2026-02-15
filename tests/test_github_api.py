
import httpx
import pytest

from admin_tools.services.github_api import GitHubClient


def _make_repo(name: str, **kwargs: object) -> dict[str, object]:
    base: dict[str, object] = {
        "name": name,
        "description": None,
        "homepage": None,
        "private": False,
        "default_branch": "main",
        "has_issues": True,
        "has_projects": True,
        "has_wiki": True,
        "has_discussions": False,
        "is_template": False,
        "allow_squash_merge": True,
        "allow_merge_commit": True,
        "allow_rebase_merge": True,
        "allow_auto_merge": False,
        "delete_branch_on_merge": False,
        "archived": False,
        "fork": False,
    }
    base.update(kwargs)
    return base


def _make_user(login: str, account_type: str = "User") -> dict[str, str]:
    return {"login": login, "type": account_type}


class TestGitHubClientInit:
    def test_requires_token(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.delenv("GH_TOKEN", raising=False)
        with pytest.raises(RuntimeError, match="GH_TOKEN"):
            GitHubClient()

    def test_accepts_explicit_token(self) -> None:
        client = GitHubClient(token="ghp_test123")
        assert client._client.headers["authorization"] == "Bearer ghp_test123"

    def test_reads_env_token(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setenv("GH_TOKEN", "ghp_envtoken")
        client = GitHubClient()
        assert client._client.headers["authorization"] == "Bearer ghp_envtoken"


class TestGetAccountType:
    @pytest.mark.asyncio
    async def test_user_account(self) -> None:
        def handler(request: httpx.Request) -> httpx.Response:
            assert "/users/nicerobot" in str(request.url)
            return httpx.Response(200, json=_make_user("nicerobot", "User"))

        transport = httpx.MockTransport(handler)
        client = GitHubClient(token="test")
        client._client = httpx.AsyncClient(
            transport=transport,
            base_url="https://api.github.com",
        )
        async with client:
            result = await client.get_account_type("nicerobot")
        assert result == "User"

    @pytest.mark.asyncio
    async def test_org_account(self) -> None:
        def handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(200, json=_make_user("myorg", "Organization"))

        transport = httpx.MockTransport(handler)
        client = GitHubClient(token="test")
        client._client = httpx.AsyncClient(
            transport=transport,
            base_url="https://api.github.com",
        )
        async with client:
            result = await client.get_account_type("myorg")
        assert result == "Organization"

    @pytest.mark.asyncio
    async def test_api_error_raises(self) -> None:
        def handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(404, json={"message": "Not Found"})

        transport = httpx.MockTransport(handler)
        client = GitHubClient(token="test")
        client._client = httpx.AsyncClient(
            transport=transport,
            base_url="https://api.github.com",
        )
        async with client:
            with pytest.raises(httpx.HTTPStatusError):
                await client.get_account_type("nonexistent")


class TestListRepos:
    @pytest.mark.asyncio
    async def test_user_repos_single_page(self) -> None:
        repos = [_make_repo("repo1"), _make_repo("repo2")]

        def handler(request: httpx.Request) -> httpx.Response:
            if request.url.path == "/user":
                return httpx.Response(200, json={"login": "nicerobot"})
            if request.url.path == "/users/nicerobot":
                return httpx.Response(
                    200, json=_make_user("nicerobot", "User")
                )
            return httpx.Response(200, json=repos)

        transport = httpx.MockTransport(handler)
        client = GitHubClient(token="test")
        client._client = httpx.AsyncClient(
            transport=transport,
            base_url="https://api.github.com",
        )
        async with client:
            result = await client.list_repos("nicerobot")
        assert len(result) == 2
        assert result[0].name == "repo1"
        assert result[1].name == "repo2"

    @pytest.mark.asyncio
    async def test_authenticated_user_uses_affiliation(self) -> None:
        captured_params: dict[str, str] = {}
        captured_urls: list[str] = []

        def handler(request: httpx.Request) -> httpx.Response:
            if request.url.path == "/user":
                return httpx.Response(200, json={"login": "nicerobot"})
            if (
                request.url.path.endswith("/nicerobot")
                and "/users/" in request.url.path
            ):
                return httpx.Response(
                    200, json=_make_user("nicerobot", "User")
                )
            captured_params.update(dict(request.url.params))
            captured_urls.append(request.url.path)
            return httpx.Response(200, json=[])

        transport = httpx.MockTransport(handler)
        client = GitHubClient(token="test")
        client._client = httpx.AsyncClient(
            transport=transport,
            base_url="https://api.github.com",
        )
        async with client:
            await client.list_repos("nicerobot")
        assert captured_params.get("affiliation") == "owner"
        assert captured_params.get("per_page") == "100"
        assert "/user/repos" in captured_urls

    @pytest.mark.asyncio
    async def test_other_user_uses_type_owner(self) -> None:
        """Non-App token viewing another user falls back to /users endpoint."""
        captured_params: dict[str, str] = {}

        def handler(request: httpx.Request) -> httpx.Response:
            if request.url.path == "/user":
                return httpx.Response(200, json={"login": "me"})
            if request.url.path == "/users/other":
                return httpx.Response(
                    200, json=_make_user("other", "User")
                )
            if request.url.path == "/installation/repositories":
                return httpx.Response(403, json={"message": "Forbidden"})
            captured_params.update(dict(request.url.params))
            return httpx.Response(200, json=[])

        transport = httpx.MockTransport(handler)
        client = GitHubClient(token="test")
        client._client = httpx.AsyncClient(
            transport=transport,
            base_url="https://api.github.com",
        )
        async with client:
            await client.list_repos("other")
        assert captured_params.get("type") == "owner"
        assert captured_params.get("per_page") == "100"

    @pytest.mark.asyncio
    async def test_app_token_uses_installation_repos(self) -> None:
        """GitHub App installation tokens use /installation/repositories."""
        repos = [
            {**_make_repo("repo1"), "owner": {"login": "nicerobot"}},
            {**_make_repo("repo2"), "owner": {"login": "nicerobot"}},
            {**_make_repo("other-repo"), "owner": {"login": "other"}},
        ]

        def handler(request: httpx.Request) -> httpx.Response:
            if request.url.path == "/user":
                return httpx.Response(403, json={"message": "Forbidden"})
            if request.url.path == "/users/nicerobot":
                return httpx.Response(
                    200, json=_make_user("nicerobot", "User")
                )
            if request.url.path == "/installation/repositories":
                return httpx.Response(
                    200,
                    json={
                        "total_count": len(repos),
                        "repositories": repos,
                    },
                )
            return httpx.Response(404, json={"message": "Not Found"})

        transport = httpx.MockTransport(handler)
        client = GitHubClient(token="test")
        client._client = httpx.AsyncClient(
            transport=transport,
            base_url="https://api.github.com",
        )
        async with client:
            result = await client.list_repos("nicerobot")
        assert len(result) == 2
        assert result[0].name == "repo1"
        assert result[1].name == "repo2"

    @pytest.mark.asyncio
    async def test_org_endpoint_uses_type_all(self) -> None:
        captured_urls: list[str] = []

        def handler(request: httpx.Request) -> httpx.Response:
            captured_urls.append(str(request.url))
            if request.url.path.endswith("/myorg") and "/users/" in request.url.path:
                return httpx.Response(200, json=_make_user("myorg", "Organization"))
            return httpx.Response(200, json=[])

        transport = httpx.MockTransport(handler)
        client = GitHubClient(token="test")
        client._client = httpx.AsyncClient(
            transport=transport,
            base_url="https://api.github.com",
        )
        async with client:
            await client.list_repos("myorg")
        repos_url = next(
            u for u in captured_urls if "/orgs/" in u
        )
        assert "type=all" in repos_url
        assert "per_page=100" in repos_url

    @pytest.mark.asyncio
    async def test_pagination(self) -> None:
        page1 = [_make_repo(f"repo{i}") for i in range(100)]
        page2 = [_make_repo("repo100"), _make_repo("repo101")]
        call_count = 0

        def handler(request: httpx.Request) -> httpx.Response:
            nonlocal call_count
            if request.url.path == "/user":
                return httpx.Response(200, json={"login": "nicerobot"})
            if (
                request.url.path.endswith("/nicerobot")
                and "/users/" in request.url.path
            ):
                return httpx.Response(
                    200, json=_make_user("nicerobot", "User")
                )
            call_count += 1
            if call_count == 1:
                next_url = (
                    "<https://api.github.com"
                    "/user/repos?page=2>"
                    '; rel="next"'
                )
                return httpx.Response(
                    200,
                    json=page1,
                    headers={"link": next_url},
                )
            return httpx.Response(200, json=page2)

        transport = httpx.MockTransport(handler)
        client = GitHubClient(token="test")
        client._client = httpx.AsyncClient(
            transport=transport,
            base_url="https://api.github.com",
        )
        async with client:
            result = await client.list_repos("nicerobot")
        assert len(result) == 102

    @pytest.mark.asyncio
    async def test_repo_visibility_computed(self) -> None:
        repos = [
            _make_repo("pub", private=False),
            _make_repo("priv", private=True),
        ]

        def handler(request: httpx.Request) -> httpx.Response:
            if request.url.path == "/user":
                return httpx.Response(200, json={"login": "nicerobot"})
            if (
                "/users/" in request.url.path
                and request.url.path.endswith("/nicerobot")
            ):
                return httpx.Response(
                    200, json=_make_user("nicerobot", "User")
                )
            return httpx.Response(200, json=repos)

        transport = httpx.MockTransport(handler)
        client = GitHubClient(token="test")
        client._client = httpx.AsyncClient(
            transport=transport,
            base_url="https://api.github.com",
        )
        async with client:
            result = await client.list_repos("nicerobot")
        assert result[0].visibility == "public"
        assert result[1].visibility == "private"

    @pytest.mark.asyncio
    async def test_fork_and_archived_parsed(self) -> None:
        repos = [
            _make_repo("forked", fork=True),
            _make_repo("old", archived=True),
        ]

        def handler(request: httpx.Request) -> httpx.Response:
            if request.url.path == "/user":
                return httpx.Response(200, json={"login": "nicerobot"})
            if (
                "/users/" in request.url.path
                and request.url.path.endswith("/nicerobot")
            ):
                return httpx.Response(
                    200, json=_make_user("nicerobot", "User")
                )
            return httpx.Response(200, json=repos)

        transport = httpx.MockTransport(handler)
        client = GitHubClient(token="test")
        client._client = httpx.AsyncClient(
            transport=transport,
            base_url="https://api.github.com",
        )
        async with client:
            result = await client.list_repos("nicerobot")
        assert result[0].fork is True
        assert result[1].archived is True


class TestRepoExists:
    @pytest.mark.asyncio
    async def test_returns_true_on_200(self) -> None:
        def handler(request: httpx.Request) -> httpx.Response:
            assert "/repos/nicerobot/myrepo" in str(request.url)
            return httpx.Response(200, json=_make_repo("myrepo"))

        transport = httpx.MockTransport(handler)
        client = GitHubClient(token="test")
        client._client = httpx.AsyncClient(
            transport=transport,
            base_url="https://api.github.com",
        )
        async with client:
            assert await client.repo_exists("nicerobot", "myrepo") is True

    @pytest.mark.asyncio
    async def test_returns_false_on_404(self) -> None:
        def handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(404, json={"message": "Not Found"})

        transport = httpx.MockTransport(handler)
        client = GitHubClient(token="test")
        client._client = httpx.AsyncClient(
            transport=transport,
            base_url="https://api.github.com",
        )
        async with client:
            assert await client.repo_exists("nicerobot", "gone") is False

    @pytest.mark.asyncio
    async def test_returns_false_on_301_renamed(self) -> None:
        def handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(
                301,
                headers={"location": "https://api.github.com/repositories/123"},
            )

        transport = httpx.MockTransport(handler)
        client = GitHubClient(token="test")
        client._client = httpx.AsyncClient(
            transport=transport,
            base_url="https://api.github.com",
        )
        async with client:
            assert await client.repo_exists("nicerobot", "old-name") is False

    @pytest.mark.asyncio
    async def test_raises_on_500(self) -> None:
        def handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(500, json={"message": "Server Error"})

        transport = httpx.MockTransport(handler)
        client = GitHubClient(token="test")
        client._client = httpx.AsyncClient(
            transport=transport,
            base_url="https://api.github.com",
        )
        async with client:
            with pytest.raises(httpx.HTTPStatusError):
                await client.repo_exists("nicerobot", "broken")


def _make_run(
    run_id: int,
    workflow_id: int = 1,
    name: str = "CI",
    created_at: str = "2025-01-01T00:00:00Z",
) -> dict[str, object]:
    return {
        "id": run_id,
        "name": name,
        "status": "completed",
        "conclusion": "success",
        "created_at": created_at,
        "workflow_id": workflow_id,
    }


class TestListWorkflowRuns:
    @pytest.mark.asyncio
    async def test_returns_runs(self) -> None:
        runs = [_make_run(1), _make_run(2)]

        def handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(
                200,
                json={"total_count": 2, "workflow_runs": runs},
            )

        transport = httpx.MockTransport(handler)
        client = GitHubClient(token="test")
        client._client = httpx.AsyncClient(
            transport=transport,
            base_url="https://api.github.com",
        )
        async with client:
            result = await client.list_workflow_runs("nicerobot", "repo1")
        assert len(result) == 2
        assert result[0].id == 1

    @pytest.mark.asyncio
    async def test_passes_created_before_param(self) -> None:
        captured_params: dict[str, str] = {}

        def handler(request: httpx.Request) -> httpx.Response:
            captured_params.update(dict(request.url.params))
            return httpx.Response(
                200,
                json={"total_count": 0, "workflow_runs": []},
            )

        transport = httpx.MockTransport(handler)
        client = GitHubClient(token="test")
        client._client = httpx.AsyncClient(
            transport=transport,
            base_url="https://api.github.com",
        )
        async with client:
            await client.list_workflow_runs(
                "nicerobot", "repo1", created_before="2025-12-01",
            )
        assert captured_params.get("created") == "<2025-12-01"

    @pytest.mark.asyncio
    async def test_pagination(self) -> None:
        page1_runs = [_make_run(i) for i in range(100)]
        page2_runs = [_make_run(100), _make_run(101)]
        call_count = 0

        def handler(request: httpx.Request) -> httpx.Response:
            nonlocal call_count
            call_count += 1
            if call_count == 1:
                next_url = (
                    "<https://api.github.com"
                    "/repos/nicerobot/repo1/actions/runs?page=2>"
                    '; rel="next"'
                )
                return httpx.Response(
                    200,
                    json={"total_count": 102, "workflow_runs": page1_runs},
                    headers={"link": next_url},
                )
            return httpx.Response(
                200,
                json={"total_count": 102, "workflow_runs": page2_runs},
            )

        transport = httpx.MockTransport(handler)
        client = GitHubClient(token="test")
        client._client = httpx.AsyncClient(
            transport=transport,
            base_url="https://api.github.com",
        )
        async with client:
            result = await client.list_workflow_runs("nicerobot", "repo1")
        assert len(result) == 102


class TestDeleteWorkflowRun:
    @pytest.mark.asyncio
    async def test_delete_returns_204(self) -> None:
        def handler(request: httpx.Request) -> httpx.Response:
            assert request.method == "DELETE"
            assert "/actions/runs/42" in str(request.url)
            return httpx.Response(204)

        transport = httpx.MockTransport(handler)
        client = GitHubClient(token="test")
        client._client = httpx.AsyncClient(
            transport=transport,
            base_url="https://api.github.com",
        )
        async with client:
            await client.delete_workflow_run("nicerobot", "repo1", 42)


class TestContextManager:
    @pytest.mark.asyncio
    async def test_aenter_aexit(self) -> None:
        client = GitHubClient(token="test")
        async with client as c:
            assert c is client
