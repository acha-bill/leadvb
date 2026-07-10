package httpapi

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type wsClient struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (c *wsClient) send(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return c.conn.WriteJSON(v)
}

type Hub struct {
	mu        sync.RWMutex
	byConv    map[int64]map[*wsClient]bool
	byAccount map[int64]map[*wsClient]bool
}

func NewHub() *Hub {
	return &Hub{byConv: map[int64]map[*wsClient]bool{}, byAccount: map[int64]map[*wsClient]bool{}}
}

func (h *Hub) addConv(id int64, c *wsClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.byConv[id] == nil {
		h.byConv[id] = map[*wsClient]bool{}
	}
	h.byConv[id][c] = true
}

func (h *Hub) removeConv(id int64, c *wsClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.byConv[id], c)
	if len(h.byConv[id]) == 0 {
		delete(h.byConv, id)
	}
}

func (h *Hub) addAccount(id int64, c *wsClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.byAccount[id] == nil {
		h.byAccount[id] = map[*wsClient]bool{}
	}
	h.byAccount[id][c] = true
}

func (h *Hub) removeAccount(id int64, c *wsClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.byAccount[id], c)
	if len(h.byAccount[id]) == 0 {
		delete(h.byAccount, id)
	}
}

func (h *Hub) ToConversation(convID int64, v any) {
	h.mu.RLock()
	clients := make([]*wsClient, 0, len(h.byConv[convID]))
	for c := range h.byConv[convID] {
		clients = append(clients, c)
	}
	h.mu.RUnlock()
	for _, c := range clients {
		c.send(v)
	}
}

func (h *Hub) ToAccount(accountID int64, v any) {
	h.mu.RLock()
	clients := make([]*wsClient, 0, len(h.byAccount[accountID]))
	for c := range h.byAccount[accountID] {
		clients = append(clients, c)
	}
	h.mu.RUnlock()
	for _, c := range clients {
		c.send(v)
	}
}

var widgetUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func (s *Server) handleWidgetWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("session_token")
	if token == "" {
		http.Error(w, "missing session_token", http.StatusBadRequest)
		return
	}
	conv, err := s.Store.GetConversationByToken(token)
	if err != nil {
		http.Error(w, "invalid session", http.StatusUnauthorized)
		return
	}
	conn, err := widgetUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := &wsClient{conn: conn}
	s.Hub.addConv(conv.ID, client)
	defer func() {
		s.Hub.removeConv(conv.ID, client)
		conn.Close()
	}()

	go pingLoop(client)

	conn.SetReadLimit(8 << 10)
	conn.SetReadDeadline(time.Now().Add(120 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		return nil
	})

	for {
		var in struct {
			Type    string `json:"type"`
			Content string `json:"content"`
		}
		if err := conn.ReadJSON(&in); err != nil {
			return
		}
		conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		if in.Type == "message" && in.Content != "" {
			if !s.Limiter.Allow("msg:"+token, 1, 5) {
				client.send(map[string]any{"type": "error", "error": "rate_limited"})
				continue
			}
			fresh, err := s.Store.GetConversationByToken(token)
			if err != nil {
				return
			}
			go func(content string) {
				if _, err := s.Engine.ProcessVisitorMessage(context.Background(), fresh, content); err != nil {
					log.Printf("ws: process error: %v", err)
				}
			}(in.Content)
		}
	}
}

func (s *Server) handleDashboardWS(w http.ResponseWriter, r *http.Request) {
	accountID, ok := s.authAccountID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	conn, err := widgetUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := &wsClient{conn: conn}
	s.Hub.addAccount(accountID, client)
	defer func() {
		s.Hub.removeAccount(accountID, client)
		conn.Close()
	}()

	go pingLoop(client)

	conn.SetReadLimit(4 << 10)
	conn.SetReadDeadline(time.Now().Add(120 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		return nil
	})
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
		conn.SetReadDeadline(time.Now().Add(120 * time.Second))
	}
}

func pingLoop(c *wsClient) {
	ticker := time.NewTicker(45 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		err := c.conn.WriteMessage(websocket.PingMessage, nil)
		c.mu.Unlock()
		if err != nil {
			return
		}
	}
}
