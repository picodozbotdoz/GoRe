package headers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func ParseExpires(expr string) (time.Time, bool) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return time.Time{}, false
	}

	if expr[0] == '@' {
		return parseExpiresAt(expr[1:])
	}

	return parseExpiresRelative(expr)
}

func parseExpiresAt(expr string) (time.Time, bool) {
	parts := strings.SplitN(expr, ":", 2)
	if len(parts) != 2 {
		return time.Time{}, false
	}

	hourStr, minStr := parts[0], parts[1]
	hour, err := strconv.Atoi(hourStr)
	if err != nil || hour < 0 || hour > 23 {
		return time.Time{}, false
	}
	min, err := strconv.Atoi(minStr)
	if err != nil || min < 0 || min > 59 {
		return time.Time{}, false
	}

	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, now.Location()), true
}

func parseExpiresRelative(expr string) (time.Time, bool) {
	if !strings.HasPrefix(expr, "access plus ") {
		return time.Time{}, false
	}
	rest := strings.TrimPrefix(expr, "access plus ")
	rest = strings.TrimSpace(rest)

	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return time.Time{}, false
	}

	var d time.Duration
	for i := 0; i < len(fields); i += 2 {
		if i+1 >= len(fields) {
			return time.Time{}, false
		}
		val, err := strconv.Atoi(fields[i])
		if err != nil {
			return time.Time{}, false
		}
		unit := strings.ToLower(fields[i+1])
		switch unit {
		case "year", "years":
			d += time.Duration(val) * 365 * 24 * time.Hour
		case "month", "months":
			d += time.Duration(val) * 30 * 24 * time.Hour
		case "week", "weeks":
			d += time.Duration(val) * 7 * 24 * time.Hour
		case "day", "days":
			d += time.Duration(val) * 24 * time.Hour
		case "hour", "hours":
			d += time.Duration(val) * time.Hour
		case "minute", "minutes":
			d += time.Duration(val) * time.Minute
		case "second", "seconds":
			d += time.Duration(val) * time.Second
		default:
			return time.Time{}, false
		}
	}

	return time.Now().Add(d), true
}

func NewExpiresHandler(expr string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if t, ok := ParseExpires(expr); ok {
				expiresHeader := t.UTC().Format(http.TimeFormat)
				w.Header().Set("Expires", expiresHeader)

				cacheControl := w.Header().Get("Cache-Control")
				if cacheControl == "" {
					maxAge := int(time.Until(t).Seconds())
					if maxAge < 0 {
						maxAge = 0
					}
					w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", maxAge))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
