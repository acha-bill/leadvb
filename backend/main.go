package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"leadqualifier/internal/ai"
	"leadqualifier/internal/config"
	"leadqualifier/internal/db"
	"leadqualifier/internal/delivery"
	"leadqualifier/internal/httpapi"
	"leadqualifier/internal/ratelimit"
	"leadqualifier/internal/store"
	"leadqualifier/internal/wire"
)

func main() {
	cfg := config.Load()

	database, err := db.Open(cfg.DBDSN)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	if err := db.Migrate(database); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	log.Println("database ready")

	st := store.New(database)
	hub := httpapi.NewHub()

	provider, err := ai.NewProvider(cfg)
	if err != nil {
		log.Fatalf("ai: %v", err)
	}
	log.Printf("AI provider: %s", cfg.AIProvider)

	engine := &ai.Engine{Store: st, Cfg: cfg, Provider: provider, Hub: hub}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	worker := &delivery.Worker{Store: st, Cfg: cfg}
	go worker.Run(ctx)

	weekly := &delivery.WeeklyReporter{Store: st, Cfg: cfg}
	go weekly.Run(ctx)

	go abandonSweeper(ctx, st, hub, cfg)

	server := &httpapi.Server{
		Cfg:     cfg,
		Store:   st,
		Engine:  engine,
		Hub:     hub,
		Limiter: ratelimit.New(),
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           server.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	log.Printf("listening on :%s (dashboard origin %s)", cfg.Port, cfg.DashboardOrigin)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
}

func abandonSweeper(ctx context.Context, st *store.Store, hub *httpapi.Hub, cfg *config.Config) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().UTC().Add(-time.Duration(cfg.IdleAbandonMinutes) * time.Minute)
			abandoned, err := st.MarkAbandoned(cutoff)
			if err != nil {
				log.Printf("sweeper: %v", err)
				continue
			}
			for _, c := range abandoned {
				hub.ToConversation(c.ID, map[string]any{"type": "status", "status": "abandoned"})
				if fresh, err := st.GetConversationByID(c.AccountID, c.ID); err == nil {
					hub.ToAccount(c.AccountID, map[string]any{"type": "conversation_updated", "conversation": wire.Conversation(fresh)})
				}
			}
		}
	}
}
