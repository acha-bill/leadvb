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

	body = fmt.Sprintf(`<div style="font-family:Arial,sans-serif;max-width:640px;margin:0 auto;background:#f6f5f0;padding:32px;color:#18272a">
<div style="background:#ffffff;border:1px solid #d9d9d2;border-radius:20px;padding:32px">
<p style="margin:0;color:#2455d6;font-size:12px;font-weight:700;text-transform:uppercase;letter-spacing:1px">Lead Qualifier</p>
<h2 style="margin:12px 0 8px">%s</h2>
<p style="color:#526064">%s</p>
<table style="border-collapse:collapse;background:#f6f5f0;border-radius:10px">%s</table>
<h3 style="margin-top:28px">Transcript</h3>
<div style="background:#f6f5f0;padding:16px;border-radius:10px;color:#526064">%s</div>
<p style="margin:28px 0 0"><a href="%s" style="display:inline-block;background:#2455d6;color:#ffffff;text-decoration:none;font-weight:700;padding:12px 18px;border-radius:999px">Open in dashboard</a></p>
</div></div>`, label, html.EscapeString(p.Summary), rows.String(), transcript.String(), p.DashboardURL)
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
