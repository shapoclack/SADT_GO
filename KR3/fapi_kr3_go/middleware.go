package main

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/time/rate"
)

// Защита от тайминг-атак (Задание 6.2)
func secureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// Rate Limiter (Задание 6.5)
var visitors = make(map[string]*rate.Limiter)
var mu sync.Mutex

func getVisitor(ip string, r rate.Limit, b int) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()
	limiter, exists := visitors[ip]
	if !exists {
		limiter = rate.NewLimiter(r, b)
		visitors[ip] = limiter
	}
	return limiter
}

func limitMiddleware(r rate.Limit, b int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			limiter := getVisitor(req.RemoteAddr, r, b)
			if !limiter.Allow() {
				http.Error(w, `{"detail": "Too many requests"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, req)
		})
	}
}

// Basic Auth для документации (Задание 6.3)
func DocsBasicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mode := os.Getenv("MODE")
		if mode == "PROD" {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		user, pass, ok := r.BasicAuth()
		expectedUser := os.Getenv("DOCS_USER")
		expectedPass := os.Getenv("DOCS_PASSWORD")

		if !ok || !secureCompare(user, expectedUser) || !secureCompare(pass, expectedPass) {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// JWT Валидация (Задание 6.4)
func JWTAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, `{"detail": "Missing or invalid token"}`, http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv("JWT_SECRET")), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, `{"detail": "Invalid credentials"}`, http.StatusUnauthorized)
			return
		}

		// В реальном проекте тут мы бы доставали Claims и клали юзера в контекст
		next.ServeHTTP(w, r)
	})
}

// RBAC (Задание 7.1)
// Упрощенная проверка ролей. В реале достаем роль из JWT Claims.
func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Мокаем: предполагаем, что пользователь 'admin' (В реальности брать из токена)
			userRole := "admin" // Заглушка для теста

			allowed := false
			for _, role := range allowedRoles {
				if userRole == role {
					allowed = true
					break
				}
			}

			if !allowed {
				http.Error(w, `{"detail": "Forbidden: insufficient permissions"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
