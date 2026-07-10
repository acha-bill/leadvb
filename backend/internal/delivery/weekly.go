package delivery

import (
	"context"
	"fmt"
	"html"
	"log"
	"time"

	"leadqualifier/internal/config"
	"leadqualifier/internal/store"
)

type WeeklyReporter struct {
	Store *store.Store
	Cfg   *config.Config
}

func (w *WeeklyReporter) Run(ctx context.Context) {
	if !w.Cfg.Features.WeeklyReports {
		return
	}
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick()
		}
	}
}

func (w *WeeklyReporter) tick() {
	accounts, err := w.Store.AccountsDueWeeklyReport()
	if err != nil {
		log.Printf("weekly: %v", err)
		return
	}
	for _, acct := range accounts {
		if !acct.ParsedSettings().WeeklyReport {
			continue
		}
		routing, err := w.Store.GetRoutingConfig(acct.ID)
		if err != nil || routing.EmailTo == "" {
			continue
		}
		stats, err := w.Store.GetWeeklyStats(acct.ID)
		if err != nil {
			continue
		}
		if stats.Conversations == 0 {
			w.Store.MarkWeeklySent(acct.ID)
			continue
		}
		saved := (stats.Qualified + stats.Disqualified) * w.Cfg.ManualScreenMinutes
		leads := ""
		for _, l := range stats.TopLeads {
			leads += "<li>" + html.EscapeString(l) + "</li>"
		}
		if leads == "" {
			leads = "<li>No qualified leads this week</li>"
		}
		body := fmt.Sprintf(`<div style="font-family:Arial,sans-serif;max-width:640px;margin:0 auto">
<h2>Your week in leads</h2>
<p><b>%d</b> conversations · <b>%d</b> qualified · <b>%d</b> screened out · ~<b>%d minutes</b> of screening saved</p>
<h3>Top leads</h3><ul>%s</ul>
<p><a href="%s" style="color:#4F46E5">Open your dashboard →</a></p>
</div>`, stats.Conversations, stats.Qualified, stats.Disqualified, saved, leads, w.Cfg.DashboardOrigin)
		EnqueueSystemEmail(w.Store, acct.ID, routing.EmailTo, "Your weekly lead report", body, "weekly")
		w.Store.MarkWeeklySent(acct.ID)
	}
}
