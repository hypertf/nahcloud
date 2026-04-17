package auth

import (
	"context"

	"github.com/hypertf/nahcloud/domain"
)

type contextKey string

const (
	// ContextKeyOrg is the context key for the authenticated organization
	ContextKeyOrg contextKey = "org"
)

// OrgFromContext retrieves the authenticated organization from the request context
func OrgFromContext(ctx context.Context) *domain.Organization {
	org, _ := ctx.Value(ContextKeyOrg).(*domain.Organization)
	return org
}

// WithOrg returns a new context with the organization stored
func WithOrg(ctx context.Context, org *domain.Organization) context.Context {
	return context.WithValue(ctx, ContextKeyOrg, org)
}
