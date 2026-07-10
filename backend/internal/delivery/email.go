package delivery

import (
	"crypto/tls"
	"fmt"
	"html"
	"net"
	"net/smtp"
	"strings"

	"leadqualifier/internal/config"
)

func SendEmail(cfg *config.Config, to, subject, htmlBody string) error {
	if cfg.SMTPHost == "" {
		return fmt.Errorf("SMTP not configured (set SMTP_HOST)")
	}
	from := cfg.SMTPFrom
	fromAddr := from
	if i := strings.Index(from, "<"); i >= 0 {
		fromAddr = strings.Trim(from[i:], "<>")
	}
	headers := map[string]string{
		"From":         from,
		"To":           to,
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": `text/html; charset="UTF-8"`,
	}
	var sb strings.Builder
	for k, v := range headers {
		sb.WriteString(k + ": " + v + "\r\n")
	}
	sb.WriteString("\r\n" + htmlBody)
	msg := []byte(sb.String())
	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)

	var auth smtp.Auth
	if cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPHost)
	}

	if cfg.SMTPPort == 465 {
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: cfg.SMTPHost})
		if err != nil {
			return err
		}
		c, err := smtp.NewClient(conn, cfg.SMTPHost)
		if err != nil {
			return err
		}
		defer c.Close()
		if auth != nil {
			if err := c.Auth(auth); err != nil {
				return err
			}
		}
		if err := c.Mail(fromAddr); err != nil {
			return err
		}
		if err := c.Rcpt(to); err != nil {
			return err
		}
		w, err := c.Data()
		if err != nil {
			return err
		}
		if _, err := w.Write(msg); err != nil {
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
		return c.Quit()
	}

	if _, _, err := net.SplitHostPort(addr); err != nil {
		return fmt.Errorf("invalid SMTP address %q: %w", addr, err)
	}
	return smtp.SendMail(addr, auth, fromAddr, []string{to}, msg)
}

func LeadEmailHTML(p LeadPayload) (subject, body string) {
	label := "New qualified lead"
	if p.Event == "handoff" {
		label = "Visitor requested a human"
	}
	score := "—"
	if p.Score != nil {
		score = fmt.Sprintf("%d / 100", *p.Score)
	}
	subject = fmt.Sprintf("%s — %s", label, contactLine(p.Contact))

	var rows strings.Builder
	add := func(k, v string) {
		if v != "" {
			rows.WriteString("<tr><td style='padding:6px 12px;color:#666'>" + k + "</td><td style='padding:6px 12px'><b>" + html.EscapeString(v) + "</b></td></tr>")
		}
	}
	add("Name", p.Contact["name"])
	add("Email", p.Contact["email"])
	add("Phone", p.Contact["phone"])
	add("Company", p.Contact["company"])
	add("Score", score)
	add("Page", p.PageURL)

	var transcript strings.Builder
	for _, t := range p.Transcript {
		who := "Visitor"
		if t.Role == "assistant" {
			who = "Assistant"
		} else if t.Role == "owner" {
			who = "You"
		}
		transcript.WriteString("<p style='margin:4px 0'><b>" + who + ":</b> " + html.EscapeString(t.Content) + "</p>")
	}

	body = fmt.Sprintf(`<div style="font-family:Arial,sans-serif;max-width:640px;margin:0 auto">
<h2 style="color:#111">%s</h2>
<p>%s</p>
<table style="border-collapse:collapse;background:#f8f8fa;border-radius:8px">%s</table>
<h3>Transcript</h3>
<div style="background:#f8f8fa;padding:12px;border-radius:8px">%s</div>
<p><a href="%s" style="color:#4F46E5">Open in dashboard →</a></p>
</div>`, label, html.EscapeString(p.Summary), rows.String(), transcript.String(), p.DashboardURL)
	return subject, body
}

func contactLine(c map[string]string) string {
	if c["name"] != "" && c["email"] != "" {
		return c["name"] + " (" + c["email"] + ")"
	}
	if c["email"] != "" {
		return c["email"]
	}
	if c["name"] != "" {
		return c["name"]
	}
	return "anonymous visitor"
}
