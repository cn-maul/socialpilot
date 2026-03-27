package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"socialpilot/internal/config"
	"socialpilot/internal/db"
	"socialpilot/internal/llm"
	"socialpilot/internal/service"
)

func newWebCmd() *cobra.Command {
	var host string
	var port int
	c := &cobra.Command{
		Use:   "web",
		Short: "Start SocialPilot web app",
		RunE: func(cmd *cobra.Command, args []string) error {
			addr := fmt.Sprintf("%s:%d", host, port)
			mux := http.NewServeMux()
			registerWebUI(mux)

			mux.HandleFunc("/api/config/get", handleConfigGet)
			mux.HandleFunc("/api/config/set", handleConfigSet)
			mux.HandleFunc("/api/prompts/get", handlePromptsGet)
			mux.HandleFunc("/api/prompts/set", handlePromptsSet)
			mux.HandleFunc("/api/prompts/reset", handlePromptsReset)

			mux.HandleFunc("/api/contact/add", handleContactAdd)
			mux.HandleFunc("/api/contact/delete", handleContactDelete)
			mux.HandleFunc("/api/contact/search", handleContactSearch)
			mux.HandleFunc("/api/contact/detail", handleContactDetail)

			mux.HandleFunc("/api/log", handleLog)
			mux.HandleFunc("/api/chat", handleChat)
			mux.HandleFunc("/api/commit", handleCommit)
			mux.HandleFunc("/api/analyze", handleAnalyze)
			mux.HandleFunc("/api/compress", handleCompress)

			srv := &http.Server{
				Addr:              addr,
				Handler:           mux,
				ReadHeaderTimeout: 10 * time.Second,
			}
			fmt.Printf("Web UI running at http://%s\n", addr)
			return srv.ListenAndServe()
		},
	}
	c.Flags().StringVar(&host, "host", "127.0.0.1", "Host to bind")
	c.Flags().IntVar(&port, "port", 8080, "Port to bind")
	return c
}

func registerWebUI(mux *http.ServeMux) {
	distDir := filepath.Join("webui", "dist")
	indexPath := filepath.Join(distDir, "index.html")
	distAbs, _ := filepath.Abs(distDir)
	if st, err := os.Stat(indexPath); err == nil && !st.IsDir() {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}

			rel := strings.TrimPrefix(filepath.Clean(r.URL.Path), "/")
			if rel == "." || rel == "" {
				http.ServeFile(w, r, indexPath)
				return
			}

			target := filepath.Join(distDir, rel)
			targetAbs, _ := filepath.Abs(target)
			if strings.HasPrefix(targetAbs, distAbs+string(filepath.Separator)) || targetAbs == distAbs {
				if st, err := os.Stat(target); err == nil && !st.IsDir() {
					http.ServeFile(w, r, target)
					return
				}
			}

			http.ServeFile(w, r, indexPath)
		})
		return
	}

	mux.HandleFunc("/", handleIndex)
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(indexHTML))
}

func handleConfigGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	cfg, _, err := config.Load()
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeOK(w, map[string]any{
		"status": "success",
		"config": map[string]any{
			"baseurl":         cfg.BaseURL,
			"apikey":          maskAPIKey(cfg.APIKey),
			"model":           cfg.Model,
			"db_path":         cfg.DBPath,
			"timeout_seconds": cfg.TimeoutSeconds,
		},
	})
}

func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

func handleConfigSet(w http.ResponseWriter, r *http.Request) {
	if !isPOST(w, r) {
		return
	}
	var req struct {
		BaseURL string `json:"baseurl"`
		APIKey  string `json:"apikey"`
		Model   string `json:"model"`
		DBPath  string `json:"db_path"`
		Timeout int    `json:"timeout_seconds"`
	}
	if !decodeReq(w, r, &req) {
		return
	}
	cfg, p, err := config.Load()
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.BaseURL) != "" {
		cfg.BaseURL = strings.TrimSpace(req.BaseURL)
	}
	if strings.TrimSpace(req.APIKey) != "" {
		cfg.APIKey = strings.TrimSpace(req.APIKey)
	}
	if strings.TrimSpace(req.Model) != "" {
		cfg.Model = strings.TrimSpace(req.Model)
	}
	if strings.TrimSpace(req.DBPath) != "" {
		cfg.DBPath = strings.TrimSpace(req.DBPath)
	}
	if req.Timeout > 0 {
		cfg.TimeoutSeconds = req.Timeout
	}
	if err := config.Save(p, cfg); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeOK(w, map[string]any{"status": "success", "config_path": p})
}

func handlePromptsGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	cfg, _, err := config.Load()
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	// Get defaults
	defExtract, defCopilot, defAnalyze, defCompress := llm.GetDefaultPrompts()

	writeOK(w, map[string]any{
		"status": "success",
		"prompts": map[string]any{
			"extract":            cfg.PromptExtract,
			"copilot":            cfg.PromptCopilot,
			"analyze":            cfg.PromptAnalyze,
			"compress":           cfg.PromptCompress,
			"default_extract":    defExtract,
			"default_copilot":    defCopilot,
			"default_analyze":    defAnalyze,
			"default_compress":   defCompress,
		},
	})
}

func handlePromptsSet(w http.ResponseWriter, r *http.Request) {
	if !isPOST(w, r) {
		return
	}
	var req struct {
		Extract  string `json:"prompt_extract"`
		Copilot  string `json:"prompt_copilot"`
		Analyze  string `json:"prompt_analyze"`
		Compress string `json:"prompt_compress"`
	}
	if !decodeReq(w, r, &req) {
		return
	}
	cfg, p, err := config.Load()
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	// Update prompts (only if provided)
	if req.Extract != "" {
		cfg.PromptExtract = req.Extract
	}
	if req.Copilot != "" {
		cfg.PromptCopilot = req.Copilot
	}
	if req.Analyze != "" {
		cfg.PromptAnalyze = req.Analyze
	}
	if req.Compress != "" {
		cfg.PromptCompress = req.Compress
	}

	if err := config.Save(p, cfg); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	// Apply prompts immediately
	llm.SetPrompts(cfg.PromptExtract, cfg.PromptCopilot, cfg.PromptAnalyze, cfg.PromptCompress)

	writeOK(w, map[string]any{"status": "success"})
}

