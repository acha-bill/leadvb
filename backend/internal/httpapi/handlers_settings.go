package httpapi

import (
	"net/http"
	"strings"

	"leadqualifier/internal/delivery"
	"leadqualifier/internal/models"
)

func (s *Server) handleGetICP(w http.ResponseWriter, r *http.Request, acct *models.Account) {
	icp, err := s.Store.GetICP(acct.ID)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, icp)
}

func (s *Server) handlePutICP(w http.ResponseWriter, r *http.Request, acct *models.Account) {
	var in models.ICP
	if !readJSON(w, r, &in) {
		return
	}
	if in.Threshold < 0 || in.Threshold > 100 {
		in.Threshold = 70
	}
	if in.Weights == (models.Weights{}) {
		in.Weights = models.DefaultWeights()
	}
	if len(in.Description) > 8000 {
		in.Description = in.Description[:8000]
	}
	if err := s.Store.UpsertICP(acct.ID, in.Description, in.Threshold, in.Weights); err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	icp, _ := s.Store.GetICP(acct.ID)
	writeJSON(w, http.StatusOK, icp)
}

func (s *Server) handleGetRouting(w http.ResponseWriter, r *http.Request, acct *models.Account) {
	cfg, err := s.Store.GetRoutingConfig(acct.ID)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) handlePutRouting(w http.ResponseWriter, r *http.Request, acct *models.Account) {
	var in models.RoutingConfig
	if !readJSON(w, r, &in) {
		return
	}
	if in.WebhookEnabled && !s.Cfg.Plan(acct.Plan).Webhooks {
		errJSON(w, http.StatusForbidden, "webhook routing requires the Professional plan or higher")
		return
	}
	if err := s.Store.UpsertRoutingConfig(acct.ID, in); err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, in)
}

