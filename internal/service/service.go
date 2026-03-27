package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"socialpilot/internal/config"
	"socialpilot/internal/db"
	"socialpilot/internal/llm"
)

var (
	globalDB   *sqlx.DB
	globalLock sync.RWMutex
)

type Service struct {
	DB  *sqlx.DB
	LLM *llm.Client
	Now func() time.Time
}

type Advice struct {
	Tone    string `json:"tone"`
	Content string `json:"content"`
}

type ContactInput struct {
	Name   string
	Gender string
	Tags   string
}

func New(dbx *sqlx.DB, llmClient *llm.Client) *Service {
	return &Service{DB: dbx, LLM: llmClient, Now: time.Now}
}

func (s *Service) AddContact(in ContactInput) (db.Contact, error) {
	now := s.Now().UTC()
	c := db.Contact{
		ID:             uuid.NewString(),
		Name:           strings.TrimSpace(in.Name),
		Gender:         normalizeGender(in.Gender),
		Tags:           strings.TrimSpace(in.Tags),
		ProfileSummary: "",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	_, err := s.DB.Exec(`INSERT INTO contacts(id,name,gender,tags,profile_summary,created_at,updated_at) VALUES (?,?,?,?,?,?,?)`,
		c.ID, c.Name, c.Gender, c.Tags, c.ProfileSummary, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		return db.Contact{}, err
	}
	return c, nil
}

func (s *Service) IngestLog(name, rawMsg string) (string, int, error) {
	contact, err := s.ensureContact(name, "unknown", "", true)
	if err != nil {
		return "", 0, err
	}
	session, err := s.ensureActiveSession(contact.ID)
	if err != nil {
		return "", 0, err
	}
	now := s.Now().UTC()
	_, err = s.DB.Exec(`INSERT INTO raw_logs(id,contact_id,raw_text,created_at) VALUES (?,?,?,?)`,
		uuid.NewString(), contact.ID, rawMsg, now)
	if err != nil {
		return "", 0, err
	}

	// Try to parse as JSON array first (direct JSON import)
	messages, err := s.tryParseJSONLog(rawMsg, contact.Name)
	if err == nil && len(messages) > 0 {
		return s.insertMessages(session.ID, messages, now)
	}

	// Fall back to LLM parsing for unstructured text
	systemPrompt := llm.BuildExtractSystem(contact.Name, contact.Gender)
	j, err := s.LLM.ChatJSON(systemPrompt, rawMsg)
	if err != nil {
		return "", 0, fmt.Errorf("llm: %w", err)
	}
	var payload struct {
		Messages []struct {
			Speaker string `json:"speaker"`
			Content string `json:"content"`
			Emotion string `json:"emotion"`
			Intent  string `json:"intent"`
		} `json:"messages"`
	}
	if err := json.Unmarshal([]byte(j), &payload); err != nil {
		return "", 0, fmt.Errorf("llm-parse: %w", err)
	}
	if len(payload.Messages) == 0 {
		return session.ID, 0, nil
	}

	// Convert to internal format
	msgs := make([]struct {
		Speaker string
		Content string
		Emotion string
		Intent  string
	}, len(payload.Messages))
	for i, m := range payload.Messages {
		msgs[i].Speaker = m.Speaker
		msgs[i].Content = m.Content
		msgs[i].Emotion = m.Emotion
		msgs[i].Intent = m.Intent
	}
	return s.insertMessages(session.ID, msgs, now)
}

// tryParseJSONLog attempts to parse rawMsg as a JSON array of chat messages.
// Supports formats like: [{"sender": "name", "message": "text"}, ...]
// or: [{"speaker": "name", "content": "text"}, ...]
func (s *Service) tryParseJSONLog(rawMsg, contactName string) ([]struct {
	Speaker string
	Content string
	Emotion string
	Intent  string
}, error) {
	rawMsg = strings.TrimSpace(rawMsg)
	if !strings.HasPrefix(rawMsg, "[") {
		return nil, fmt.Errorf("not a JSON array")
	}

	// Try multiple JSON formats
	var rawMessages []map[string]any
	if err := json.Unmarshal([]byte(rawMsg), &rawMessages); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w", err)
	}

	result := make([]struct {
		Speaker string
		Content string
		Emotion string
		Intent  string
	}, 0, len(rawMessages))

	for _, m := range rawMessages {
		// Try different field names for sender
		speaker := ""
		for _, key := range []string{"sender", "speaker", "from", "name", "user", "author"} {
			if v, ok := m[key]; ok {
				speaker = fmt.Sprintf("%v", v)
				break
			}
		}

		// Try different field names for content
		content := ""
		for _, key := range []string{"message", "content", "text", "msg", "body"} {
			if v, ok := m[key]; ok {
				content = fmt.Sprintf("%v", v)
				break
			}
		}

		if content == "" {
			continue
		}

		// Normalize speaker: contact name -> "contact", "我"/"self"/"user" -> "user"
		speaker = normalizeSpeaker(speaker, contactName)

		result = append(result, struct {
			Speaker string
			Content string
			Emotion string
			Intent  string
		}{
			Speaker: speaker,
			Content: strings.TrimSpace(content),
			Emotion: "",
			Intent:  "",
		})
	}

	return result, nil
}

