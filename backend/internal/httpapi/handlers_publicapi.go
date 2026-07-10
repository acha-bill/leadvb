package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"leadqualifier/internal/auth"
	"leadqualifier/internal/models"
	"leadqualifier/internal/store"
	"leadqualifier/internal/wire"
)

func (s *Server) handleAPICreateSession(w http.ResponseWriter, r *http.Request, acct *models.Account) {
	var in struct {
		PageURL   string `json:"page_url"`
		VisitorID string `json:"visitor_id"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if s.quotaExceeded(acct) {
		errJSON(w, http.StatusPaymentRequired, "monthly conversation quota reached")
		return
	}
	conv, err := s.Store.CreateConversation(acct.ID, in.VisitorID, in.PageURL)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	cfg, _ := s.Store.GetWidgetConfig(acct.ID, acct.Company)
	if cfg.Greeting != "" {
		s.Store.InsertMessage(conv.ID, "assistant", cfg.Greeting, "")
	}
	msgs, _ := s.Store.ListMessages(conv.ID, 0)
	writeJSON(w, http.StatusCreated, map[string]any{"session_token": conv.Token, "conversation_id": conv.ID, "messages": wire.Messages(msgs)})
}

func (s *Server) handleAPISendMessage(w http.ResponseWriter, r *http.Request, acct *models.Account) {
	token := r.PathValue("token")
	conv, err := s.Store.GetConversationByToken(token)
	if err != nil || conv.AccountID != acct.ID {
		errJSON(w, http.StatusNotFound, "session not found")
		return
	}
	var in struct {
		Content string `json:"content"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.Content) == "" {
		errJSON(w, http.StatusBadRequest, "content required")
		return
	}
	msg, err := s.Engine.ProcessVisitorMessage(context.Background(), conv, in.Content)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	fresh, _ := s.Store.GetConversationByToken(token)
	out := map[string]any{"status": fresh.Status}
	if msg != nil {
		out["reply"] = wire.Message(msg)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleAPIListConversations(w http.ResponseWriter, r *http.Request, acct *models.Account) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	convs, total, err := s.Store.ListConversations(acct.ID, store.ConversationFilter{
		Status: q.Get("status"),
		Limit:  50,
		Offset: (page - 1) * 50,
	})
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]map[string]any, 0, len(convs))
	for _, c := range convs {
		items = append(items, wire.Conversation(c))
	}
	writeJSON(w, http.StatusOK, map[string]any{"conversations": items, "total": total, "page": page})
}

func (s *Server) handleAPIGetConversation(w http.ResponseWriter, r *http.Request, acct *models.Account) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		errJSON(w, http.StatusBadRequest, "invalid id")
		return
	}
	conv, err := s.Store.GetConversationByID(acct.ID, id)
	if err != nil {
		errJSON(w, http.StatusNotFound, "not found")
		return
	}
	msgs, _ := s.Store.ListMessages(conv.ID, 0)
	out := wire.Conversation(conv)
	out["messages"] = wire.Messages(msgs)
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleAPIGetICP(w http.ResponseWriter, r *http.Request, acct *models.Account) {
	icp, err := s.Store.GetICP(acct.ID)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, icp)
}

func (s *Server) handleAPIPutICP(w http.ResponseWriter, r *http.Request, acct *models.Account) {
	var in models.ICP
	if !readJSON(w, r, &in) {
		return
	}
	if in.Threshold <= 0 || in.Threshold > 100 {
		in.Threshold = 70
	}
	if in.Weights == (models.Weights{}) {
		in.Weights = models.DefaultWeights()
	}
	if err := s.Store.UpsertICP(acct.ID, in.Description, in.Threshold, in.Weights); err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	icp, _ := s.Store.GetICP(acct.ID)
	writeJSON(w, http.StatusOK, icp)
}

func (s *Server) handleAPICreateChildAccount(w http.ResponseWriter, r *http.Request, acct *models.Account) {
	plan := s.Cfg.Plan(acct.Plan)
	if plan.SubAccounts == 0 {
		errJSON(w, http.StatusForbidden, "sub-accounts require an Agency plan")
		return
	}
	n, _ := s.Store.CountChildAccounts(acct.ID)
	if n >= plan.SubAccounts {
		errJSON(w, http.StatusForbidden, "sub-account limit reached for your plan")
		return
	}
	var in struct {
		Name    string `json:"name"`
		Company string `json:"company"`
		Email   string `json:"email"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if in.Name == "" || !strings.Contains(in.Email, "@") {
		errJSON(w, http.StatusBadRequest, "name and valid email required")
		return
	}
	if _, err := s.Store.GetAccountByEmail(strings.ToLower(in.Email)); err == nil {
		errJSON(w, http.StatusConflict, "email already in use")
		return
	}
	password := auth.RandomHex(12)
	hash, err := auth.HashPassword(password)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	parent := acct.ID
	id, err := s.Store.CreateAccount(in.Name, in.Company, strings.ToLower(in.Email), hash, "professional", &parent)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.bootstrapAccount(id, in.Company, strings.ToLower(in.Email))
	keys, _ := s.Store.GetKeys(id)
	writeJSON(w, http.StatusCreated, map[string]any{
		"account_id":         id,
		"email":              strings.ToLower(in.Email),
		"temporary_password": password,
		"public_key":         keys.PublicKey,
		"secret_key":         keys.SecretKey,
	})
}

func (s *Server) handleAPIListChildAccounts(w http.ResponseWriter, r *http.Request, acct *models.Account) {
	children, err := s.Store.ListChildAccounts(acct.ID)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]map[string]any, 0, len(children))
	for _, c := range children {
		keys, _ := s.Store.GetKeys(c.ID)
		item := map[string]any{"account_id": c.ID, "name": c.Name, "company": c.Company, "email": c.Email, "plan": c.Plan}
		if keys != nil {
			item["public_key"] = keys.PublicKey
		}
		out = append(out, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"accounts": out})
}