func (s *Server) handleTestRouting(w http.ResponseWriter, r *http.Request, acct *models.Account) {
	var in struct {
		Channel string `json:"channel"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	routing, err := s.Store.GetRoutingConfig(acct.ID)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	score := int64(85)
	conf := int64(90)
	sample := delivery.LeadPayload{
		Event:     "lead.test",
		AccountID: acct.ID, Company: acct.Company,
		Status: "qualified", Score: &score, Confidence: &conf,
		Contact:      map[string]string{"name": "Test Lead", "email": "test@example.com", "phone": "+1 555 0100", "company": "Acme Ltd"},
		Summary:      "This is a test delivery from your Lead Qualifier dashboard.",
		PageURL:      "https://example.com/pricing",
		DashboardURL: s.Cfg.DashboardOrigin,
		Transcript: []delivery.TranscriptLine{
			{Role: "assistant", Content: "Hi! How can I help?"},
			{Role: "visitor", Content: "We need help with our IT, budget around $3k/month."},
		},
	}
	var sendErr error
	switch in.Channel {
	case "email":
		if routing.EmailTo == "" {
			errJSON(w, http.StatusBadRequest, "set a destination email first")
			return
		}
		subject, body := delivery.LeadEmailHTML(sample)
		sendErr = delivery.SendEmail(s.Cfg, routing.EmailTo, "[TEST] "+subject, body)
	case "slack":
		if routing.SlackWebhookURL == "" {
			errJSON(w, http.StatusBadRequest, "set a Slack webhook URL first")
			return
		}
		sendErr = delivery.SendSlack(routing.SlackWebhookURL, sample)
	case "webhook":
		if routing.WebhookURL == "" {
			errJSON(w, http.StatusBadRequest, "set a webhook URL first")
			return
		}
		sendErr = delivery.SendWebhook(routing.WebhookURL, routing.WebhookSecret, []byte(models.JSONString(sample)), "lead.test")
	default:
		errJSON(w, http.StatusBadRequest, "channel must be email, slack or webhook")
		return
	}
	if sendErr != nil {
		errJSON(w, http.StatusBadGateway, sendErr.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleGetWidgetConfig(w http.ResponseWriter, r *http.Request, acct *models.Account) {
	cfg, err := s.Store.GetWidgetConfig(acct.ID, acct.Company)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	keys, _ := s.Store.GetKeys(acct.ID)
	pk := ""
	if keys != nil {
		pk = keys.PublicKey
	}
	writeJSON(w, http.StatusOK, map[string]any{"config": cfg, "public_key": pk, "base_url": s.Cfg.PublicBaseURL})
}

func (s *Server) handlePutWidgetConfig(w http.ResponseWriter, r *http.Request, acct *models.Account) {
	var in models.WidgetConfig
	if !readJSON(w, r, &in) {
		return
	}
	if len(in.LogoURL) > 300_000 {
		errJSON(w, http.StatusBadRequest, "logo too large (max ~200KB)")
		return
	}
	if in.Position != "left" {
		in.Position = "right"
	}
	if in.PrimaryColor == "" {
		in.PrimaryColor = "#4F46E5"
	}
	if in.Proactive.DelaySeconds < 2 {
		in.Proactive.DelaySeconds = 2
	}
	if !s.Cfg.Plan(acct.Plan).WhiteLabel {
		in.Branding = true
	}
	if err := s.Store.UpsertWidgetConfig(acct.ID, in); err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, in)
}

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request, acct *models.Account) {
	writeJSON(w, http.StatusOK, map[string]any{"settings": acct.ParsedSettings(), "plan": acct.Plan})
}

func (s *Server) handlePutSettings(w http.ResponseWriter, r *http.Request, acct *models.Account) {
	var in struct {
		Settings models.AccountSettings `json:"settings"`
		Plan     string                 `json:"plan"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if err := s.Store.UpdateAccountSettings(acct.ID, models.JSONString(in.Settings)); err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	switch strings.TrimSpace(in.Plan) {
	case "", acct.Plan:
	case "starter", "professional", "agency", "enterprise":
		if err := s.Store.UpdateAccountPlan(acct.ID, in.Plan); err != nil {
			errJSON(w, http.StatusInternalServerError, err.Error())
			return
		}
	default:
		errJSON(w, http.StatusBadRequest, "unknown plan")
		return
	}
	fresh, _ := s.Store.GetAccountByID(acct.ID)
	writeJSON(w, http.StatusOK, s.accountJSON(fresh))
}

func (s *Server) handleGetKeys(w http.ResponseWriter, r *http.Request, acct *models.Account) {
	keys, err := s.Store.GetKeys(acct.ID)
	if err != nil {
		keys, err = s.Store.CreateAPIKeys(acct.ID)
		if err != nil {
			errJSON(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	domains, _ := s.Store.GetWidgetDomains(acct.ID)
	writeJSON(w, http.StatusOK, map[string]any{
		"public_key":  keys.PublicKey,
		"secret_key":  keys.SecretKey,
		"domains":     domains,
		"snippet":     `<script src="` + s.Cfg.PublicBaseURL + `/widget/chat.js" async></script>`,
		"key_snippet": `<script src="` + s.Cfg.PublicBaseURL + `/widget/chat.js" data-widget-key="` + keys.PublicKey + `" async></script>`,
	})
}

func (s *Server) handleGetWidgetDomains(w http.ResponseWriter, r *http.Request, acct *models.Account) {
	domains, err := s.Store.GetWidgetDomains(acct.ID)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"domains": domains})
}

func (s *Server) handlePutWidgetDomains(w http.ResponseWriter, r *http.Request, acct *models.Account) {
	var in struct {
		Domains []string `json:"domains"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if len(in.Domains) > 50 {
		errJSON(w, http.StatusBadRequest, "too many domains (max 50)")
		return
	}
	saved, err := s.Store.SetWidgetDomains(acct.ID, in.Domains)
	if err != nil {
		errJSON(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"domains": saved})
}

func (s *Server) handleRotateKeys(w http.ResponseWriter, r *http.Request, acct *models.Account) {
	keys, err := s.Store.RotateKeys(acct.ID)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"public_key": keys.PublicKey, "secret_key": keys.SecretKey})
}
