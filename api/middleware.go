package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/hypertf/nahcloud/domain"
	"github.com/hypertf/nahcloud/service"
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

// AuthMiddleware creates middleware that validates org tokens for API routes
func AuthMiddleware(svc *service.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeAuthError(w, "missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				writeAuthError(w, "invalid authorization header format")
				return
			}

			token := parts[1]
			org, err := svc.GetOrganizationByToken(token)
			if err != nil {
				if domain.IsNotFound(err) || domain.IsUnauthorized(err) {
					writeAuthError(w, "invalid token")
					return
				}
				writeServerError(w)
				return
			}

			ctx := context.WithValue(r.Context(), ContextKeyOrg, org)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeAuthError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":"` + message + `"}`))
}

func writeServerError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(`{"error":"internal server error"}`))
}
