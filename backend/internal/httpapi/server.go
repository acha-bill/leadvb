package httpapi

import (
	"net/http"

	"leadqualifier/internal/ai"
	"leadqualifier/internal/config"
	"leadqualifier/internal/ratelimit"
	"leadqualifier/internal/store"
)

type Server struct {
	Cfg     *config.Config
	Store   *store.Store
	Engine  *ai.Engine
	Hub     *Hub
	Limiter *ratelimit.Limiter
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	mux.HandleFunc("POST /api/auth/signup", s.handleSignup)
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", s.handleLogout)
	mux.HandleFunc("GET /api/auth/me", s.requireAuth(s.handleMe))

	mux.HandleFunc("GET /api/icp", s.requireAuth(s.handleGetICP))
	mux.HandleFunc("PUT /api/icp", s.requireAuth(s.handlePutICP))
	mux.HandleFunc("GET /api/routing", s.requireAuth(s.handleGetRouting))
	mux.HandleFunc("PUT /api/routing", s.requireAuth(s.handlePutRouting))
	mux.HandleFunc("POST /api/routing/test", s.requireAuth(s.handleTestRouting))
	mux.HandleFunc("GET /api/widget-config", s.requireAuth(s.handleGetWidgetConfig))
	mux.HandleFunc("PUT /api/widget-config", s.requireAuth(s.handlePutWidgetConfig))
	mux.HandleFunc("GET /api/settings", s.requireAuth(s.handleGetSettings))
	mux.HandleFunc("PUT /api/settings", s.requireAuth(s.handlePutSettings))
	mux.HandleFunc("GET /api/apikeys", s.requireAuth(s.handleGetKeys))
	mux.HandleFunc("POST /api/apikeys/rotate", s.requireAuth(s.handleRotateKeys))
	mux.HandleFunc("GET /api/widget-domains", s.requireAuth(s.handleGetWidgetDomains))
	mux.HandleFunc("PUT /api/widget-domains", s.requireAuth(s.handlePutWidgetDomains))

	mux.HandleFunc("GET /api/conversations", s.requireAuth(s.handleListConversations))
	mux.HandleFunc("GET /api/conversations/{id}", s.requireAuth(s.handleGetConversation))
	mux.HandleFunc("POST /api/conversations/{id}/override", s.requireAuth(s.handleOverride))
	mux.HandleFunc("POST /api/conversations/{id}/reply", s.requireAuth(s.handleOwnerReply))
	mux.HandleFunc("POST /api/conversations/{id}/close", s.requireAuth(s.handleCloseConversation))
	mux.HandleFunc("GET /api/metrics", s.requireAuth(s.handleMetrics))

	mux.HandleFunc("GET /api/widget/boot", s.handleWidgetBoot)
	mux.HandleFunc("POST /api/widget/session", s.handleWidgetSession)
	mux.HandleFunc("POST /api/widget/message", s.handleWidgetMessage)
	mux.HandleFunc("GET /api/widget/poll", s.handleWidgetPoll)
	mux.HandleFunc("POST /api/widget/event", s.handleWidgetEvent)

	mux.HandleFunc("GET /widget/chat.js", s.handleWidgetJS)
	mux.HandleFunc("GET /widget/demo", s.handleWidgetDemo)

	mux.HandleFunc("POST /api/v1/sessions", s.requireSecretKey(s.handleAPICreateSession))
	mux.HandleFunc("POST /api/v1/sessions/{token}/messages", s.requireSecretKey(s.handleAPISendMessage))
	mux.HandleFunc("GET /api/v1/conversations", s.requireSecretKey(s.handleAPIListConversations))
	mux.HandleFunc("GET /api/v1/conversations/{id}", s.requireSecretKey(s.handleAPIGetConversation))
	mux.HandleFunc("GET /api/v1/icp", s.requireSecretKey(s.handleAPIGetICP))
	mux.HandleFunc("PUT /api/v1/icp", s.requireSecretKey(s.handleAPIPutICP))
	mux.HandleFunc("POST /api/v1/accounts", s.requireSecretKey(s.handleAPICreateChildAccount))
	mux.HandleFunc("GET /api/v1/accounts", s.requireSecretKey(s.handleAPIListChildAccounts))

	mux.HandleFunc("GET /ws/widget", s.handleWidgetWS)
	mux.HandleFunc("GET /ws/dashboard", s.handleDashboardWS)

	return s.cors(mux)
}