func (s *Service) insertMessages(sessionID string, messages []struct {
	Speaker string
	Content string
	Emotion string
	Intent  string
}, now time.Time) (string, int, error) {
	if len(messages) == 0 {
		return sessionID, 0, nil
	}

	tx, err := s.DB.Beginx()
	if err != nil {
		return "", 0, err
	}
	defer tx.Rollback()

	for _, m := range messages {
		if _, err := tx.Exec(`INSERT INTO messages(id,session_id,speaker,content,emotion,intent,timestamp) VALUES (?,?,?,?,?,?,?)`,
			uuid.NewString(), sessionID, m.Speaker, strings.TrimSpace(m.Content), strings.TrimSpace(m.Emotion), strings.TrimSpace(m.Intent), now); err != nil {
			return "", 0, err
		}
	}
	if _, err := tx.Exec(`UPDATE sessions SET updated_at=? WHERE id=?`, now, sessionID); err != nil {
		return "", 0, err
	}
	if err := tx.Commit(); err != nil {
		return "", 0, err
	}

	return sessionID, len(messages), nil
}

func (s *Service) ChatAdvice(name, incoming string) ([]Advice, string, error) {
	contact, err := s.ensureContact(name, "unknown", "", true)
	if err != nil {
		return nil, "", err
	}
	session, err := s.ensureActiveSession(contact.ID)
	if err != nil {
		return nil, "", err
	}
	now := s.Now().UTC()
	if _, err := s.DB.Exec(`INSERT INTO messages(id,session_id,speaker,content,emotion,intent,timestamp) VALUES (?,?,?,?,?,?,?)`,
		uuid.NewString(), session.ID, "contact", strings.TrimSpace(incoming), "", "", now); err != nil {
		return nil, "", err
	}
	if _, err := s.DB.Exec(`UPDATE sessions SET updated_at=? WHERE id=?`, now, session.ID); err != nil {
		return nil, "", err
	}

	var summaries []string
	if err := s.DB.Select(&summaries, `SELECT summary FROM sessions WHERE contact_id=? AND status='compressed' AND summary<>'' ORDER BY updated_at DESC LIMIT 5`, contact.ID); err != nil {
		return nil, "", err
	}

	var recent []db.Message
	if err := s.DB.Select(&recent, `SELECT id,session_id,speaker,content,emotion,intent,timestamp FROM messages WHERE session_id=? ORDER BY timestamp DESC LIMIT 20`, session.ID); err != nil {
		return nil, "", err
	}
	reverseMessages(recent)
	recentRows := make([]string, 0, len(recent))
	for _, m := range recent {
		extra := ""
		if m.Emotion != "" || m.Intent != "" {
			extra = fmt.Sprintf(" (%s/%s)", m.Emotion, m.Intent)
		}
		recentRows = append(recentRows, fmt.Sprintf("%s: %s%s", m.Speaker, m.Content, extra))
	}

	profile := contact.ProfileSummary
	if strings.TrimSpace(profile) == "" {
		profile = "暂无画像"
	}
	bg := strings.Join(summaries, "\n")
	if strings.TrimSpace(bg) == "" {
		bg = "暂无长期记忆"
	}
	recentText := strings.Join(recentRows, "\n")
	if strings.TrimSpace(recentText) == "" {
		recentText = "暂无近期对话"
	}

	systemPrompt := llm.BuildCopilotSystem(profile, bg, recentText, incoming)
	j, err := s.LLM.ChatJSON(systemPrompt, "请按要求输出 JSON")
	if err != nil {
		return nil, "", fmt.Errorf("llm: %w", err)
	}
	var out struct {
		Advice []Advice `json:"advice"`
	}
	if err := json.Unmarshal([]byte(j), &out); err != nil {
		return nil, "", fmt.Errorf("llm-parse: %w", err)
	}
	if out.Advice == nil {
		out.Advice = []Advice{}
	}
	return out.Advice, session.ID, nil
}

