package github_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nicerobot/tools.admin/internal/github"
	"github.com/nicerobot/tools.admin/internal/repo"
)

func TestUserAccountType(t *testing.T) {
	u := github.User{Login: "myorg", Type: "Organization"}
	assert.Equal(t, repo.AccountTypeOrganization, u.AccountType())
}

func TestRepositoryVisibilityPublic(t *testing.T) {
	r := github.Repository{IsPrivate: false}
	assert.Equal(t, repo.VisibilityPublic, r.Visibility())
}

func TestRepositoryVisibilityPrivate(t *testing.T) {
	r := github.Repository{IsPrivate: true}
	assert.Equal(t, repo.VisibilityPrivate, r.Visibility())
}