func handlePromptsReset(w http.ResponseWriter, r *http.Request) {
	if !isPOST(w, r) {
		return
	}
	cfg, p, err := config.Load()
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	// Clear custom prompts (use defaults)
	cfg.PromptExtract = ""
	cfg.PromptCopilot = ""
	cfg.PromptAnalyze = ""
	cfg.PromptCompress = ""

	if err := config.Save(p, cfg); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	// Reset to defaults in memory
	llm.ResetPrompts()

	writeOK(w, map[string]any{"status": "success"})
}

func handleContactAdd(w http.ResponseWriter, r *http.Request) {
	if !isPOST(w, r) {
		return
	}
	var req struct {
		Name   string `json:"name"`
		Gender string `json:"gender"`
		Tags   string `json:"tags"`
	}
	if !decodeReq(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeErr(w, http.StatusBadRequest, "name is required")
		return
	}
	svc, closeFn, err := openService(false)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	defer closeFn()

	contact, err := svc.AddContact(service.ContactInput{Name: req.Name, Gender: req.Gender, Tags: req.Tags})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeOK(w, map[string]any{"status": "success", "id": contact.ID, "name": contact.Name})
}

func handleContactDelete(w http.ResponseWriter, r *http.Request) {
	if !isPOST(w, r) {
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if !decodeReq(w, r, &req) {
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeErr(w, http.StatusBadRequest, "name is required")
		return
	}

	svc, closeFn, err := openService(false)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	defer closeFn()

	var contact db.Contact
	if err := svc.DB.Get(&contact, `
SELECT id,name,gender,tags,profile_summary,created_at,updated_at
FROM contacts
WHERE name=?`, name); err != nil {
		if err == sql.ErrNoRows {
			writeErr(w, http.StatusNotFound, "contact not found")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	tx, err := svc.DB.Beginx()
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
DELETE FROM messages
WHERE session_id IN (SELECT id FROM sessions WHERE contact_id=?)`, contact.ID); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := tx.Exec(`DELETE FROM raw_logs WHERE contact_id=?`, contact.ID); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := tx.Exec(`DELETE FROM sessions WHERE contact_id=?`, contact.ID); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := tx.Exec(`DELETE FROM contacts WHERE id=?`, contact.ID); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	writeOK(w, map[string]any{"status": "success", "name": contact.Name})
}

func handleContactSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	svc, closeFn, err := openService(false)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	defer closeFn()

	pattern := "%"
	if q != "" {
		pattern = "%" + q + "%"
	}
	var contacts []db.Contact
	err = svc.DB.Select(&contacts, `
SELECT id,name,gender,tags,profile_summary,created_at,updated_at
FROM contacts
WHERE name LIKE ?
ORDER BY updated_at DESC
LIMIT 100`, pattern)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeOK(w, map[string]any{"status": "success", "contacts": contacts})
}

func handleContactDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		writeErr(w, http.StatusBadRequest, "name is required")
		return
	}
	svc, closeFn, err := openService(false)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	defer closeFn()

	var contact db.Contact
	err = svc.DB.Get(&contact, `
SELECT id,name,gender,tags,profile_summary,created_at,updated_at
FROM contacts
WHERE name=?`, name)
	if err != nil {
		if err == sql.ErrNoRows {
			writeErr(w, http.StatusNotFound, "contact not found")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	type msgItem struct {
		Speaker   string    `db:"speaker" json:"speaker"`
		Content   string    `db:"content" json:"content"`
		Emotion   string    `db:"emotion" json:"emotion"`
		Intent    string    `db:"intent" json:"intent"`
		Timestamp time.Time `db:"timestamp" json:"timestamp"`
	}
	var messages []msgItem
	err = svc.DB.Select(&messages, `
SELECT m.speaker,m.content,m.emotion,m.intent,m.timestamp
FROM messages m
JOIN sessions s ON s.id = m.session_id
WHERE s.contact_id=?
ORDER BY m.timestamp DESC
LIMIT 50`, contact.ID)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	var sessionCount int
	_ = svc.DB.Get(&sessionCount, `SELECT COUNT(1) FROM sessions WHERE contact_id=?`, contact.ID)
	var messageCount int
	_ = svc.DB.Get(&messageCount, `
SELECT COUNT(1)
FROM messages m
JOIN sessions s ON s.id = m.session_id
WHERE s.contact_id=?`, contact.ID)
	var compressedCount int
	_ = svc.DB.Get(&compressedCount, `SELECT COUNT(1) FROM sessions WHERE contact_id=? AND status='compressed'`, contact.ID)

	var latestContactMsg string
	_ = svc.DB.Get(&latestContactMsg, `
SELECT m.content
FROM messages m
JOIN sessions s ON s.id = m.session_id
WHERE s.contact_id=? AND m.speaker='contact'
ORDER BY m.timestamp DESC
LIMIT 1`, contact.ID)
	var latestUserMsg string
	_ = svc.DB.Get(&latestUserMsg, `
SELECT m.content
FROM messages m
JOIN sessions s ON s.id = m.session_id
WHERE s.contact_id=? AND m.speaker='user'
ORDER BY m.timestamp DESC
LIMIT 1`, contact.ID)
	lastInteraction := ""
	if len(messages) > 0 {
		lastInteraction = messages[0].Timestamp.Format("2006-01-02 15:04:05")
	}

	writeOK(w, map[string]any{
		"status":  "success",
		"contact": contact,
		"stats": map[string]any{
			"session_count":      sessionCount,
			"message_count":      messageCount,
			"compressed_count":   compressedCount,
			"last_interaction":   lastInteraction,
			"latest_contact_msg": latestContactMsg,
			"latest_user_msg":    latestUserMsg,
		},
		"messages": messages,
	})
}

func handleLog(w http.ResponseWriter, r *http.Request) {
	if !isPOST(w, r) {
		return
	}
	var req struct {
		Name    string `json:"name"`
		Message string `json:"message"`
	}
	if !decodeReq(w, r, &req) {
		return
	}
	svc, closeFn, err := openService(true)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	defer closeFn()

	sessionID, n, err := svc.IngestLog(req.Name, req.Message)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeOK(w, map[string]any{"status": "success", "session_id": sessionID, "inserted_count": n})
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	if !isPOST(w, r) {
		return
	}
	var req struct {
		Name    string `json:"name"`
		Message string `json:"message"`
	}
	if !decodeReq(w, r, &req) {
		return
	}
	svc, closeFn, err := openService(true)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	defer closeFn()

	advice, sessionID, err := svc.ChatAdvice(req.Name, req.Message)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeOK(w, map[string]any{"status": "success", "session_id": sessionID, "advice": advice})
}

func handleCommit(w http.ResponseWriter, r *http.Request) {
	if !isPOST(w, r) {
		return
	}
	var req struct {
		Name    string `json:"name"`
		Message string `json:"message"`
	}
	if !decodeReq(w, r, &req) {
		return
	}
	svc, closeFn, err := openService(false)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	defer closeFn()

	sessionID, err := svc.CommitMessage(req.Name, req.Message)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeOK(w, map[string]any{"status": "success", "session_id": sessionID})
}

func handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if !isPOST(w, r) {
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if !decodeReq(w, r, &req) {
		return
	}
	svc, closeFn, err := openService(true)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	defer closeFn()

	summary, err := svc.AnalyzeContact(req.Name)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeOK(w, map[string]any{"status": "success", "profile_summary": summary})
}

func handleCompress(w http.ResponseWriter, r *http.Request) {
	if !isPOST(w, r) {
		return
	}
	var req struct {
		All  bool   `json:"all"`
		Name string `json:"name"`
	}
	if !decodeReq(w, r, &req) {
		return
	}
	svc, closeFn, err := openService(true)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	defer closeFn()

	count, err := svc.Compress(req.All, req.Name)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeOK(w, map[string]any{"status": "success", "compressed_count": count})
}

func openService(requireLLM bool) (*service.Service, func(), error) {
	return service.OpenService(requireLLM)
}

func isPOST(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return false
	}
	return true
}

func decodeReq(w http.ResponseWriter, r *http.Request, out any) bool {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return false
	}
	return true
}

func writeOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "error", "error": msg})
}

const indexHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width,initial-scale=1" />
  <title>SocialPilot</title>
  <style>
    :root {
      --bg: #eef2f8;
      --ink: #162033;
      --muted: #6b7690;
      --line: #dce3f0;
      --card: #ffffff;
      --primary: #2258ff;
      --primary-2: #4d7bff;
      --ok: #0b8f55;
      --warn: #b26a00;
      --danger: #c0392b;
      --shadow: 0 14px 36px rgba(14,31,80,0.10);
      --radius: 14px;
    }
    [data-theme="dark"] {
      --bg: #0f1420;
      --ink: #e8ecf7;
      --muted: #9eabc9;
      --line: #2a344c;
      --card: #182135;
      --primary: #4d7bff;
      --primary-2: #79a0ff;
      --shadow: 0 16px 38px rgba(0,0,0,0.45);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      color: var(--ink);
      font: 14px/1.5 "PingFang SC","Microsoft YaHei","Segoe UI",sans-serif;
      background:
        radial-gradient(circle at 10% 0%, #f7faff 0%, transparent 45%),
        radial-gradient(circle at 100% 10%, #e9efff 0%, transparent 40%),
        var(--bg);
    }
    .shell {
      min-height: 100vh;
      display: grid;
      grid-template-columns: 220px 1fr;
    }
    .nav {
      border-right: 1px solid var(--line);
      background: linear-gradient(180deg, #f8fbff, #f2f6ff);
      padding: 20px 14px;
    }
    [data-theme="dark"] .nav { background: linear-gradient(180deg, #141b2b, #101726); }
    .logo {
      padding: 6px 10px 16px;
      font-weight: 700;
      letter-spacing: .3px;
      color: #1a3174;
    }
    [data-theme="dark"] .logo { color: #b9cbff; }
    .logo small {
      display:block;
      color: var(--muted);
      font-weight: 500;
      margin-top: 4px;
    }
    .menu button {
      width: 100%;
      text-align: left;
      border: 1px solid transparent;
      border-radius: 10px;
      padding: 10px 12px;
      margin: 0 0 8px;
      background: transparent;
      color: #294078;
      font-weight: 600;
      cursor: pointer;
    }
    [data-theme="dark"] .menu button { color: #b4c3e8; }
    .menu button.active {
      background: #e8efff;
      border-color: #cbd9ff;
      color: #1d3ea8;
    }
    [data-theme="dark"] .menu button.active { background: #25314c; border-color: #3a4c72; color: #d5e0ff; }
    .main {
      padding: 20px;
    }
    .top {
      display: flex;
      align-items: baseline;
      justify-content: space-between;
      margin-bottom: 14px;
    }
    .top h1 {
      margin: 0;
      font-size: 22px;
      color: #13285c;
    }
    .top .hint {
      color: var(--muted);
      font-size: 13px;
    }
    .theme-toggle {
      border: 1px solid var(--line);
      border-radius: 10px;
      background: var(--card);
      color: var(--ink);
      padding: 8px 10px;
      cursor: pointer;
      font-weight: 700;
    }
    .page { display: none; }
    .page.active { display: block; }

    .layout {
      display: grid;
      grid-template-columns: 360px 1fr;
      gap: 14px;
      min-height: calc(100vh - 110px);
    }
    .card {
      background: var(--card);
      border: 1px solid var(--line);
      border-radius: var(--radius);
      box-shadow: var(--shadow);
      padding: 14px;
      transition: transform .2s ease, box-shadow .2s ease;
    }
    .card:hover { transform: translateY(-2px); box-shadow: 0 18px 36px rgba(14,31,80,0.14); }
    .card h3 {
      margin: 0 0 8px;
      color: #1e356d;
      font-size: 16px;
    }
    .desc {
      margin: 0 0 10px;
      color: var(--muted);
      font-size: 12px;
    }
    .row { margin-bottom: 10px; }
    label { display:block; color: #5c6783; font-size: 12px; margin-bottom: 4px; }
    input, textarea, select {
      width: 100%;
      border: 1px solid var(--line);
      border-radius: 10px;
      padding: 9px 10px;
      font: 13px "Consolas","Courier New",monospace;
      background: #fff;
    }
    [data-theme="dark"] input,
    [data-theme="dark"] textarea,
    [data-theme="dark"] select {
      background: #121a2b;
      color: #e8ecf7;
      border-color: #32405f;
    }
    textarea { min-height: 82px; resize: vertical; }
    .btn {
      border: none;
      border-radius: 10px;
      background: linear-gradient(120deg,var(--primary),var(--primary-2));
      color: #fff;
      padding: 9px 12px;
      font-weight: 700;
      cursor: pointer;
    }
    .btn.secondary {
      background: #edf2ff;
      color: #2448a8;
      border: 1px solid #cfdcff;
    }
    [data-theme="dark"] .btn.secondary {
      background: #25314c;
      color: #dbe6ff;
      border-color: #3c4f77;
    }
    .btn.warn {
      background: #fff3e1;
      color: var(--warn);
      border: 1px solid #ffd8a2;
    }
    .btnline { display:flex; gap:8px; flex-wrap: wrap; }

    .searchbar { display:flex; gap:8px; }
    .list {
      max-height: 290px;
      overflow: auto;
      border: 1px solid var(--line);
      border-radius: 10px;
      background: #fbfdff;
    }
    [data-theme="dark"] .list { background: #111a2a; }
    .item {
      border-bottom: 1px solid #edf1fa;
      padding: 10px;
      cursor: pointer;
      transition: background .18s ease, transform .18s ease;
    }
    .item:last-child { border-bottom: none; }
    .item:hover { background: #f3f7ff; transform: translateY(-1px); }
    .item.active { background: #edf3ff; }
    .item .name { font-weight: 700; color: #1f346b; }
    .item .meta { font-size: 12px; color: var(--muted); }
    [data-theme="dark"] .item:hover { background: #1f2940; }
    [data-theme="dark"] .item.active { background: #25314c; }
    .meta-line { display:flex; gap:6px; align-items:center; margin-top:4px; }

    .detail-head {
      display:flex;
      justify-content: space-between;
      gap:10px;
      align-items: flex-start;
      border-bottom: 1px solid var(--line);
      padding-bottom: 10px;
      margin-bottom: 10px;
    }
    .pill {
      display: inline-block;
      padding: 3px 8px;
      border-radius: 999px;
      background: #edf2ff;
      color: #2f4ba6;
      font-size: 12px;
      margin-right: 6px;
    }
    .pill.gender-male { background: #e8f1ff; color: #2053b4; border: 1px solid #c8dcff; }
    .pill.gender-female { background: #ffe8f2; color: #ad2a63; border: 1px solid #ffc6de; }
    .badge {
      display:inline-block;
      padding: 2px 7px;
      border-radius: 999px;
      font-size: 11px;
      font-weight: 700;
      border: 1px solid transparent;
    }
    .badge.male { background:#e8f1ff; color:#2053b4; border-color:#c8dcff; }
    .badge.female { background:#ffe8f2; color:#ad2a63; border-color:#ffc6de; }
    .badge.tag { background:#f2f5fb; color:#42527b; border-color:#d8e0ef; }
    .section {
      border: 1px solid var(--line);
      border-radius: 12px;
      padding: 10px;
      margin-bottom: 10px;
      background: #fbfdff;
    }
    [data-theme="dark"] .section { background: #131d2f; }
    .section h4 {
      margin: 0 0 8px;
      font-size: 14px;
      color: #1f3568;
    }
    .kv {
      display: grid;
      grid-template-columns: 110px 1fr;
      gap: 6px;
      margin-bottom: 6px;
      align-items: start;
    }
    .kv .k {
      color: #506089;
      font-size: 12px;
      font-weight: 700;
    }
    .kv .v {
      color: #1d2b49;
      font-size: 13px;
      white-space: pre-wrap;
      word-break: break-word;
    }
    .profile-grid {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 8px;
    }
    .profile-block {
      border: 1px solid #e4e9f5;
      border-radius: 10px;
      background: #fff;
      padding: 8px;
      min-height: 72px;
    }
    .profile-block .k {
      font-size: 12px;
      color: #4e5f87;
      margin-bottom: 4px;
      font-weight: 700;
    }
    .profile-block .v {
      font-size: 13px;
      color: #1d2b49;
      white-space: pre-wrap;
      word-break: break-word;
    }
    [data-theme="dark"] .kv .v,
    [data-theme="dark"] .profile-block .v {
      color: #d9e4ff;
    }
    .advice {
      border: 1px solid #d7e2ff;
      border-radius: 10px;
      background: #f4f7ff;
      padding: 8px;
      margin-bottom: 8px;
    }
    .advice .tone { font-size: 12px; color: #3b57b8; font-weight: 700; }
    .advice .txt { margin: 4px 0 8px; font-size: 13px; }

    .timeline {
      max-height: 280px;
      overflow: auto;
      border: 1px solid var(--line);
      border-radius: 10px;
      background: #fff;
    }
    [data-theme="dark"] .timeline { background: #111a2a; }
    .msg {
      border-bottom: 1px solid #edf1fa;
      padding: 8px 10px;
    }
    .msg:last-child { border-bottom: none; }
    .msg .head { font-size: 12px; color: #4d5e85; }
    .msg .text { margin-top: 4px; }
    .speaker-chip {
      display:inline-block;
      padding: 1px 7px;
      border-radius: 999px;
      font-size: 11px;
      margin-right: 5px;
      border: 1px solid transparent;
    }
    .speaker-user { background:#edf6ff; color:#1c5a99; border-color:#cfe5ff; }
    .speaker-contact { background:#f7efff; color:#6a35a8; border-color:#e3d3ff; }

    .status {
      margin-top: 8px;
      font-size: 12px;
      color: #3e507a;
      white-space: pre-wrap;
      word-break: break-word;
    }

    @media (max-width: 1080px) {
      .shell { grid-template-columns: 1fr; }
      .nav { border-right: none; border-bottom: 1px solid var(--line); }
      .layout { grid-template-columns: 1fr; min-height: auto; }
    }
  </style>
</head>
<body>
<div class="shell">
  <aside class="nav">
    <div class="logo">SocialPilot<small>本地社交关系助手</small></div>
    <div class="menu">
      <button id="menu_contacts" class="active" onclick="switchPage('contacts')">联系人与详情</button>
      <button id="menu_settings" onclick="switchPage('settings')">设置</button>
    </div>
  </aside>

  <main class="main">
    <div class="top">
      <h1 id="page_title">联系人与详情</h1>
      <button id="theme_btn" class="theme-toggle" onclick="toggleTheme()">暗色模式</button>
    </div>

    <section id="page_contacts" class="page active">
      <div class="layout">
        <div>
          <div class="card" style="margin-bottom:12px;">
            <h3>联系人搜索</h3>
            <p class="desc">按姓名关键字检索联系人，点击某人后右侧会加载人物详情与历史消息。</p>
            <div class="searchbar row">
              <input id="search_name" placeholder="输入姓名关键字，例如：林" />
              <button class="btn" onclick="searchContacts()">搜索</button>
            </div>
            <div id="search_status" class="status"></div>
            <div id="contact_list" class="list" style="margin-top:8px;"></div>
          </div>

          <div class="card">
            <h3>新增联系人</h3>
            <p class="desc">创建联系人后可在右侧查看并继续录入与分析。</p>
            <div class="row"><label>姓名</label><input id="add_name" /></div>
            <div class="row">
              <label>性别</label>
              <select id="add_gender">
                <option value="male" selected>男</option>
                <option value="female">女</option>
              </select>
            </div>
            <div class="row">
              <label>标签</label>
              <select id="add_tag">
                <option value="同事">同事</option>
                <option value="同学">同学</option>
                <option value="朋友" selected>朋友</option>
                <option value="亲戚">亲戚</option>
              </select>
            </div>
            <button class="btn" onclick="addContact()">创建联系人</button>
            <div id="add_status" class="status"></div>
          </div>
        </div>

        <div>
          <div class="card" id="detail_card">
            <div id="detail_empty" class="desc">请先从左侧选择一个联系人。</div>

            <div id="detail_content" style="display:none;">
              <div class="detail-head">
                <div>
                  <h3 id="d_name" style="margin-bottom:4px;"></h3>
                  <div>
                    <span class="pill" id="d_gender"></span>
                    <span class="pill" id="d_tag"></span>
                    <span class="pill" id="d_stats"></span>
                  </div>
                </div>
                <div class="btnline">
                  <button class="btn secondary" onclick="refreshDetail()">刷新详情</button>
                </div>
              </div>

              <div class="section">
                <h4>人物画像</h4>
                <div class="profile-block" style="margin-bottom:8px;">
                  <div class="k">人物速览</div>
                  <div class="kv"><div class="k">最近互动</div><div id="d_last" class="v">-</div></div>
                  <div class="kv"><div class="k">对方最近一句</div><div id="d_last_contact" class="v">-</div></div>
                  <div class="kv"><div class="k">我最近一句</div><div id="d_last_user" class="v">-</div></div>
                </div>
                <div class="profile-block" style="margin-bottom:8px;">
                  <div class="k">压缩介绍</div>
                  <div id="d_intro" class="v">暂无画像</div>
                </div>
                <div class="profile-grid">
                  <div class="profile-block">
                    <div class="k">核心特征</div>
                    <div id="d_core" class="v">-</div>
                  </div>
                  <div class="profile-block">
                    <div class="k">沟通偏好/雷区</div>
                    <div id="d_pref" class="v">-</div>
                  </div>
                  <div class="profile-block" style="grid-column: 1 / span 2;">
                    <div class="k">关系温度评估</div>
                    <div id="d_relation" class="v">-</div>
                  </div>
                </div>
              </div>

              <div class="section">
                <h4>结构化录入（Log）</h4>
                <div class="row"><label>原始描述</label><textarea id="d_log_text" placeholder="例如：今天她说方案不够清晰。"></textarea></div>
                <button class="btn" onclick="runLog()">执行录入</button>
                <div id="d_log_status" class="status"></div>
              </div>

              <div class="section">
                <h4>智能回复建议 + 采纳回写（闭环）</h4>
                <div class="row"><label>对方新消息</label><textarea id="d_chat_text" placeholder="例如：你们什么时候给新版方案？"></textarea></div>
                <button class="btn" onclick="runChat()">生成建议</button>
                <div id="d_chat_status" class="status"></div>
                <div id="d_advice_list" style="margin-top:8px;"></div>
                <div class="row"><label>最终回写内容（可编辑）</label><textarea id="d_commit_text"></textarea></div>
                <button class="btn secondary" onclick="runCommit()">采纳并回写</button>
                <div id="d_commit_status" class="status"></div>
              </div>

              <div class="section">
                <h4>分析与压缩</h4>
                <div class="btnline">
                  <button class="btn secondary" onclick="runAnalyze()">更新人物画像</button>
                  <button class="btn warn" onclick="runCompress()">压缩该联系人历史会话</button>
                </div>
                <div id="d_ops_status" class="status"></div>
              </div>

              <div class="section">
                <h4>最近消息（最多50条）</h4>
                <div id="d_messages" class="timeline"></div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>

    <section id="page_settings" class="page">
      <div class="card" style="max-width:760px;">
        <h3>AI 与系统设置</h3>
        <p class="desc">所有需要 AI 的功能都使用这里的配置。建议先保存后再去联系人页测试。</p>
        <div class="row"><label>Base URL</label><input id="cfg_baseurl" placeholder="https://qianfan.baidubce.com/v2/coding/v1" /></div>
        <div class="row"><label>API Key</label><input id="cfg_apikey" placeholder="输入 API Key" /></div>
        <div class="row"><label>Model</label><input id="cfg_model" placeholder="例如：minimax-m2.5" /></div>
        <div class="row"><label>超时（秒）</label><input id="cfg_timeout" placeholder="60" /></div>
        <div class="row"><label>数据库路径（可选）</label><input id="cfg_db" placeholder="/tmp/socialpilot.db" /></div>
        <button class="btn" onclick="saveConfig()">保存设置</button>
        <div id="cfg_status" class="status"></div>
      </div>
    </section>
  </main>
</div>

<script>
let currentPage = 'contacts';
let contacts = [];
let selectedName = '';
let lastAdvice = [];
let darkMode = false;

function setText(id, txt){
  const el = document.getElementById(id);
  if (el) { el.textContent = txt || ''; }
}
function val(id){
  const el = document.getElementById(id);
  return el ? el.value : '';
}
function esc(s){
  return String(s || '').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
}
function stripMd(md){
  return String(md || '')
    .replace(/^#+\s*/gm, '')
    .replace(/^\|.*\|$/gm, ' ')
    .replace(/^:?-{2,}:?$/gm, ' ')
    .replace(/\*\*(.*?)\*\*/g, '$1')
    .replace(/\|/g, ' ')
    .replace(/[-*]\s+/g, '')
    .replace(/\d+\.\s+/g, '')
    .replace(/\s+/g, ' ')
    .trim();
}
function clip(txt, n){
  const s = String(txt || '').trim();
  if (s.length <= n) return s;
  return s.slice(0, n) + '...';
}
function pickKeyLines(text, limit){
  const parts = String(text || '')
    .split(/[。；\n]/)
    .map(x => stripMd(x))
    .filter(Boolean);
  return clip(parts.slice(0, limit).join('；'), 180);
}
function extractProfileSections(md){
  const src = String(md || '');
  if (!src.trim()) {
    return {intro:'暂无画像', core:'-', pref:'-', relation:'-'};
  }
  const lines = src.split('\n');
  let key = '';
  const sections = {core:[], pref:[], relation:[], other:[]};
  for (const raw of lines) {
    const line = raw.trim();
    if (/^##+\s*/.test(line)) {
      if (line.includes('核心')) key = 'core';
      else if (line.includes('雷区') || line.includes('偏好')) key = 'pref';
      else if (line.includes('关系温度')) key = 'relation';
      else key = 'other';
      continue;
    }
    if (!line) continue;
    sections[key || 'other'].push(line);
  }
  const core = pickKeyLines(sections.core.join('\n'), 3) || '-';
  const pref = pickKeyLines(sections.pref.join('\n'), 3) || '-';
  const relation = pickKeyLines(sections.relation.join('\n'), 3) || '-';
  const introBase = [core, pref, relation].filter(x => x && x !== '-').join('；');
  const intro = clip(introBase || stripMd(src), 140) || '暂无画像';
  return {intro:intro, core:core, pref:pref, relation:relation};
}
function renderProfile(md){
  const p = extractProfileSections(md);
  setText('d_intro', p.intro);
  setText('d_core', p.core);
  setText('d_pref', p.pref);
  setText('d_relation', p.relation);
}
function normName(c){
  return (c && (c.name || c.Name)) ? (c.name || c.Name) : '';
}
function normGender(raw){
  const g = String(raw || '').toLowerCase();
  if (g === 'female') return {label:'女', cls:'female'};
  return {label:'男', cls:'male'};
}

async function apiGet(url){
  const res = await fetch(url);
  return await res.json();
}
async function apiPost(url, body){
  const res = await fetch(url, {
    method:'POST',
    headers:{'Content-Type':'application/json'},
    body:JSON.stringify(body)
  });
  return await res.json();
}

function switchPage(page){
  currentPage = page;
  document.getElementById('menu_contacts').classList.toggle('active', page === 'contacts');
  document.getElementById('menu_settings').classList.toggle('active', page === 'settings');
  document.getElementById('page_contacts').classList.toggle('active', page === 'contacts');
  document.getElementById('page_settings').classList.toggle('active', page === 'settings');
  setText('page_title', page === 'contacts' ? '联系人与详情' : '设置');
}
function applyTheme(){
  document.documentElement.setAttribute('data-theme', darkMode ? 'dark' : 'light');
  setText('theme_btn', darkMode ? '浅色模式' : '暗色模式');
}
function toggleTheme(){
  darkMode = !darkMode;
  localStorage.setItem('sp_theme', darkMode ? 'dark' : 'light');
  applyTheme();
}

function renderContacts(){
  const box = document.getElementById('contact_list');
  if (!contacts.length) {
    box.innerHTML = '<div class="item"><div class="meta">没有匹配的联系人</div></div>';
    return;
  }
  box.innerHTML = contacts.map(function(c, idx){
    const name = normName(c);
    const genderRaw = c.gender || c.Gender || '';
    const gender = normGender(genderRaw);
    const tags = c.tags || c.Tags || '-';
    const ps = c.profile_summary || c.ProfileSummary || '';
    const active = name === selectedName ? ' active' : '';
    const profile = ps ? ps.slice(0, 30) + (ps.length > 30 ? '...' : '') : '暂无画像';
    return '<div class="item' + active + '" onclick="openContactByIndex(' + idx + ')">' +
      '<div class="name">' + esc(name) + '</div>' +
      '<div class="meta-line"><span class="badge ' + gender.cls + '">' + gender.label + '</span><span class="badge tag">' + esc(tags) + '</span></div>' +
      '<div class="meta">' + esc(profile) + '</div>' +
      '</div>';
  }).join('');
}

function openContactByIndex(idx){
  const c = contacts[idx];
  const name = normName(c);
  if (!name) return;
  openContact(name);
}

async function searchContacts(){
  const q = encodeURIComponent(val('search_name').trim());
  setText('search_status', '搜索中...');
  try {
    const ret = await apiGet('/api/contact/search?q=' + q);
    if (ret.status !== 'success') {
      setText('search_status', ret.error || '搜索失败');
      return;
    }
    contacts = ret.contacts || [];
    renderContacts();
    setText('search_status', '共找到 ' + contacts.length + ' 个联系人');
    if (!selectedName && contacts.length > 0) {
      openContact(normName(contacts[0]));
    }
  } catch (e) {
    setText('search_status', String(e));
  }
}

async function addContact(){
  const name = val('add_name').trim();
  if (!name) {
    setText('add_status', '请输入姓名');
    return;
  }
  setText('add_status', '创建中...');
  try {
    const ret = await apiPost('/api/contact/add', {
      name: name,
      gender: val('add_gender'),
      tags: val('add_tag')
    });
    if (ret.status !== 'success') {
      setText('add_status', ret.error || '创建失败');
      return;
    }
    setText('add_status', '创建成功：' + ret.name);
    document.getElementById('add_name').value = '';
    await searchContacts();
    openContact(ret.name || '');
  } catch (e) {
    setText('add_status', String(e));
  }
}

async function openContact(name){
  selectedName = name;
  renderContacts();
  await loadDetail();
}

async function refreshDetail(){
  if (!selectedName) { return; }
  await loadDetail();
}

function showDetail(on){
  document.getElementById('detail_empty').style.display = on ? 'none' : 'block';
  document.getElementById('detail_content').style.display = on ? 'block' : 'none';
}

async function loadDetail(){
  if (!selectedName) {
    showDetail(false);
    return;
  }
  showDetail(true);
  setText('d_name', selectedName + '（加载中）');
  setText('d_last', '-');
  setText('d_last_contact', '-');
  setText('d_last_user', '-');
  try {
    const ret = await apiGet('/api/contact/detail?name=' + encodeURIComponent(selectedName));
    if (ret.status !== 'success') {
      setText('d_name', selectedName);
      renderProfile(ret.error || '加载失败');
      return;
    }
    const c = ret.contact || {};
    const st = ret.stats || {};
    const msgs = ret.messages || [];

    const cname = c.name || c.Name || selectedName;
    const cgenderRaw = c.gender || c.Gender || 'male';
    const cgender = normGender(cgenderRaw);
    const ctags = c.tags || c.Tags || '-';
    const cprofile = c.profile_summary || c.ProfileSummary || '';

    setText('d_name', cname);
    const dGender = document.getElementById('d_gender');
    dGender.textContent = '性别：' + cgender.label;
    dGender.className = 'pill gender-' + cgender.cls;
    setText('d_tag', '标签：' + ctags);
    setText('d_stats', '会话 ' + (st.session_count || 0) + ' / 消息 ' + (st.message_count || 0) + ' / 压缩 ' + (st.compressed_count || 0));
    setText('d_last', st.last_interaction || '暂无');
    setText('d_last_contact', clip(st.latest_contact_msg || '暂无', 120));
    setText('d_last_user', clip(st.latest_user_msg || '暂无', 120));
    renderProfile(cprofile || '');

    const box = document.getElementById('d_messages');
    if (!msgs.length) {
      box.innerHTML = '<div class="msg"><div class="head">暂无消息</div></div>';
    } else {
      box.innerHTML = msgs.map(function(m){
        const speaker = String(m.speaker || '');
        const scls = speaker === 'user' ? 'speaker-user' : 'speaker-contact';
        const slabel = speaker === 'user' ? '我' : '对方';
        const em = m.emotion ? (' / ' + m.emotion) : '';
        const it = m.intent ? (' / ' + m.intent) : '';
        const ts = m.timestamp ? String(m.timestamp).replace('T',' ').slice(0,19) : '';
        return '<div class="msg">' +
          '<div class="head"><span class="speaker-chip ' + scls + '">' + slabel + '</span>' + em + it + ' · ' + esc(ts) + '</div>' +
          '<div class="text">' + esc(m.content) + '</div>' +
          '</div>';
      }).join('');
    }
  } catch (e) {
    setText('d_name', selectedName);
    renderProfile(String(e));
  }
}

async function runLog(){
  if (!selectedName) { setText('d_log_status', '请先选择联系人'); return; }
  const txt = val('d_log_text').trim();
  if (!txt) { setText('d_log_status', '请输入原始描述'); return; }
  setText('d_log_status', '执行中...');
  try {
    const ret = await apiPost('/api/log', { name: selectedName, message: txt });
    if (ret.status !== 'success') { setText('d_log_status', ret.error || '执行失败'); return; }
    setText('d_log_status', '成功：写入 ' + ret.inserted_count + ' 条，session=' + ret.session_id);
    document.getElementById('d_log_text').value = '';
    await loadDetail();
  } catch (e) {
    setText('d_log_status', String(e));
  }
}

function renderAdvice(list){
  const box = document.getElementById('d_advice_list');
  if (!list || !list.length) {
    box.innerHTML = '<div class="status">没有建议返回</div>';
    return;
  }
  box.innerHTML = list.map(function(a, i){
    return '<div class="advice">' +
      '<div class="tone">建议 ' + (i+1) + ' · ' + esc(a.tone || '未命名') + '</div>' +
      '<div class="txt">' + esc(a.content || '') + '</div>' +
      '<button class="btn secondary" onclick="pickAdvice(' + i + ')">采用此建议</button>' +
      '</div>';
  }).join('');
}

function pickAdvice(i){
  if (!lastAdvice[i]) { return; }
  document.getElementById('d_commit_text').value = lastAdvice[i].content || '';
}

async function runChat(){
  if (!selectedName) { setText('d_chat_status', '请先选择联系人'); return; }
  const txt = val('d_chat_text').trim();
  if (!txt) { setText('d_chat_status', '请输入对方新消息'); return; }
  setText('d_chat_status', '生成中...');
  try {
    const ret = await apiPost('/api/chat', { name: selectedName, message: txt });
    if (ret.status !== 'success') { setText('d_chat_status', ret.error || '生成失败'); return; }
    lastAdvice = ret.advice || [];
    renderAdvice(lastAdvice);
    if (lastAdvice.length) {
      document.getElementById('d_commit_text').value = lastAdvice[0].content || '';
    }
    setText('d_chat_status', '成功：已生成 ' + lastAdvice.length + ' 条建议');
    await loadDetail();
  } catch (e) {
    setText('d_chat_status', String(e));
  }
}

async function runCommit(){
  if (!selectedName) { setText('d_commit_status', '请先选择联系人'); return; }
  const txt = val('d_commit_text').trim();
  if (!txt) { setText('d_commit_status', '请填写最终回写内容'); return; }
  setText('d_commit_status', '提交中...');
  try {
    const ret = await apiPost('/api/commit', { name: selectedName, message: txt });
    if (ret.status !== 'success') { setText('d_commit_status', ret.error || '提交失败'); return; }
    setText('d_commit_status', '回写成功，session=' + ret.session_id);
    await loadDetail();
  } catch (e) {
    setText('d_commit_status', String(e));
  }
}

async function runAnalyze(){
  if (!selectedName) { setText('d_ops_status', '请先选择联系人'); return; }
  setText('d_ops_status', '分析中...');
  try {
    const ret = await apiPost('/api/analyze', { name: selectedName });
    if (ret.status !== 'success') { setText('d_ops_status', ret.error || '分析失败'); return; }
    setText('d_ops_status', '画像更新成功');
    await loadDetail();
  } catch (e) {
    setText('d_ops_status', String(e));
  }
}

async function runCompress(){
  if (!selectedName) { setText('d_ops_status', '请先选择联系人'); return; }
  setText('d_ops_status', '压缩中...');
  try {
    const ret = await apiPost('/api/compress', { all: false, name: selectedName });
    if (ret.status !== 'success') { setText('d_ops_status', ret.error || '压缩失败'); return; }
    setText('d_ops_status', '压缩成功：' + ret.compressed_count + ' 个会话');
  } catch (e) {
    setText('d_ops_status', String(e));
  }
}

async function loadConfig(){
  setText('cfg_status', '读取配置中...');
  try {
    const ret = await apiGet('/api/config/get');
    if (ret.status !== 'success') { setText('cfg_status', ret.error || '读取失败'); return; }
    const c = ret.config || {};
    document.getElementById('cfg_baseurl').value = c.baseurl || '';
    document.getElementById('cfg_apikey').value = c.apikey || '';
    document.getElementById('cfg_model').value = c.model || '';
    document.getElementById('cfg_db').value = c.db_path || '';
    document.getElementById('cfg_timeout').value = String(c.timeout_seconds || 60);
    setText('cfg_status', '配置已加载');
  } catch (e) {
    setText('cfg_status', String(e));
  }
}

async function saveConfig(){
  const t = parseInt(val('cfg_timeout'), 10);
  setText('cfg_status', '保存中...');
  try {
    const ret = await apiPost('/api/config/set', {
      baseurl: val('cfg_baseurl'),
      apikey: val('cfg_apikey'),
      model: val('cfg_model'),
      db_path: val('cfg_db'),
      timeout_seconds: Number.isNaN(t) ? 60 : t
    });
    if (ret.status !== 'success') { setText('cfg_status', ret.error || '保存失败'); return; }
    setText('cfg_status', '保存成功：' + (ret.config_path || ''));
  } catch (e) {
    setText('cfg_status', String(e));
  }
}

window.addEventListener('DOMContentLoaded', async function(){
  darkMode = localStorage.getItem('sp_theme') === 'dark';
  applyTheme();
  await loadConfig();
  await searchContacts();
});
</script>
</body>
</html>`
