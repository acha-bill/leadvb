package httpapi

import (
	"net/http"
	"strconv"

	"leadqualifier/internal/models"
	"leadqualifier/internal/store"
	"leadqualifier/internal/wire"
)

func (s *Server) handleListConversations(w http.ResponseWriter, r *http.Request, acct *models.Account) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	limit := 25
	f := store.ConversationFilter{
		Status: q.Get("status"),
		Query:  q.Get("q"),
		Limit:  limit,
		Offset: (page - 1) * limit,
	}
	convs, total, err := s.Store.ListConversations(acct.ID, f)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]map[string]any, 0, len(convs))
	for _, c := range convs {
		items = append(items, wire.Conversation(c))
	}
	writeJSON(w, http.StatusOK, map[string]any{"conversations": items, "total": total, "page": page, "per_page": limit})
}

func (s *Server) pathConversation(w http.ResponseWriter, r *http.Request, acct *models.Account) *models.Conversation {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		errJSON(w, http.StatusBadRequest, "invalid conversation id")
		return nil
	}
	conv, err := s.Store.GetConversationByID(acct.ID, id)
	if err != nil {
		errJSON(w, http.StatusNotFound, "conversation not found")
		return nil
	}
	return conv
}

func (s *Server) handleGetConversation(w http.ResponseWriter, r *http.Request, acct *models.Account) {
	conv := s.pathConversation(w, r, acct)
	if conv == nil {
		return
	}
	msgs, err := s.Store.ListMessages(conv.ID, 0)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := wire.Conversation(conv)
	out["messages"] = wire.Messages(msgs)
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleOverride(w http.ResponseWriter, r *http.Request, acct *models.Account) {
	conv := s.pathConversation(w, r, acct)
	if conv == nil {
		return
	}
	var in struct {
		Status string `json:"status"`
		Note   string `json:"note"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if in.Status != "hot" && in.Status != "cold" {
		errJSON(w, http.StatusBadRequest, "status must be 'hot' or 'cold'")
		return
	}
	if err := s.Store.SetOverride(acct.ID, conv.ID, in.Status, in.Note); err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.Store.InsertFeedback(acct.ID, conv.ID, conv.Status, in.Status, in.Note)

	fresh, _ := s.Store.GetConversationByID(acct.ID, conv.ID)
	s.Hub.ToAccount(acct.ID, map[string]any{"type": "conversation_updated", "conversation": wire.Conversation(fresh)})
	writeJSON(w, http.StatusOK, wire.Conversation(fresh))
}

func (s *Server) handleOwnerReply(w http.ResponseWriter, r *http.Request, acct *models.Account) {
	conv := s.pathConversation(w, r, acct)
	if conv == nil {
		return
	}
	if conv.Status != "active" && conv.Status != "handoff" {
		errJSON(w, http.StatusBadRequest, "conversation has ended")
		return
	}
	var in struct {
		Content string `json:"content"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if in.Content == "" {
		errJSON(w, http.StatusBadRequest, "content required")
		return
	}
	if conv.Status == "active" {
		s.Store.SetConversationStatus(conv.ID, "handoff", false)
		s.Hub.ToConversation(conv.ID, map[string]any{"type": "status", "status": "handoff"})
	}
	msg, err := s.Store.InsertMessage(conv.ID, "owner", in.Content, "")
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.Hub.ToConversation(conv.ID, map[string]any{"type": "message", "message": wire.Message(msg)})
	fresh, _ := s.Store.GetConversationByID(acct.ID, conv.ID)
	s.Hub.ToAccount(acct.ID, map[string]any{"type": "conversation_updated", "conversation": wire.Conversation(fresh)})
	writeJSON(w, http.StatusOK, wire.Message(msg))
}

func (s *Server) handleCloseConversation(w http.ResponseWriter, r *http.Request, acct *models.Account) {
	conv := s.pathConversation(w, r, acct)
	if conv == nil {
		return
	}
	if err := s.Store.SetConversationStatus(conv.ID, "closed", true); err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.Hub.ToConversation(conv.ID, map[string]any{"type": "status", "status": "closed"})
	fresh, _ := s.Store.GetConversationByID(acct.ID, conv.ID)
	s.Hub.ToAccount(acct.ID, map[string]any{"type": "conversation_updated", "conversation": wire.Conversation(fresh)})
	writeJSON(w, http.StatusOK, wire.Conversation(fresh))
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request, acct *models.Account) {
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 || days > 365 {
		days = 30
	}
	m, err := s.Store.GetMetrics(acct.ID, days, s.Cfg.ManualScreenMinutes)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, m)
}
