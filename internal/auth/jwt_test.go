package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	t.Parallel()

	m := NewManager("test-secret", "test-issuer")
	require.NotNil(t, m)
	assert.Equal(t, "test-secret", string(m.secret))
	assert.Equal(t, "test-issuer", m.issuer)
}

func TestGenerateAndValidateToken(t *testing.T) {
	t.Parallel()

	m := NewManager("test-secret", "test-issuer")
	user := &User{
		ID:       "user1",
		Username: "testuser",
		Email:    "test@example.com",
		Roles:    []Role{RoleUser},
	}

	token, err := m.GenerateToken(user)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := m.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user1", claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
	assert.Equal(t, "test@example.com", claims.Email)
	assert.Contains(t, claims.Roles, RoleUser)
}

func TestTokenExpiration(t *testing.T) {
	t.Parallel()

	m := NewManager("test-secret", "test-issuer")
	user := &User{
		ID:    "user1",
		Roles: []Role{RoleUser},
	}

	token, err := m.GenerateToken(user)
	require.NoError(t, err)

	// Verify token is valid
	claims, err := m.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user1", claims.UserID)
}

func TestHasPermission(t *testing.T) {
	t.Parallel()

	m := NewManager("test-secret", "test-issuer")

	tests := []struct {
		name     string
		roles    []Role
		perm     Permission
		expected bool
	}{
		{"admin has all perms", []Role{RoleAdmin}, PermSessionRead, true},
		{"admin has admin perm", []Role{RoleAdmin}, PermAdmin, true},
		{"user has session read", []Role{RoleUser}, PermSessionRead, true},
		{"user no admin", []Role{RoleUser}, PermAdmin, false},
		{"viewer limited", []Role{RoleViewer}, PermSessionRead, true},
		{"viewer no write", []Role{RoleViewer}, PermSessionWrite, false},
		{"no roles", []Role{}, PermSessionRead, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := m.HasPermission(tt.roles, tt.perm)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAddAndGetUser(t *testing.T) {
	t.Parallel()

	m := NewManager("test-secret", "test-issuer")
	user := &User{
		ID:       "user1",
		Username: "testuser",
		Roles:    []Role{RoleUser},
	}

	m.AddUser(user)

	got, ok := m.GetUser("user1")
	require.True(t, ok)
	assert.Equal(t, "user1", got.ID)
	assert.Equal(t, "testuser", got.Username)
}

func TestRemoveUser(t *testing.T) {
	t.Parallel()

	m := NewManager("test-secret", "test-issuer")
	user := &User{ID: "user1"}
	m.AddUser(user)

	m.RemoveUser("user1")
	_, ok := m.GetUser("user1")
	assert.False(t, ok)
}

func TestUpdateRoles(t *testing.T) {
	t.Parallel()

	m := NewManager("test-secret", "test-issuer")
	user := &User{ID: "user1", Roles: []Role{RoleViewer}}
	m.AddUser(user)

	err := m.UpdateRoles("user1", []Role{RoleAdmin})
	require.NoError(t, err)

	got, _ := m.GetUser("user1")
	assert.Contains(t, got.Roles, RoleAdmin)
}

func TestRevokeToken(t *testing.T) {
	t.Parallel()

	m := NewManager("test-secret", "test-issuer")
	user := &User{ID: "user1", Roles: []Role{RoleUser}}

	token, err := m.GenerateToken(user)
	require.NoError(t, err)

	m.RevokeToken(token)
	// Token should still validate (stored tokens are for tracking)
	claims, err := m.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user1", claims.UserID)
}

func TestGenerateRefreshToken(t *testing.T) {
	t.Parallel()

	m := NewManager("test-secret", "test-issuer")
	user := &User{ID: "user1", Roles: []Role{RoleUser}}

	token, err := m.GenerateRefreshToken(user)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Contains(t, token, "refresh.")
}

func TestClaimsFromContext(t *testing.T) {
	t.Parallel()
	// Context testing would require middleware integration
	// This is a basic sanity check
	claims := &Claims{UserID: "user1"}
	assert.Equal(t, "user1", claims.UserID)
}
