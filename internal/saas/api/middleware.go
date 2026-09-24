package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/auth"
)

type ctxKey int

const claimsKey ctxKey = 1

// AuthMiddleware validates Bearer JWT and injects claims into context.
func AuthMiddleware(authSvc *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tok := bearerToken(r)
			if tok == "" {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			claims, err := authSvc.ParseToken(tok)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRoles rejects requests whose JWT role is not in the allowed set.
func RequireRoles(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c := ClaimsFrom(r.Context())
			if c == nil {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			if _, ok := allowed[c.Role]; !ok {
				writeError(w, http.StatusForbidden, "forbidden")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func ClaimsFrom(ctx context.Context) *auth.Claims {
	v, _ := ctx.Value(claimsKey).(*auth.Claims)
	return v
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
