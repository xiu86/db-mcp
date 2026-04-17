package middleware

import (
	"net/http"
	"strings"
)

type TokenAuth struct {
	tokens map[string]bool
}

func NewTokenAuth(tokens []string) *TokenAuth {
	m := make(map[string]bool)
	for _, t := range tokens {
		m[t] = true
	}
	return &TokenAuth{tokens: m}
}

func (t *TokenAuth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(t.tokens) == 0 {
			next.ServeHTTP(w, r)
			return
		}

		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		if !t.tokens[token] {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
