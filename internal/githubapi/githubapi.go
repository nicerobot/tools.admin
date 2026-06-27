// Package githubapi is the GitHub REST client tools.admin uses. All I/O goes
// through an injected Doer so tests run with a fake transport — no network. It
// reproduces the original client's endpoint-selection strategy (org vs
// authenticated-user vs App-installation vs public user) and Link-header
// pagination exactly.
package githubapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"

	"github.com/nicerobot/tools.admin/internal/adminerr"
	"github.com/nicerobot/tools.admin/internal/domain"
	"github.com/nicerobot/tools.admin/internal/githubmodel"
)

const (
	// DefaultBaseURL is the production GitHub REST API root.
	DefaultBaseURL = "https://api.github.com"
	tokenEnv       = "GH_TOKEN"
	acceptHeader   = "application/vnd.github+json"
	apiVersion     = "2022-11-28"
	perPage        = "100"
)

var linkNextRe = regexp.MustCompile(`<([^>]+)>;\s*rel="next"`)

// Doer issues an HTTP request; satisfied by *http.Client in production.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// EnvLookup reads an environment variable; injected for testability.
type EnvLookup func(key string) string

// Client is the GitHub API client. It is an immutable value safe to copy.
type Client struct {
	doer    Doer
	baseURL string
	token   string
}

// New builds a Client, reading the token from GH_TOKEN via env. A missing token
// is fatal (ErrNoToken), matching the original constructor. An empty baseURL
// resolves to DefaultBaseURL.
func New(doer Doer, baseURL string, env EnvLookup) (Client, error) {
	token := env(tokenEnv)
	if token == "" {
		return Client{}, adminerr.ErrNoToken.With(nil)
	}
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return Client{doer: doer, baseURL: baseURL, token: token}, nil
}

// GetAccountType returns the owner's account type (Organization, User, ...).
func (c Client) GetAccountType(owner domain.Owner) (domain.AccountType, error) {
	var u githubmodel.User
	if err := c.getJSON("/users/"+string(owner), &u); err != nil {
		return "", err
	}
	return u.AccountType(), nil
}

// RepoExists reports whether a repo exists at exactly owner/name. 404 and 301
// (renamed/transferred) count as gone; any other non-2xx is an error so the
// caller can abort safely.
func (c Client) RepoExists(owner domain.Owner, name domain.RepoName) (bool, error) {
	path := fmt.Sprintf("/repos/%s/%s", owner, name)
	resp, _, err := c.do(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return false, err
	}
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMovedPermanently {
		return false, nil
	}
	if !is2xx(resp.StatusCode) {
		return false, statusErr(resp.StatusCode, path)
	}
	return true, nil
}

// ListRepos lists every repository owner owns, choosing the endpoint by account
// type and token kind.
func (c Client) ListRepos(owner domain.Owner) ([]githubmodel.Repository, error) {
	at, err := c.GetAccountType(owner)
	if err != nil {
		return nil, err
	}
	raw, err := c.listReposRaw(owner, at)
	if err != nil {
		return nil, err
	}
	return decode[githubmodel.Repository](raw)
}

func (c Client) listReposRaw(owner domain.Owner, at domain.AccountType) ([]json.RawMessage, error) {
	if at == domain.AccountTypeOrganization {
		return c.paginate("/orgs/"+string(owner)+"/repos", params("type", "all"), "")
	}
	return c.listUserRepos(owner)
}

func (c Client) listUserRepos(owner domain.Owner) ([]json.RawMessage, error) {
	login, err := c.authenticatedLogin()
	if err != nil {
		return nil, err
	}
	if login != "" && login == string(owner) {
		return c.paginate("/user/repos", params("affiliation", "owner"), "")
	}
	install, ok, err := c.listInstallationRepos(owner)
	if err != nil {
		return nil, err
	}
	if ok {
		return install, nil
	}
	return c.paginate("/users/"+string(owner)+"/repos", params("type", "owner"), "")
}