func (s *Service) CommitMessage(name, content string) (string, error) {
	contact, err := s.ensureContact(name, "unknown", "", true)
	if err != nil {
		return "", err
	}
	session, err := s.ensureActiveSession(contact.ID)
	if err != nil {
		return "", err
	}
	now := s.Now().UTC()
	if _, err := s.DB.Exec(`INSERT INTO messages(id,session_id,speaker,content,emotion,intent,timestamp) VALUES (?,?,?,?,?,?,?)`,
		uuid.NewString(), session.ID, "user", strings.TrimSpace(content), "", "", now); err != nil {
		return "", err
	}
	if _, err := s.DB.Exec(`UPDATE sessions SET updated_at=? WHERE id=?`, now, session.ID); err != nil {
		return "", err
	}
	return session.ID, nil
}

func (s *Service) AnalyzeContact(name string) (string, error) {
	contact, err := s.getContactByName(name)
	if err != nil {
		return "", err
	}
	var rows []struct {
		Speaker string    `db:"speaker"`
		Content string    `db:"content"`
		Ts      time.Time `db:"timestamp"`
	}
	q := `
SELECT m.speaker,m.content,m.timestamp
FROM messages m
JOIN sessions s ON s.id = m.session_id
WHERE s.contact_id=?
ORDER BY m.timestamp DESC
LIMIT 120`
	if err := s.DB.Select(&rows, q, contact.ID); err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", fmt.Errorf("no messages for contact")
	}
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	lines := make([]string, 0, len(rows))
	for _, r := range rows {
		lines = append(lines, fmt.Sprintf("[%s] %s: %s", r.Ts.Format(time.RFC3339), r.Speaker, r.Content))
	}
	result, err := s.LLM.Chat(llm.PromptAnalyze, strings.Join(lines, "\n"))
	if err != nil {
		return "", fmt.Errorf("llm: %w", err)
	}
	if _, err := s.DB.Exec(`UPDATE contacts SET profile_summary=?,updated_at=? WHERE id=?`, result, s.Now().UTC(), contact.ID); err != nil {
		return "", err
	}
	return result, nil
}

func (s *Service) Compress(all bool, name string) (int, error) {
	contactIDs := []string{}
	if all {
		if err := s.DB.Select(&contactIDs, `SELECT id FROM contacts`); err != nil {
			return 0, err
		}
	} else {
		c, err := s.getContactByName(name)
		if err != nil {
			return 0, err
		}
		contactIDs = append(contactIDs, c.ID)
	}

	cutoff := s.Now().UTC().Add(-7 * 24 * time.Hour)
	compressed := 0
	for _, cid := range contactIDs {
		var sessions []db.Session
		q := `
SELECT id,contact_id,parent_session_id,status,summary,created_at,updated_at
FROM sessions
WHERE contact_id=? AND status IN ('active','closed') AND updated_at < ?
ORDER BY updated_at ASC`
		if err := s.DB.Select(&sessions, q, cid, cutoff); err != nil {
			return compressed, err
		}
		for _, sess := range sessions {
			var msgs []db.Message
			if err := s.DB.Select(&msgs, `SELECT id,session_id,speaker,content,emotion,intent,timestamp FROM messages WHERE session_id=? ORDER BY timestamp ASC LIMIT 200`, sess.ID); err != nil {
				return compressed, err
			}
			if len(msgs) == 0 {
				continue
			}
			rows := make([]string, 0, len(msgs))
			for _, m := range msgs {
				rows = append(rows, fmt.Sprintf("%s: %s", m.Speaker, m.Content))
			}
			summary, err := s.LLM.Chat(llm.PromptCompress, strings.Join(rows, "\n"))
			if err != nil {
				return compressed, fmt.Errorf("llm: %w", err)
			}
			if _, err := s.DB.Exec(`UPDATE sessions SET summary=?,status='compressed',updated_at=? WHERE id=?`, strings.TrimSpace(summary), s.Now().UTC(), sess.ID); err != nil {
				return compressed, err
			}
			compressed++
		}
	}
	return compressed, nil
}

func (s *Service) getContactByName(name string) (db.Contact, error) {
	var c db.Contact
	err := s.DB.Get(&c, `SELECT id,name,gender,tags,profile_summary,created_at,updated_at FROM contacts WHERE name=?`, strings.TrimSpace(name))
	if err != nil {
		return db.Contact{}, err
	}
	return c, nil
}

