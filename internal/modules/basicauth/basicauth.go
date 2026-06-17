package basicauth

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func New(realm string, users map[string]string) func(http.Handler) http.Handler {
	if len(users) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	if realm == "" {
		realm = "Restricted"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth()
			if !ok {
				w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			expectedPass, exists := users[user]
			if !exists {
				w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			if strings.HasPrefix(expectedPass, "$2a$") || strings.HasPrefix(expectedPass, "$2b$") {
				if err := bcrypt.CompareHashAndPassword([]byte(expectedPass), []byte(pass)); err != nil {
					w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
			} else if subtle.ConstantTimeCompare([]byte(pass), []byte(expectedPass)) != 1 {
				w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
