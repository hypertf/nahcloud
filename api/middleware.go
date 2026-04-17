package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/hypertf/nahcloud/domain"
	"github.com/hypertf/nahcloud/pkg/auth"
	"github.com/hypertf/nahcloud/service"
)

const (
	// Cookie names
	CookieSession  = "nah_session"
	CookieAPIToken = "nah_api_token"

	// Cookie expiration
	SessionMaxAge  = 30 * 24 * time.Hour  // 30 days
	APITokenMaxAge = 365 * 24 * time.Hour // 1 year
)

// OrgFromContext retrieves the authenticated organization from the request context
func OrgFromContext(ctx context.Context) *domain.Organization {
	return auth.OrgFromContext(ctx)
}

// AuthMiddleware creates middleware that validates org tokens for API routes
// It checks Authorization header first, then falls back to cookie
// If no valid token found, it auto-creates a new org and sets a cookie
func AuthMiddleware(svc *service.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var token string

			// Try Authorization header first
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
					token = parts[1]
				}
			}

			// Fall back to cookie if no header
			if token == "" {
				if cookie, err := r.Cookie(CookieAPIToken); err == nil {
					token = cookie.Value
				}
			}

			// If we have a token, validate it
			if token != "" {
				org, err := svc.GetOrganizationByToken(token)
				if err == nil {
					ctx := auth.WithOrg(r.Context(), org)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
				// Invalid token - clear cookie if it was a cookie-based auth
				if authHeader == "" {
					http.SetCookie(w, &http.Cookie{
						Name:     CookieAPIToken,
						Value:    "",
						Path:     "/v1",
						MaxAge:   -1,
						HttpOnly: true,
						SameSite: http.SameSiteLaxMode,
					})
				}
				// If it was header auth, return error
				if authHeader != "" {
					if domain.IsNotFound(err) || domain.IsUnauthorized(err) {
						writeAuthError(w, "invalid token")
						return
					}
					writeServerError(w)
					return
				}
			}

			// No valid token - auto-create org for API requests
			orgWithKey, err := svc.CreateOrganizationWithAPIKey()
			if err != nil {
				writeServerError(w)
				return
			}

			// Set API token cookie
			http.SetCookie(w, &http.Cookie{
				Name:     CookieAPIToken,
				Value:    orgWithKey.APIKey.Token,
				Path:     "/v1",
				MaxAge:   int(APITokenMaxAge.Seconds()),
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
				// Secure: true, // Enable in production with HTTPS
			})

			// Also return the token in the response header for clients to save
			w.Header().Set("X-API-Token", orgWithKey.APIKey.Token)

			ctx := auth.WithOrg(r.Context(), &orgWithKey.Organization)
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

// WebSessionMiddleware creates middleware for web routes that handles session-based auth
// If no valid session, it creates a new org and session automatically
func WebSessionMiddleware(svc *service.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to get session from cookie
			cookie, err := r.Cookie(CookieSession)
			if err == nil && cookie.Value != "" {
				// Validate session token
				org, err := svc.GetOrganizationBySessionToken(cookie.Value)
				if err == nil {
					// Valid session, set org in context and continue
					ctx := auth.WithOrg(r.Context(), org)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
				// Invalid/expired session, clear the cookie
				http.SetCookie(w, &http.Cookie{
					Name:     CookieSession,
					Value:    "",
					Path:     "/",
					MaxAge:   -1,
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
				})
			}

			// No valid session - create new org and session
			orgWithSession, err := svc.CreateOrganizationWithSession()
			if err != nil {
				writeServerError(w)
				return
			}

			// Set session cookie
			http.SetCookie(w, &http.Cookie{
				Name:     CookieSession,
				Value:    orgWithSession.Session.Token,
				Path:     "/",
				MaxAge:   int(SessionMaxAge.Seconds()),
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
				// Secure: true, // Enable in production with HTTPS
			})

			// Set org in context and continue
			ctx := auth.WithOrg(r.Context(), &orgWithSession.Organization)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
