package httpapi

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"leadqualifier/internal/auth"
	"leadqualifier/internal/models"
)

type ctxKey string

const ctxAccount ctxKey = "account"

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		public := strings.HasPrefix(path, "/api/widget") || strings.HasPrefix(path, "/api/v1") || strings.HasPrefix(path, "/widget/")
		origin := r.Header.Get("Origin")
		if public {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		} else if origin != "" && origin == s.Cfg.DashboardOrigin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) authAccountID(r *http.Request) (int64, bool) {
	c, err := r.Cookie("lq_session")
	if err != nil || c.Value == "" {
		return 0, false
	}
	id, err := auth.VerifyJWT(s.Cfg.JWTSecret, c.Value)
	if err != nil {
		return 0, false
	}
	return id, true
}

func (s *Server) requireAuth(next func(w http.ResponseWriter, r *http.Request, acct *models.Account)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := s.authAccountID(r)
		if !ok {
			errJSON(w, http.StatusUnauthorized, "not authenticated")
			return
		}
		acct, err := s.Store.GetAccountByID(id)
		if err != nil {
			errJSON(w, http.StatusUnauthorized, "account not found")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), ctxAccount, acct)), acct)
	}
}

func (s *Server) requireSecretKey(next func(w http.ResponseWriter, r *http.Request, acct *models.Account)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.Cfg.Features.PublicAPI {
			errJSON(w, http.StatusForbidden, "public API disabled")
			return
		}
		h := r.Header.Get("Authorization")
		key := strings.TrimPrefix(h, "Bearer ")
		if key == "" || key == h {
			errJSON(w, http.StatusUnauthorized, "missing Authorization: Bearer <secret_key>")
			return
		}
		if !s.Limiter.Allow("api:"+key, 10, 30) {
			errJSON(w, http.StatusTooManyRequests, "rate limited")
			return
		}
		acct, err := s.Store.GetAccountBySecretKey(key)
		if err != nil {
			errJSON(w, http.StatusUnauthorized, "invalid API key")
			return
		}
		if !s.Cfg.Plan(acct.Plan).API {
			errJSON(w, http.StatusForbidden, "your plan does not include API access")
			return
		}
		next(w, r, acct)
	}
}

func (s *Server) setSessionCookie(w http.ResponseWriter, accountID int64) error {
	token, err := auth.IssueJWT(s.Cfg.JWTSecret, accountID, 30*24*time.Hour)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "lq_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.Cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   30 * 24 * 3600,
	})
	return nil
}

func clientIP(r *http.Request) string {
	if xf := r.Header.Get("X-Forwarded-For"); xf != "" {
		parts := strings.Split(xf, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func errJSON(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func readJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		errJSON(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return false
	}
	return true
}
