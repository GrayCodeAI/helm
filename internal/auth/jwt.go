// Package auth provides token-based authentication and RBAC authorization.
package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Role represents user roles
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleUser   Role = "user"
	RoleViewer Role = "viewer"
)

// Permission represents a permission
type Permission string

const (
	PermSessionRead   Permission = "session:read"
	PermSessionWrite  Permission = "session:write"
	PermSessionDelete Permission = "session:delete"
	PermCostRead      Permission = "cost:read"
	PermMemoryRead    Permission = "memory:read"
	PermMemoryWrite   Permission = "memory:write"
	PermConfigRead    Permission = "config:read"
	PermConfigWrite   Permission = "config:write"
	PermAdmin         Permission = "admin"
)

// RolePermissions maps roles to permissions
var RolePermissions = map[Role][]Permission{
	RoleAdmin: {
		PermSessionRead, PermSessionWrite, PermSessionDelete,
		PermCostRead, PermMemoryRead, PermMemoryWrite,
		PermConfigRead, PermConfigWrite, PermAdmin,
	},
	RoleUser: {
		PermSessionRead, PermSessionWrite,
		PermCostRead, PermMemoryRead, PermMemoryWrite,
		PermConfigRead,
	},
	RoleViewer: {
		PermSessionRead, PermCostRead, PermMemoryRead, PermConfigRead,
	},
}

// Claims represents token claims
type Claims struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Roles     []Role `json:"roles"`
	ProjectID string `json:"project_id"`
	ExpiresAt int64  `json:"expires_at"`
	IssuedAt  int64  `json:"issued_at"`
	Issuer    string `json:"issuer"`
}

// Manager manages tokens and RBAC
type Manager struct {
	secret        []byte
	issuer        string
	expiry        time.Duration
	refreshExpiry time.Duration
	users         map[string]*User
	tokens        map[string]*Claims
	mu            sync.RWMutex
}

// User represents a user
type User struct {
	ID       string
	Username string
	Email    string
	Password string
	Roles    []Role
	Active   bool
	Created  time.Time
}

// NewManager creates a new auth manager
func NewManager(secret string, issuer string) *Manager {
	if secret == "" {
		secret = generateSecret()
	}
	return &Manager{
		secret:        []byte(secret),
		issuer:        issuer,
		expiry:        24 * time.Hour,
		refreshExpiry: 7 * 24 * time.Hour,
		users:         make(map[string]*User),
		tokens:        make(map[string]*Claims),
	}
}

// GenerateToken generates a token
func (m *Manager) GenerateToken(user *User) (string, error) {
	now := time.Now().Unix()
	claims := Claims{
		UserID:    user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Roles:     user.Roles,
		ProjectID: "",
		ExpiresAt: now + int64(m.expiry.Seconds()),
		IssuedAt:  now,
		Issuer:    m.issuer,
	}

	// Create token: header.payload.signature
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, _ := json.Marshal(claims)
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payload)

	// Sign
	signature := sign(header+"."+payloadEncoded, m.secret)

	token := header + "." + payloadEncoded + "." + signature

	// Store token
	m.mu.Lock()
	m.tokens[token] = &claims
	m.mu.Unlock()

	return token, nil
}

// GenerateRefreshToken generates a refresh token
func (m *Manager) GenerateRefreshToken(user *User) (string, error) {
	now := time.Now().Unix()
	claims := Claims{
		UserID:    user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Roles:     user.Roles,
		ExpiresAt: now + int64(m.refreshExpiry.Seconds()),
		IssuedAt:  now,
		Issuer:    m.issuer,
	}

	payload, _ := json.Marshal(claims)
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payload)
	signature := sign(payloadEncoded, m.secret)

	token := "refresh." + payloadEncoded + "." + signature

	m.mu.Lock()
	m.tokens[token] = &claims
	m.mu.Unlock()

	return token, nil
}

// ValidateToken validates a token
func (m *Manager) ValidateToken(tokenString string) (*Claims, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	// Verify signature
	signature := sign(parts[0]+"."+parts[1], m.secret)
	if !hmac.Equal([]byte(signature), []byte(parts[2])) {
		return nil, fmt.Errorf("invalid signature")
	}

	// Decode payload
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid payload: %w", err)
	}

	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("invalid claims: %w", err)
	}

	// Check expiration
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, fmt.Errorf("token expired")
	}

	return &claims, nil
}

// RevokeToken revokes a token
func (m *Manager) RevokeToken(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tokens, token)
}

// HasPermission checks if user has permission
func (m *Manager) HasPermission(roles []Role, perm Permission) bool {
	for _, role := range roles {
		permissions := RolePermissions[role]
		for _, p := range permissions {
			if p == perm || p == PermAdmin {
				return true
			}
		}
	}
	return false
}

// Middleware returns authentication middleware
func (m *Manager) Middleware(requiredPerms ...Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				http.Error(w, `{"error":"invalid authorization header"}`, http.StatusUnauthorized)
				return
			}

			claims, err := m.ValidateToken(parts[1])
			if err != nil {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}

			// Check permissions
			for _, perm := range requiredPerms {
				if !m.HasPermission(claims.Roles, perm) {
					http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
					return
				}
			}

			ctx := context.WithValue(r.Context(), "claims", claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AddUser adds a user
func (m *Manager) AddUser(user *User) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[user.ID] = user
}

// GetUser gets a user by ID
func (m *Manager) GetUser(id string) (*User, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	user, ok := m.users[id]
	return user, ok
}

// ListUsers lists all users
func (m *Manager) ListUsers() []*User {
	m.mu.RLock()
	defer m.mu.RUnlock()
	users := make([]*User, 0, len(m.users))
	for _, u := range m.users {
		users = append(users, u)
	}
	return users
}

// RemoveUser removes a user
func (m *Manager) RemoveUser(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.users, id)
}

// UpdateRoles updates user roles
func (m *Manager) UpdateRoles(userID string, roles []Role) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	user, ok := m.users[userID]
	if !ok {
		return fmt.Errorf("user not found: %s", userID)
	}
	user.Roles = roles
	return nil
}

// Helper functions

func sign(data string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

func generateSecret() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

// ClaimsFromContext extracts claims from context
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value("claims").(*Claims)
	return claims, ok
}

// UserIDFromContext extracts user ID from context
func UserIDFromContext(ctx context.Context) (string, bool) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return "", false
	}
	return claims.UserID, true
}
