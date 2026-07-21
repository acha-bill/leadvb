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
		body := fmt.Sprintf(`<div style="font-family:Arial,sans-serif;max-width:640px;margin:0 auto;background:#f6f5f0;padding:32px;color:#18272a">
<div style="background:#ffffff;border:1px solid #d9d9d2;border-radius:20px;padding:32px">
<p style="margin:0;color:#2455d6;font-size:12px;font-weight:700;text-transform:uppercase;letter-spacing:1px">Weekly report</p>
<h2 style="margin:12px 0 8px">Your week in leads</h2>
<p style="color:#526064"><b>%d</b> conversations, <b>%d</b> qualified, <b>%d</b> screened out, and about <b>%d minutes</b> of screening saved.</p>
<h3 style="margin-top:28px">Top leads</h3><ul style="color:#526064;line-height:1.7">%s</ul>
<p style="margin:28px 0 0"><a href="%s" style="display:inline-block;background:#2455d6;color:#ffffff;text-decoration:none;font-weight:700;padding:12px 18px;border-radius:999px">Open your dashboard</a></p>
</div></div>`, stats.Conversations, stats.Qualified, stats.Disqualified, saved, leads, w.Cfg.DashboardOrigin)
		EnqueueSystemEmail(w.Store, acct.ID, routing.EmailTo, "Your weekly lead report", body, "weekly")
		w.Store.MarkWeeklySent(acct.ID)
	}
}
