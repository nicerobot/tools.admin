package githubmodel_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nicerobot/tools.admin/internal/domain"
	"github.com/nicerobot/tools.admin/internal/githubmodel"
)

func TestUserAccountType(t *testing.T) {
	u := githubmodel.User{Login: "myorg", Type: "Organization"}
	assert.Equal(t, domain.AccountTypeOrganization, u.AccountType())
}

func TestRepositoryVisibilityPublic(t *testing.T) {
	r := githubmodel.Repository{Private: false}
	assert.Equal(t, domain.VisibilityPublic, r.Visibility())
}

func TestRepositoryVisibilityPrivate(t *testing.T) {
	r := githubmodel.Repository{Private: true}
	assert.Equal(t, domain.VisibilityPrivate, r.Visibility())
}
