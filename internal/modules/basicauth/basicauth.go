package basicauth

import (
	"crypto/subtle"
	"net/http"
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
			if !exists || subtle.ConstantTimeCompare([]byte(pass), []byte(expectedPass)) != 1 {
				w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
