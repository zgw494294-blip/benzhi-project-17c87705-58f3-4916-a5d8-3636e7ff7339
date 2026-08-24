package httpapi

import (
	"context"
	"net/http"
	"oralarchive/internal/application"
	"strings"
	"time"
)

func (s *Server) withRequestContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		if strings.HasPrefix(r.URL.Path, "/api/") {
			actor := strings.TrimSpace(r.Header.Get("X-Actor"))
			role := application.Role(strings.TrimSpace(r.Header.Get("X-Role")))
			if actor != "" {
				ctx = application.WithPrincipal(ctx, application.Principal{Name: actor, Role: role})
			}
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