func (s *Service) ensureContact(name, gender, tags string, createIfMissing bool) (db.Contact, error) {
	c, err := s.getContactByName(name)
	if err == nil {
		return c, nil
	}
	if err != sql.ErrNoRows {
		return db.Contact{}, err
	}
	if !createIfMissing {
		return db.Contact{}, err
	}
	return s.AddContact(ContactInput{Name: name, Gender: gender, Tags: tags})
}

func (s *Service) ensureActiveSession(contactID string) (db.Session, error) {
	var sess db.Session
	err := s.DB.Get(&sess, `SELECT id,contact_id,parent_session_id,status,summary,created_at,updated_at FROM sessions WHERE contact_id=? AND status='active' ORDER BY updated_at DESC LIMIT 1`, contactID)
	if err == nil {
		return sess, nil
	}
	if err != sql.ErrNoRows {
		return db.Session{}, err
	}

	var parentID *string
	var latestID string
	if e := s.DB.Get(&latestID, `SELECT id FROM sessions WHERE contact_id=? ORDER BY updated_at DESC LIMIT 1`, contactID); e == nil {
		parentID = &latestID
	}
	now := s.Now().UTC()
	sess = db.Session{
		ID:              uuid.NewString(),
		ContactID:       contactID,
		ParentSessionID: parentID,
		Status:          "active",
		Summary:         "",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	_, err = s.DB.Exec(`INSERT INTO sessions(id,contact_id,parent_session_id,status,summary,created_at,updated_at) VALUES (?,?,?,?,?,?,?)`,
		sess.ID, sess.ContactID, sess.ParentSessionID, sess.Status, sess.Summary, sess.CreatedAt, sess.UpdatedAt)
	if err != nil {
		return db.Session{}, err
	}
	return sess, nil
}

func normalizeGender(g string) string {
	s := strings.ToLower(strings.TrimSpace(g))
	switch s {
	case "male", "female", "other", "unknown":
		return s
	default:
		return "unknown"
	}
}

func normalizeSpeaker(raw, contactName string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	// Check if this is the contact
	if s == "contact" || s == strings.ToLower(contactName) {
		return "contact"
	}
	// Check if this is the user (self)
	userAliases := []string{"user", "我", "self", "me", "自己", "本人", "主"}
	for _, alias := range userAliases {
		if s == alias {
			return "user"
		}
	}
	// If the raw name matches contact name (case insensitive), treat as contact
	if contactName != "" && strings.Contains(s, strings.ToLower(contactName)) {
		return "contact"
	}
	// Default: treat unknown names as user (safer assumption for imported logs)
	return "user"
}

func reverseMessages(rows []db.Message) {
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
}

// OpenService opens a service with global database connection pooling.
// The cleanup function only closes the LLM client if needed, not the database.
func OpenService(requireLLM bool) (*Service, func(), error) {
	cfg, _, err := config.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("load config failed: %w", err)
	}

	dbx, err := getOrOpenDB(cfg.DBPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open database failed: %w", err)
	}

	// Load custom prompts from config
	llm.SetPrompts(cfg.PromptExtract, cfg.PromptCopilot, cfg.PromptAnalyze, cfg.PromptCompress)

	var llmClient *llm.Client
	if requireLLM {
		if strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.APIKey) == "" || strings.TrimSpace(cfg.Model) == "" {
			return nil, nil, fmt.Errorf("missing llm config, run: socialpilot config set --baseurl ... --apikey ... --model ...")
		}
		llmClient = llm.New(cfg.BaseURL, cfg.APIKey, cfg.Model, time.Duration(cfg.TimeoutSeconds)*time.Second)
	}

	cleanup := func() {
		// Database connection is pooled, don't close it here
	}
	return New(dbx, llmClient), cleanup, nil
}

// getOrOpenDB returns the global database connection, creating it if necessary.
func getOrOpenDB(path string) (*sqlx.DB, error) {
	globalLock.RLock()
	if globalDB != nil {
		globalLock.RUnlock()
		return globalDB, nil
	}
	globalLock.RUnlock()

	globalLock.Lock()
	defer globalLock.Unlock()

	// Double check after acquiring write lock
	if globalDB != nil {
		return globalDB, nil
	}

	dbx, err := db.Open(path)
	if err != nil {
		return nil, err
	}
	globalDB = dbx
	return globalDB, nil
}

// CloseDB closes the global database connection. Useful for graceful shutdown.
func CloseDB() error {
	globalLock.Lock()
	defer globalLock.Unlock()

	if globalDB != nil {
		err := globalDB.Close()
		globalDB = nil
		return err
	}
	return nil
}
