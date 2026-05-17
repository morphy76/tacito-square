// Package auth provides JWT/OIDC authentication middleware
// for Tacito Square API-first design (bearer token auth).
package auth

import (
	"context"
	"errors"
	"strings"
)

// Claims represents the essential claims extracted from a JWT token.
type Claims struct {
	Subject string
	Email   string
	Roles   []string
}

type contextKey string

const claimsKey contextKey = "auth_claims"

// ContextWithClaims returns a new context with the claims stored.
func ContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// ClaimsFromContext extracts claims from the context.
// Returns nil if no claims are present.
func ClaimsFromContext(ctx context.Context) *Claims {
	claims, _ := ctx.Value(claimsKey).(*Claims)
	return claims
}

// ExtractBearerToken extracts the token from an "Authorization: Bearer <token>" header value.
func ExtractBearerToken(authHeader string) (string, error) {
	if authHeader == "" {
		return "", errors.New("missing authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", errors.New("invalid authorization header format, expected 'Bearer <token>'")
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", errors.New("empty bearer token")
	}

	return token, nil
}
