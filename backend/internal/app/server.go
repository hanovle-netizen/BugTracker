package app

import (
	"TaskTracker/internal/handler"
	"TaskTracker/internal/store/postgres"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

type inMemoryRateLimiter struct {
	mu      sync.Mutex
	entries map[string][]time.Time
	limit   int
	window  time.Duration
}

func newRateLimiter(limit int, window time.Duration) *inMemoryRateLimiter {
	return &inMemoryRateLimiter{
		entries: make(map[string][]time.Time),
		limit:   limit,
		window:  window,
	}
}

func (l *inMemoryRateLimiter) allow(ip string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	h := l.entries[ip]
	cutoff := now.Add(-l.window)
	kept := h[:0]
	for _, t := range h {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= l.limit {
		l.entries[ip] = kept
		return false
	}
	kept = append(kept, now)
	l.entries[ip] = kept
	return true
}

func rateLimitMiddleware(l *inMemoryRateLimiter) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if i := strings.LastIndex(ip, ":"); i > 0 {
				ip = ip[:i]
			}
			if !l.allow(ip) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "rate limit exceeded"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"status", wrapped.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "https://bugtracker.sytes.net")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func AuthMiddleware(jwtSecret string, store *postgres.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized"}`))
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized"}`))
				return
			}

			token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized"}`))
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized"}`))
				return
			}

			userID := int(claims["sub"].(float64))
			tokenVer := int(claims["ver"].(float64))

			_, _, _, _, currentVer, err := store.GetUserByID(r.Context(), userID)
			if err != nil {
				slog.Error("auth middleware: db error", "error", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized"}`))
				return
			}

			if tokenVer < currentVer {
				slog.Warn("outdated token used", "user_id", userID, "token_ver", tokenVer, "db_ver", currentVer)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
				return
			}

			ctx := context.WithValue(r.Context(), "user_id", userID)
			ctx = context.WithValue(ctx, "role", claims["role"].(string))

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func TeacherOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := r.Context().Value("role").(string)
		if !ok || role != "admin" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"not allowed"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func NewRouter(h *handler.Handler, jwtSecret string, store *postgres.Store) http.Handler {
	router := mux.NewRouter().StrictSlash(true)
	auth := AuthMiddleware(jwtSecret, store)
	authRateLimited := router.NewRoute().Subrouter()
	authRateLimited.Use(rateLimitMiddleware(newRateLimiter(15, time.Minute)))

	// Аутентификация
	authRateLimited.Path("/api/users").Methods("POST").HandlerFunc(h.CreateUser)
	authRateLimited.Path("/api/login").Methods("POST").HandlerFunc(h.Login)
	// Пользователь (текущий)
	router.Path("/api/me").Methods("GET").Handler(auth(http.HandlerFunc(h.GetMe)))
	router.Path("/api/me/login").Methods("PATCH").Handler(auth(http.HandlerFunc(h.UpdateEmail)))
	router.Path("/api/me/password").Methods("PATCH").Handler(auth(http.HandlerFunc(h.UpdatePassword)))
	router.Path("/api/me/logout-all").Methods("POST").Handler(auth(http.HandlerFunc(h.LogoutAll)))
	router.Path("/api/users/{id:[0-9]+}").Methods("GET").Handler(auth(http.HandlerFunc(h.GetUserByID)))
	// Управление пользователями (только admin)
	adminOnly := TeacherOnly
	router.Handle("/api/admin/users", auth(adminOnly(http.HandlerFunc(h.AdminGetAllUsers)))).Methods("GET")
	router.Handle("/api/admin/users/{id:[0-9]+}", auth(TeacherOnly(http.HandlerFunc(h.AdminUpdateUserRole)))).Methods("PATCH")
	router.Handle("/api/admin/users/{id:[0-9]+}/password", auth(TeacherOnly(http.HandlerFunc(h.AdminResetPassword)))).Methods("PATCH")
	router.Handle("/api/admin/users/{id:[0-9]+}", auth(TeacherOnly(http.HandlerFunc(h.AdminDeleteUser)))).Methods("DELETE")
	// Группы (Организации)
	router.Path("/api/orgs").Methods("GET").Handler(auth(http.HandlerFunc(h.GetMyOrganizations)))
	router.Handle("/api/orgs", auth(TeacherOnly(http.HandlerFunc(h.CreateOrganization)))).Methods("POST")
	router.Path("/api/orgs/{id:[0-9]+}/members").Methods("GET").Handler(auth(http.HandlerFunc(h.GetOrgMembers)))
	router.Path("/api/orgs/{id:[0-9]+}/members").Methods("POST").Handler(auth(http.HandlerFunc(h.AddOrgMember)))
	router.Path("/api/orgs/{id:[0-9]+}/members/{userId:[0-9]+}").Methods("PATCH").Handler(auth(http.HandlerFunc(h.UpdateOrgMemberRole)))
	router.Path("/api/orgs/{id:[0-9]+}/members/{userId:[0-9]+}").Methods("DELETE").Handler(auth(http.HandlerFunc(h.RemoveOrgMember)))
	// Проекты
	router.Path("/api/projects").Methods("GET").Queries("org_id", "{org_id:[0-9]+}").Handler(auth(http.HandlerFunc(h.GetProjects)))
	router.Handle("/api/projects", auth(http.HandlerFunc(h.CreateProject))).Methods("POST")
	router.Path("/api/projects/{id:[0-9]+}/members").Methods("GET").Handler(auth(http.HandlerFunc(h.GetProjectMembers)))
	router.Path("/api/projects/{id:[0-9]+}/members").Methods("POST").Handler(auth(http.HandlerFunc(h.AddProjectMember)))
	router.Path("/api/projects/{id:[0-9]+}/members/{userId:[0-9]+}").Methods("PATCH").Handler(auth(http.HandlerFunc(h.UpdateProjectMember)))
	router.Path("/api/projects/{id:[0-9]+}/members/{userId:[0-9]+}").Methods("DELETE").Handler(auth(http.HandlerFunc(h.RemoveProjectMember)))
	// Задачи
	router.Path("/api/tasks").Methods("GET").Queries("project_id", "{project_id:[0-9]+}").Handler(auth(http.HandlerFunc(h.GetTasks)))
	router.Path("/api/tasks").Methods("POST").Handler(auth(http.HandlerFunc(h.CreateTask)))
	router.Path("/api/tasks/{id:[0-9]+}").Methods("PATCH").Handler(auth(http.HandlerFunc(h.UpdateTask)))
	router.Path("/api/tasks/{id:[0-9]+}").Methods("DELETE").Handler(auth(http.HandlerFunc(h.DeleteTask)))
	router.Path("/api/tasks/{id:[0-9]+}/photos").Methods("POST").Handler(auth(http.HandlerFunc(h.AddTaskPhoto)))
	router.Path("/api/tasks/{id:[0-9]+}/photos").Methods("GET").Handler(auth(http.HandlerFunc(h.GetTaskPhotos)))
	router.Path("/api/tasks/{id:[0-9]+}/photos/{photo_id:[0-9]+}").Methods("DELETE").Handler(auth(http.HandlerFunc(h.DeleteTaskPhoto)))
	router.Path("/api/tasks/{id:[0-9]+}/comments").Methods("GET").Handler(auth(http.HandlerFunc(h.GetComments)))
	router.Path("/api/tasks/{id:[0-9]+}/comments").Methods("POST").Handler(auth(http.HandlerFunc(h.CreateComment)))
	router.Path("/api/tasks/{id:[0-9]+}/audit").Methods("GET").Handler(auth(http.HandlerFunc(h.GetTaskAudit)))
	router.Path("/api/tasks/{id:[0-9]+}/relations").Methods("GET").Handler(auth(http.HandlerFunc(h.GetTaskRelations)))
	router.Path("/api/tasks/{id:[0-9]+}/relations").Methods("POST").Handler(auth(http.HandlerFunc(h.CreateTaskRelation)))
	router.Path("/api/relations/{rel_id:[0-9]+}").Methods("DELETE").Handler(auth(http.HandlerFunc(h.DeleteRelation)))
	router.Path("/api/tasks/{id:[0-9]+}/tags").Methods("GET").Handler(auth(http.HandlerFunc(h.GetTaskTags)))
	router.Path("/api/tasks/{id:[0-9]+}/tags").Methods("PUT").Handler(auth(http.HandlerFunc(h.ReplaceTaskTags)))
	router.Path("/api/templates").Methods("GET").Handler(auth(http.HandlerFunc(h.GetTemplates)))
	router.Handle("/api/templates", auth(TeacherOnly(http.HandlerFunc(h.CreateTemplate)))).Methods("POST")
	router.Handle("/api/templates/{id:[0-9]+}", auth(TeacherOnly(http.HandlerFunc(h.DeleteTemplate)))).Methods("DELETE")
	router.Path("/api/stats").Methods("GET").Handler(auth(http.HandlerFunc(h.GetStats)))
	router.Path("/api/stats/tasks/{task_id:[0-9]+}").Methods("GET").Handler(auth(http.HandlerFunc(h.GetTaskStats)))
	// Баги
	router.Path("/api/bugs/{task_id:[0-9]+}").Methods("GET").Handler(auth(http.HandlerFunc(h.GetBug)))
	router.Path("/api/bugs/{task_id:[0-9]+}").Methods("POST").Handler(auth(http.HandlerFunc(h.CreateBug)))
	router.Path("/api/bugs/{bug_id:[0-9]+}").Methods("PATCH").Handler(auth(http.HandlerFunc(h.UpdateBug)))
	router.Path("/api/bugs/{bug_id:[0-9]+}").Methods("DELETE").Handler(auth(http.HandlerFunc(h.DeleteBug)))
	router.Path("/api/bugs/{bug_id:[0-9]+}/comments").Methods("GET").Handler(auth(http.HandlerFunc(h.GetBugComments)))
	router.Path("/api/bugs/{bug_id:[0-9]+}/comments").Methods("POST").Handler(auth(http.HandlerFunc(h.CreateBugComment)))
	router.Path("/api/bugs/{bug_id:[0-9]+}/photos").Methods("GET").Handler(auth(http.HandlerFunc(h.GetBugPhotos)))
	router.Path("/api/bugs/{bug_id:[0-9]+}/photos").Methods("POST").Handler(auth(http.HandlerFunc(h.AddBugPhoto)))
	router.Path("/api/bugs/{bug_id:[0-9]+}/photos/{photo_id:[0-9]+}").Methods("DELETE").Handler(auth(http.HandlerFunc(h.DeleteBugPhoto)))

	// Чат
	router.Path("/api/chat/threads").Methods("POST").Handler(auth(http.HandlerFunc(h.CreateChatThread)))
	router.Path("/api/chat/threads").Methods("GET").Handler(auth(http.HandlerFunc(h.GetThreads)))
	router.Path("/api/chat/threads/{id:[0-9]+}/messages").Methods("GET").Handler(auth(http.HandlerFunc(h.GetMessages)))
	router.Path("/api/chat/threads/{id:[0-9]+}/messages").Methods("POST").Handler(auth(http.HandlerFunc(h.SendMessage)))
	router.Path("/api/chat/messages/{id:[0-9]+}").Methods("PATCH").Handler(auth(http.HandlerFunc(h.EditChatMessage)))
	router.Path("/api/chat/messages/{id:[0-9]+}").Methods("DELETE").Handler(auth(http.HandlerFunc(h.DeleteChatMessage)))
	router.Path("/api/chat/threads/{id:[0-9]+}/read").Methods("POST").Handler(auth(http.HandlerFunc(h.MarkChatRead)))
	router.Path("/api/chat/threads/{id:[0-9]+}/typing").Methods("GET").Handler(auth(http.HandlerFunc(h.GetTyping)))
	router.Path("/api/chat/threads/{id:[0-9]+}/typing").Methods("POST").Handler(auth(http.HandlerFunc(h.ReportTyping)))
	router.Path("/api/healthz").Methods("GET").HandlerFunc(h.HealthCheck)
	router.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	return CORSMiddleware(loggingMiddleware(router))
}