func (c Client) listInstallationRepos(owner domain.Owner) ([]json.RawMessage, bool, error) {
	raw, err := c.paginate("/installation/repositories", params(), "repositories")
	if err != nil {
		if errors.Is(err, adminerr.ErrHTTPStatus) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return filterByOwner(raw, owner), true, nil
}

func (c Client) authenticatedLogin() (string, error) {
	resp, body, err := c.do(http.MethodGet, c.baseURL+"/user", nil)
	if err != nil {
		return "", err
	}
	if !is2xx(resp.StatusCode) {
		return "", nil
	}
	var u struct {
		Login string `json:"login"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return "", decodeErr(err)
	}
	return u.Login, nil
}

// ListWorkflowRuns lists completed runs, optionally filtered to those created
// before the given date (the GitHub created=<YYYY-MM-DD syntax).
func (c Client) ListWorkflowRuns(
	owner domain.Owner,
	repo domain.RepoName,
	before domain.CreatedBefore,
) ([]githubmodel.WorkflowRun, error) {
	path := fmt.Sprintf("/repos/%s/%s/actions/runs", owner, repo)
	raw, err := c.paginate(path, runParams(before), "workflow_runs")
	if err != nil {
		return nil, err
	}
	return decode[githubmodel.WorkflowRun](raw)
}

// DeleteWorkflowRun deletes a single workflow run by id.
func (c Client) DeleteWorkflowRun(owner domain.Owner, repo domain.RepoName, id domain.RunID) error {
	path := fmt.Sprintf("/repos/%s/%s/actions/runs/%d", owner, repo, id)
	resp, _, err := c.do(http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	if !is2xx(resp.StatusCode) {
		return statusErr(resp.StatusCode, path)
	}
	return nil
}

// paginate walks Link rel="next" pages, collecting raw items from each. When
// itemsKey is non-empty the page body is an object whose itemsKey field is the
// array; otherwise the body is the array itself.
func (c Client) paginate(path string, query url.Values, itemsKey string) ([]json.RawMessage, error) {
	var out []json.RawMessage
	next := c.baseURL + path
	for next != "" {
		items, link, err := c.page(next, query, itemsKey)
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
		next = link
		query = nil
	}
	return out, nil
}

func (c Client) page(rawurl string, query url.Values, itemsKey string) ([]json.RawMessage, string, error) {
	resp, body, err := c.do(http.MethodGet, rawurl, query)
	if err != nil {
		return nil, "", err
	}
	if !is2xx(resp.StatusCode) {
		return nil, "", statusErr(resp.StatusCode, rawurl)
	}
	items, err := extractItems(body, itemsKey)
	if err != nil {
		return nil, "", err
	}
	return items, nextLink(resp.Header.Get("Link")), nil
}

func (c Client) getJSON(path string, v any) error {
	resp, body, err := c.do(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	if !is2xx(resp.StatusCode) {
		return statusErr(resp.StatusCode, path)
	}
	if err := json.Unmarshal(body, v); err != nil {
		return decodeErr(err)
	}
	return nil
}

func (c Client) do(method, rawurl string, query url.Values) (*http.Response, []byte, error) {
	full := rawurl
	if query != nil {
		full = rawurl + "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(context.Background(), method, full, nil)
	if err != nil {
		return nil, nil, adminerr.ErrHTTPStatus.With(err, "url", full)
	}
	setHeaders(req, c.token)
	resp, err := c.doer.Do(req)
	if err != nil {
		return nil, nil, adminerr.ErrHTTPStatus.With(err, "url", full)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, adminerr.ErrHTTPStatus.With(err, "url", full)
	}
	return resp, body, nil
}

func setHeaders(req *http.Request, token string) {
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", acceptHeader)
	req.Header.Set("X-GitHub-Api-Version", apiVersion)
}

func extractItems(body []byte, itemsKey string) ([]json.RawMessage, error) {
	if itemsKey == "" {
		return unmarshalArray(body)
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil, decodeErr(err)
	}
	return unmarshalArray(obj[itemsKey])
}

func unmarshalArray(body []byte) ([]json.RawMessage, error) {
	var arr []json.RawMessage
	if err := json.Unmarshal(body, &arr); err != nil {
		return nil, decodeErr(err)
	}
	return arr, nil
}

func filterByOwner(raw []json.RawMessage, owner domain.Owner) []json.RawMessage {
	out := make([]json.RawMessage, 0, len(raw))
	for _, r := range raw {
		var probe struct {
			Owner struct {
				Login string `json:"login"`
			} `json:"owner"`
		}
		if json.Unmarshal(r, &probe) == nil && probe.Owner.Login == string(owner) {
			out = append(out, r)
		}
	}
	return out
}

func decode[T any](raw []json.RawMessage) ([]T, error) {
	out := make([]T, 0, len(raw))
	for _, r := range raw {
		var v T
		if err := json.Unmarshal(r, &v); err != nil {
			return nil, decodeErr(err)
		}
		out = append(out, v)
	}
	return out, nil
}

func runParams(before domain.CreatedBefore) url.Values {
	v := params("status", "completed")
	if before != "" {
		v.Set("created", "<"+string(before))
	}
	return v
}

func params(pairs ...string) url.Values {
	v := url.Values{}
	v.Set("per_page", perPage)
	for i := 0; i+1 < len(pairs); i += 2 {
		v.Set(pairs[i], pairs[i+1])
	}
	return v
}

func nextLink(header string) string {
	m := linkNextRe.FindStringSubmatch(header)
	if m == nil {
		return ""
	}
	return m[1]
}

func is2xx(code int) bool { return code >= 200 && code < 300 }

func statusErr(code int, where string) error {
	return adminerr.ErrHTTPStatus.With(nil, "status", code, "url", where)
}

func decodeErr(err error) error { return adminerr.ErrDecodeResponse.With(err) }
