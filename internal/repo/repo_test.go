package repo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestScalars pins the constant vocabulary so a rename of any value is a
// deliberate, reviewed change rather than a silent drift.
func TestScalars(t *testing.T) {
	t.Parallel()
	want := assert.New(t)

	want.Equal(AccountType("Organization"), AccountTypeOrganization)
	want.Equal(CommentSource("org"), CommentSourceOrg)
	want.Equal(CommentSource("account"), CommentSourceAccount)
	want.Equal(Visibility("public"), VisibilityPublic)
	want.Equal(Visibility("private"), VisibilityPrivate)
}
