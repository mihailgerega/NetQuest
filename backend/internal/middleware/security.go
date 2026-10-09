package middleware

import (
	"net/http"
	"strings"
)

// Значения CORS по умолчанию.
const (
	defaultCORSMethods       = "GET,POST,PATCH,DELETE,OPTIONS"
	defaultCORSHeaders       = "Authorization,Content-Type,Idempotency-Key,X-Request-ID"
	defaultCORSExposeHeaders = "X-Request-ID"
)

// SecureHeaders ставит заголовки, которые браузер применяет к ответу API:
// запрет угадывать Content-Type, встраивать ответ во frame, отдавать Referer
// на чужие сайты, использовать камеру и геолокацию, а также строгий CSP.
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("X-Frame-Options", "DENY")
		header.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		header.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		header.Set("Cross-Origin-Resource-Policy", "same-site")
		header.Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")

		next.ServeHTTP(w, r)
	})
}

// CORSConfig — какие сайты и как могут обращаться к API из браузера.
type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string // пусто — defaultCORSMethods
	AllowedHeaders []string // пусто — defaultCORSHeaders
	ExposeHeaders  []string // пусто — defaultCORSExposeHeaders
	// AllowCredentials — разрешить cookie: refresh-токен ходит в httpOnly-cookie,
	// поэтому фронтенду на другом origin это нужно.
	AllowCredentials bool
}

// CORS отвечает браузеру, можно ли странице с Origin читать ответы API.
//
// Разрешённый origin получает CORS-заголовки, с Vary: Origin — чтобы кеши
// не отдали ответ с чужим Access-Control-Allow-Origin. Любой preflight
// (OPTIONS) получает 204 сразу, до аутентификации; для чужого origin — без
// CORS-заголовков, и браузер сам заблокирует запрос.
func CORS(cfg CORSConfig) Middleware {
	allowed := make(map[string]struct{}, len(cfg.AllowedOrigins))
	for _, origin := range cfg.AllowedOrigins {
		allowed[origin] = struct{}{}
	}

	methods := joinOrDefault(cfg.AllowedMethods, defaultCORSMethods)
	headers := joinOrDefault(cfg.AllowedHeaders, defaultCORSHeaders)
	expose := joinOrDefault(cfg.ExposeHeaders, defaultCORSExposeHeaders)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if _, ok := allowed[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", methods)
				w.Header().Set("Access-Control-Allow-Headers", headers)
				w.Header().Set("Access-Control-Expose-Headers", expose)

				if cfg.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// joinOrDefault склеивает список через запятую или возвращает значение по умолчанию.
func joinOrDefault(values []string, fallback string) string {
	if len(values) == 0 {
		return fallback
	}

	return strings.Join(values, ",")
}
