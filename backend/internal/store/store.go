package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"leadqualifier/internal/auth"
	"leadqualifier/internal/models"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	DB *sql.DB
}

func New(db *sql.DB) *Store { return &Store{DB: db} }

const accountCols = "id, name, company, email, password_hash, plan, parent_account_id, white_label, settings, created_at"

func scanAccount(row interface{ Scan(...any) error }) (*models.Account, error) {
	a := &models.Account{}
	err := row.Scan(&a.ID, &a.Name, &a.Company, &a.Email, &a.PasswordHash, &a.Plan, &a.ParentID, &a.WhiteLabel, &a.Settings, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return a, err
}

func (s *Store) CreateAccount(name, company, email, hash, plan string, parentID *int64) (int64, error) {
	var parent any
	if parentID != nil {
		parent = *parentID
	}
	res, err := s.DB.Exec(`INSERT INTO accounts (name, company, email, password_hash, plan, parent_account_id) VALUES (?,?,?,?,?,?)`,
		name, company, email, hash, plan, parent)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) GetAccountByEmail(email string) (*models.Account, error) {
	return scanAccount(s.DB.QueryRow("SELECT "+accountCols+" FROM accounts WHERE email=?", email))
}

func (s *Store) GetAccountByID(id int64) (*models.Account, error) {
	return scanAccount(s.DB.QueryRow("SELECT "+accountCols+" FROM accounts WHERE id=?", id))
}

func (s *Store) UpdateAccountSettings(id int64, settingsJSON string) error {
	_, err := s.DB.Exec("UPDATE accounts SET settings=? WHERE id=?", settingsJSON, id)
	return err
}

func (s *Store) UpdateAccountPlan(id int64, plan string) error {
	_, err := s.DB.Exec("UPDATE accounts SET plan=? WHERE id=?", plan, id)
	return err
}

func (s *Store) SetWhiteLabel(id int64, on bool) error {
	_, err := s.DB.Exec("UPDATE accounts SET white_label=? WHERE id=?", on, id)
	return err
}

func (s *Store) ListChildAccounts(parentID int64) ([]*models.Account, error) {
	rows, err := s.DB.Query("SELECT "+accountCols+" FROM accounts WHERE parent_account_id=? ORDER BY id", parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Account
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) CountChildAccounts(parentID int64) (int, error) {
	var n int
	err := s.DB.QueryRow("SELECT COUNT(*) FROM accounts WHERE parent_account_id=?", parentID).Scan(&n)
	return n, err
}

func (s *Store) CreateAPIKeys(accountID int64) (*models.APIKey, error) {
	k := &models.APIKey{AccountID: accountID, PublicKey: auth.NewPublicKey(), SecretKey: auth.NewSecretKey(), Active: true}
	res, err := s.DB.Exec("INSERT INTO api_keys (account_id, public_key, secret_key) VALUES (?,?,?)", accountID, k.PublicKey, k.SecretKey)
	if err != nil {
		return nil, err
	}
	k.ID, _ = res.LastInsertId()
	return k, nil
}

func (s *Store) GetKeys(accountID int64) (*models.APIKey, error) {
	k := &models.APIKey{}
	err := s.DB.QueryRow("SELECT id, account_id, public_key, secret_key, active FROM api_keys WHERE account_id=? AND active=1 ORDER BY id DESC LIMIT 1", accountID).
		Scan(&k.ID, &k.AccountID, &k.PublicKey, &k.SecretKey, &k.Active)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return k, err
}

func (s *Store) RotateKeys(accountID int64) (*models.APIKey, error) {
	if _, err := s.DB.Exec("UPDATE api_keys SET active=0 WHERE account_id=?", accountID); err != nil {
		return nil, err
	}
	return s.CreateAPIKeys(accountID)
}

func (s *Store) GetAccountByPublicKey(pk string) (*models.Account, error) {
	return scanAccount(s.DB.QueryRow("SELECT a."+strings.ReplaceAll(accountCols, ", ", ", a.")+` FROM accounts a
		JOIN api_keys k ON k.account_id = a.id WHERE k.public_key=? AND k.active=1`, pk))
}

func (s *Store) GetAccountBySecretKey(sk string) (*models.Account, error) {
	return scanAccount(s.DB.QueryRow("SELECT a."+strings.ReplaceAll(accountCols, ", ", ", a.")+` FROM accounts a
		JOIN api_keys k ON k.account_id = a.id WHERE k.secret_key=? AND k.active=1`, sk))
}

func normalizeDomain(d string) string {
	d = strings.ToLower(strings.TrimSpace(d))
	d = strings.TrimPrefix(d, "https://")
	d = strings.TrimPrefix(d, "http://")
	if i := strings.IndexAny(d, "/?#"); i >= 0 {
		d = d[:i]
	}
	if i := strings.LastIndex(d, ":"); i >= 0 && !strings.Contains(d, "]") {
		if _, err := fmt.Sscanf(d[i+1:], "%d", new(int)); err == nil {
			d = d[:i]
		}
	}
	return strings.TrimSuffix(d, ".")
}

func (s *Store) SetWidgetDomains(accountID int64, domains []string) ([]string, error) {
	clean := make([]string, 0, len(domains))
	seen := map[string]bool{}
	for _, d := range domains {
		d = normalizeDomain(d)
		if d == "" || seen[d] || len(d) > 190 {
			continue
		}
		seen[d] = true
		clean = append(clean, d)
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM widget_domains WHERE account_id=?", accountID); err != nil {
		return nil, err
	}
	for _, d := range clean {
		var owner int64
		err := tx.QueryRow("SELECT account_id FROM widget_domains WHERE domain=?", d).Scan(&owner)
		if err == nil && owner != accountID {
			return nil, fmt.Errorf("domain %q is already claimed by another account", d)
		}
		if _, err := tx.Exec("INSERT INTO widget_domains (domain, account_id) VALUES (?,?)", d, accountID); err != nil {
			return nil, err
		}
	}
	return clean, tx.Commit()
}

func (s *Store) GetWidgetDomains(accountID int64) ([]string, error) {
	rows, err := s.DB.Query("SELECT domain FROM widget_domains WHERE account_id=? ORDER BY domain", accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// GetAccountByDomain resolves a request host to an account, walking up parent
// domains so a registered "example.com" also matches "www.example.com".
func (s *Store) GetAccountByDomain(host string) (*models.Account, error) {
	host = normalizeDomain(host)
	if host == "" {
		return nil, ErrNotFound
	}
	labels := strings.Split(host, ".")
	for i := 0; i < len(labels); i++ {
		candidate := strings.Join(labels[i:], ".")
		if candidate == "" {
			break
		}
		var accountID int64
		err := s.DB.QueryRow("SELECT account_id FROM widget_domains WHERE domain=?", candidate).Scan(&accountID)
		if err == nil {
			return s.GetAccountByID(accountID)
		}
		if err != sql.ErrNoRows {
			return nil, err
		}
		if len(labels)-i <= 2 {
			break
		}
	}
	return nil, ErrNotFound
}

func (s *Store) GetICP(accountID int64) (*models.ICP, error) {
	icp := &models.ICP{AccountID: accountID}
	var weights sql.NullString
	err := s.DB.QueryRow("SELECT description, threshold, weights FROM icp_profiles WHERE account_id=?", accountID).
		Scan(&icp.Description, &icp.Threshold, &weights)
	if err == sql.ErrNoRows {
		return &models.ICP{AccountID: accountID, Threshold: 70, Weights: models.DefaultWeights()}, nil
	}
	if err != nil {
		return nil, err
	}
	icp.Weights = models.DefaultWeights()
	if weights.Valid && weights.String != "" {
		unmarshal(weights.String, &icp.Weights)
	}
	return icp, nil
}

func (s *Store) UpsertICP(accountID int64, description string, threshold int, weights models.Weights) error {
	_, err := s.DB.Exec(`INSERT INTO icp_profiles (account_id, description, threshold, weights) VALUES (?,?,?,?)
		ON DUPLICATE KEY UPDATE description=VALUES(description), threshold=VALUES(threshold), weights=VALUES(weights)`,
		accountID, description, threshold, models.JSONString(weights))
	return err
}

func (s *Store) GetWidgetConfig(accountID int64, company string) (models.WidgetConfig, error) {
	var raw string
	err := s.DB.QueryRow("SELECT config FROM widget_configs WHERE account_id=?", accountID).Scan(&raw)
	if err == sql.ErrNoRows {
		return models.DefaultWidgetConfig(company), nil
	}
	if err != nil {
		return models.WidgetConfig{}, err
	}
	cfg := models.DefaultWidgetConfig(company)
	unmarshal(raw, &cfg)
	return cfg, nil
}

func (s *Store) UpsertWidgetConfig(accountID int64, cfg models.WidgetConfig) error {
	_, err := s.DB.Exec(`INSERT INTO widget_configs (account_id, config) VALUES (?,?)
		ON DUPLICATE KEY UPDATE config=VALUES(config)`, accountID, models.JSONString(cfg))
	return err
}

func (s *Store) GetRoutingConfig(accountID int64) (models.RoutingConfig, error) {
	var raw string
	err := s.DB.QueryRow("SELECT config FROM routing_configs WHERE account_id=?", accountID).Scan(&raw)
	if err == sql.ErrNoRows {
		return models.RoutingConfig{Disqualified: models.DisqualifiedAction{Mode: "polite"}}, nil
	}
	if err != nil {
		return models.RoutingConfig{}, err
	}
	var cfg models.RoutingConfig
	unmarshal(raw, &cfg)
	return cfg, nil
}

func (s *Store) UpsertRoutingConfig(accountID int64, cfg models.RoutingConfig) error {
	_, err := s.DB.Exec(`INSERT INTO routing_configs (account_id, config) VALUES (?,?)
		ON DUPLICATE KEY UPDATE config=VALUES(config)`, accountID, models.JSONString(cfg))
	return err
}

const convCols = "id, account_id, visitor_id, token, page_url, status, score, bant, contact, summary, confidence, language, override_status, override_note, message_count, started_at, ended_at, last_activity_at"

func scanConv(row interface{ Scan(...any) error }) (*models.Conversation, error) {
	c := &models.Conversation{}
	err := row.Scan(&c.ID, &c.AccountID, &c.VisitorID, &c.Token, &c.PageURL, &c.Status, &c.Score, &c.Bant, &c.Contact,
		&c.Summary, &c.Confidence, &c.Language, &c.OverrideStatus, &c.OverrideNote, &c.MessageCount, &c.StartedAt, &c.EndedAt, &c.LastActivityAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return c, err
}

func (s *Store) CreateConversation(accountID int64, visitorID, pageURL string) (*models.Conversation, error) {
	token := auth.NewSessionToken()
	res, err := s.DB.Exec(`INSERT INTO conversations (account_id, visitor_id, token, page_url) VALUES (?,?,?,?)`,
		accountID, visitorID, token, truncate(pageURL, 500))
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.GetConversationByID(accountID, id)
}

func (s *Store) GetConversationByToken(token string) (*models.Conversation, error) {
	return scanConv(s.DB.QueryRow("SELECT "+convCols+" FROM conversations WHERE token=?", token))
}

func (s *Store) GetConversationByID(accountID, id int64) (*models.Conversation, error) {
	return scanConv(s.DB.QueryRow("SELECT "+convCols+" FROM conversations WHERE id=? AND account_id=?", id, accountID))
}

type ConversationFilter struct {
	Status string
	Query  string
	Limit  int
	Offset int
}

func (s *Store) ListConversations(accountID int64, f ConversationFilter) ([]*models.Conversation, int, error) {
	where := "WHERE account_id=?"
	args := []any{accountID}
	if f.Status != "" {
		where += " AND status=?"
		args = append(args, f.Status)
	}
	if f.Query != "" {
		where += " AND (contact LIKE ? OR summary LIKE ? OR page_url LIKE ?)"
		q := "%" + f.Query + "%"
		args = append(args, q, q, q)
	}
	var total int
	if err := s.DB.QueryRow("SELECT COUNT(*) FROM conversations "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 25
	}
	rows, err := s.DB.Query("SELECT "+convCols+" FROM conversations "+where+" ORDER BY last_activity_at DESC LIMIT ? OFFSET ?",
		append(args, f.Limit, f.Offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*models.Conversation
	for rows.Next() {
		c, err := scanConv(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, c)
	}
	return out, total, rows.Err()
}

func (s *Store) CountConversationsThisMonth(accountID int64) (int, error) {
	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	var n int
	err := s.DB.QueryRow("SELECT COUNT(*) FROM conversations WHERE account_id=? AND started_at>=?", accountID, monthStart).Scan(&n)
	return n, err
}

func (s *Store) TouchConversation(id int64) error {
	_, err := s.DB.Exec("UPDATE conversations SET last_activity_at=NOW() WHERE id=?", id)
	return err
}

func (s *Store) UpdateConversationAI(id int64, status string, score *int, bantJSON, contactJSON, summary string, confidence *int, language string, ended bool) error {
	q := `UPDATE conversations SET status=?, score=?, bant=?, contact=?, summary=?, confidence=?, language=?, last_activity_at=NOW()`
	args := []any{status, nilIfZero(score), nilIfEmpty(bantJSON), nilIfEmpty(contactJSON), nilIfEmpty(summary), nilIfZero(confidence), language}
	if ended {
		q += ", ended_at=NOW()"
	}
	q += " WHERE id=?"
	args = append(args, id)
	_, err := s.DB.Exec(q, args...)
	return err
}

func (s *Store) SetConversationStatus(id int64, status string, ended bool) error {
	q := "UPDATE conversations SET status=?"
	if ended {
		q += ", ended_at=COALESCE(ended_at, NOW())"
	}
	q += " WHERE id=?"
	_, err := s.DB.Exec(q, status, id)
	return err
}

func (s *Store) SetOverride(accountID, id int64, status, note string) error {
	res, err := s.DB.Exec("UPDATE conversations SET override_status=?, override_note=? WHERE id=? AND account_id=?", status, nilIfEmpty(note), id, accountID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) MarkAbandoned(idleBefore time.Time) ([]*models.Conversation, error) {
	rows, err := s.DB.Query("SELECT "+convCols+" FROM conversations WHERE status='active' AND last_activity_at<?", idleBefore)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Conversation
	for rows.Next() {
		c, err := scanConv(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, c := range out {
		s.DB.Exec("UPDATE conversations SET status='abandoned', ended_at=last_activity_at WHERE id=?", c.ID)
	}
	return out, nil
}

func (s *Store) InsertMessage(convID int64, role, content, quickRepliesJSON string) (*models.Message, error) {
	res, err := s.DB.Exec("INSERT INTO messages (conversation_id, role, content, quick_replies) VALUES (?,?,?,?)",
		convID, role, content, nilIfEmpty(quickRepliesJSON))
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	s.DB.Exec("UPDATE conversations SET message_count=message_count+1, last_activity_at=NOW() WHERE id=?", convID)
	m := &models.Message{ID: id, ConversationID: convID, Role: role, Content: content, CreatedAt: time.Now().UTC()}
	if quickRepliesJSON != "" {
		m.QuickReplies = sql.NullString{String: quickRepliesJSON, Valid: true}
	}
	return m, nil
}

func (s *Store) ListMessages(convID int64, afterID int64) ([]*models.Message, error) {
	rows, err := s.DB.Query("SELECT id, conversation_id, role, content, quick_replies, created_at FROM messages WHERE conversation_id=? AND id>? ORDER BY id", convID, afterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Message
	for rows.Next() {
		m := &models.Message{}
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.QuickReplies, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) EnqueueDelivery(accountID int64, convID *int64, channel, kind, payload string) error {
	var cid any
	if convID != nil {
		cid = *convID
	}
	_, err := s.DB.Exec("INSERT INTO deliveries (account_id, conversation_id, channel, kind, payload) VALUES (?,?,?,?,?)",
		accountID, cid, channel, kind, payload)
	return err
}

func (s *Store) FetchDueDeliveries(limit int) ([]*models.Delivery, error) {
	rows, err := s.DB.Query(`SELECT id, account_id, conversation_id, channel, kind, status, attempts, last_error, payload, next_attempt_at, created_at, sent_at
		FROM deliveries WHERE status='pending' AND next_attempt_at<=NOW() ORDER BY id LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Delivery
	for rows.Next() {
		d := &models.Delivery{}
		if err := rows.Scan(&d.ID, &d.AccountID, &d.ConversationID, &d.Channel, &d.Kind, &d.Status, &d.Attempts, &d.LastError, &d.Payload, &d.NextAttemptAt, &d.CreatedAt, &d.SentAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) ClaimDelivery(id int64) (bool, error) {
	res, err := s.DB.Exec("UPDATE deliveries SET status='processing' WHERE id=? AND status='pending'", id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *Store) MarkDeliverySent(id int64) error {
	_, err := s.DB.Exec("UPDATE deliveries SET status='sent', sent_at=NOW() WHERE id=?", id)
	return err
}

func (s *Store) MarkDeliveryFailed(id int64, attempts int, errMsg string, maxAttempts int) error {
	status := "pending"
	if attempts >= maxAttempts {
		status = "failed"
	}
	backoff := time.Duration(attempts*attempts) * 30 * time.Second
	_, err := s.DB.Exec("UPDATE deliveries SET status=?, attempts=?, last_error=?, next_attempt_at=? WHERE id=?",
		status, attempts, truncate(errMsg, 5000), time.Now().UTC().Add(backoff), id)
	return err
}

func (s *Store) InsertFeedback(accountID, convID int64, original, corrected, note string) error {
	_, err := s.DB.Exec("INSERT INTO feedback (account_id, conversation_id, original_status, corrected_status, note) VALUES (?,?,?,?,?)",
		accountID, convID, original, corrected, nilIfEmpty(note))
	return err
}

type FeedbackExample struct {
	Summary         string
	OriginalStatus  string
	CorrectedStatus string
	Note            string
}

func (s *Store) RecentFeedback(accountID int64, limit int) ([]FeedbackExample, error) {
	rows, err := s.DB.Query(`SELECT COALESCE(c.summary,''), f.original_status, f.corrected_status, COALESCE(f.note,'')
		FROM feedback f JOIN conversations c ON c.id=f.conversation_id
		WHERE f.account_id=? ORDER BY f.id DESC LIMIT ?`, accountID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FeedbackExample
	for rows.Next() {
		var fe FeedbackExample
		if err := rows.Scan(&fe.Summary, &fe.OriginalStatus, &fe.CorrectedStatus, &fe.Note); err != nil {
			return nil, err
		}
		out = append(out, fe)
	}
	return out, rows.Err()
}

func (s *Store) InsertEvent(accountID int64, visitorID, eventType, pageURL string) error {
	_, err := s.DB.Exec("INSERT INTO events (account_id, visitor_id, type, page_url) VALUES (?,?,?,?)",
		accountID, visitorID, eventType, truncate(pageURL, 500))
	return err
}

type DailyPoint struct {
	Date          string `json:"date"`
	Conversations int    `json:"conversations"`
	Qualified     int    `json:"qualified"`
}

type Metrics struct {
	Conversations     int          `json:"conversations"`
	Qualified         int          `json:"qualified"`
	Disqualified      int          `json:"disqualified"`
	Abandoned         int          `json:"abandoned"`
	Handoff           int          `json:"handoff"`
	Active            int          `json:"active"`
	QualificationRate float64      `json:"qualification_rate"`
	AvgQualifySeconds float64      `json:"avg_qualify_seconds"`
	TimeSavedMinutes  float64      `json:"time_saved_minutes"`
	ConfidenceBuckets []int        `json:"confidence_buckets"`
	Daily             []DailyPoint `json:"daily"`
	WidgetLoads       int          `json:"widget_loads"`
	ChatOpens         int          `json:"chat_opens"`
	OpenRate          float64      `json:"open_rate"`
}

func (s *Store) GetMetrics(accountID int64, days, manualScreenMinutes int) (*Metrics, error) {
	since := time.Now().UTC().AddDate(0, 0, -days)
	m := &Metrics{ConfidenceBuckets: make([]int, 5)}

	rows, err := s.DB.Query("SELECT status, COUNT(*) FROM conversations WHERE account_id=? AND started_at>=? GROUP BY status", accountID, since)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var status string
		var n int
		if err := rows.Scan(&status, &n); err != nil {
			rows.Close()
			return nil, err
		}
		m.Conversations += n
		switch status {
		case "qualified":
			m.Qualified = n
		case "disqualified":
			m.Disqualified = n
		case "abandoned":
			m.Abandoned = n
		case "handoff":
			m.Handoff = n
		case "active":
			m.Active = n
		}
	}
	rows.Close()

	completed := m.Qualified + m.Disqualified
	if completed > 0 {
		m.QualificationRate = float64(m.Qualified) / float64(completed) * 100
	}
	m.TimeSavedMinutes = float64(completed) * float64(manualScreenMinutes)

	s.DB.QueryRow(`SELECT COALESCE(AVG(TIMESTAMPDIFF(SECOND, started_at, ended_at)),0) FROM conversations
		WHERE account_id=? AND started_at>=? AND ended_at IS NOT NULL AND status IN ('qualified','disqualified')`, accountID, since).
		Scan(&m.AvgQualifySeconds)

	crows, err := s.DB.Query("SELECT confidence FROM conversations WHERE account_id=? AND started_at>=? AND confidence IS NOT NULL", accountID, since)
	if err != nil {
		return nil, err
	}
	for crows.Next() {
		var c int
		if err := crows.Scan(&c); err != nil {
			crows.Close()
			return nil, err
		}
		idx := c / 20
		if idx > 4 {
			idx = 4
		}
		if idx < 0 {
			idx = 0
		}
		m.ConfidenceBuckets[idx]++
	}
	crows.Close()

	drows, err := s.DB.Query(`SELECT DATE(started_at), COUNT(*), SUM(status='qualified') FROM conversations
		WHERE account_id=? AND started_at>=? GROUP BY DATE(started_at) ORDER BY DATE(started_at)`, accountID, since)
	if err != nil {
		return nil, err
	}
	byDate := map[string]DailyPoint{}
	for drows.Next() {
		var d time.Time
		var p DailyPoint
		if err := drows.Scan(&d, &p.Conversations, &p.Qualified); err != nil {
			drows.Close()
			return nil, err
		}
		p.Date = d.Format("2006-01-02")
		byDate[p.Date] = p
	}
	drows.Close()
	for i := days - 1; i >= 0; i-- {
		date := time.Now().UTC().AddDate(0, 0, -i).Format("2006-01-02")
		p, ok := byDate[date]
		if !ok {
			p = DailyPoint{Date: date}
		}
		m.Daily = append(m.Daily, p)
	}

	erows, err := s.DB.Query("SELECT type, COUNT(*) FROM events WHERE account_id=? AND created_at>=? GROUP BY type", accountID, since)
	if err != nil {
		return nil, err
	}
	for erows.Next() {
		var t string
		var n int
		if err := erows.Scan(&t, &n); err != nil {
			erows.Close()
			return nil, err
		}
		switch t {
		case "loaded":
			m.WidgetLoads = n
		case "opened":
			m.ChatOpens = n
		}
	}
	erows.Close()
	if m.WidgetLoads > 0 {
		m.OpenRate = float64(m.ChatOpens) / float64(m.WidgetLoads) * 100
	}
	return m, nil
}

type WeeklyStats struct {
	Conversations int
	Qualified     int
	Disqualified  int
	TopLeads      []string
}

func (s *Store) GetWeeklyStats(accountID int64) (*WeeklyStats, error) {
	since := time.Now().UTC().AddDate(0, 0, -7)
	ws := &WeeklyStats{}
	err := s.DB.QueryRow(`SELECT COUNT(*), COALESCE(SUM(status='qualified'),0), COALESCE(SUM(status='disqualified'),0)
		FROM conversations WHERE account_id=? AND started_at>=?`, accountID, since).
		Scan(&ws.Conversations, &ws.Qualified, &ws.Disqualified)
	if err != nil {
		return nil, err
	}
	rows, err := s.DB.Query(`SELECT COALESCE(contact,''), COALESCE(score,0), COALESCE(summary,'') FROM conversations
		WHERE account_id=? AND started_at>=? AND status='qualified' ORDER BY score DESC LIMIT 5`, accountID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var contact, summary string
		var score int
		if err := rows.Scan(&contact, &score, &summary); err != nil {
			return nil, err
		}
		ws.TopLeads = append(ws.TopLeads, fmt.Sprintf("Score %d — %s — %s", score, contact, truncate(summary, 120)))
	}
	return ws, rows.Err()
}

func (s *Store) AccountsDueWeeklyReport() ([]*models.Account, error) {
	rows, err := s.DB.Query("SELECT a." + strings.ReplaceAll(accountCols, ", ", ", a.") + ` FROM accounts a
		LEFT JOIN weekly_reports w ON w.account_id=a.id
		WHERE w.account_id IS NULL OR w.last_sent_at < DATE_SUB(NOW(), INTERVAL 7 DAY)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Account
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) MarkWeeklySent(accountID int64) error {
	_, err := s.DB.Exec(`INSERT INTO weekly_reports (account_id, last_sent_at) VALUES (?, NOW())
		ON DUPLICATE KEY UPDATE last_sent_at=NOW()`, accountID)
	return err
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nilIfZero(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func unmarshal(raw string, dst any) {
	_ = json.Unmarshal([]byte(raw), dst)
}
