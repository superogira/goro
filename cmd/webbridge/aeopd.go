package main

// AEOPD document-registry API for the in-game saraban NPC: the browser
// client cannot reach MySQL directly, so the bridge proxies two endpoints
// (same origin as the game through the nginx /aeopd/ location):
//
//	POST /aeopd/login  {"username","password"} -> {"token","name"}
//	GET  /aeopd/docs?token=..&page=N           -> paginated register rows
//
// Logins verify against the document system's own users table (bcrypt
// hashes, same as PHP password_verify). Tokens are random, kept in memory,
// and expire; the bridge restart forgets them, which is fine for a game UI.

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

// aeopdDSN comes only from the AEOPD_DSN environment variable — never
// bake database credentials into the source or the binary. The deploy
// host sets it in the service unit / start script (see docs/DEPLOY-WEB.txt).
var aeopdDSN = os.Getenv("AEOPD_DSN")

const (
	aeopdTokenTTL  = 2 * time.Hour
	aeopdPageRows  = 10
	aeopdLoginSlow = 300 * time.Millisecond
)

var aeopdDocTypeLabels = map[string]string{
	"receive":        "รับ",
	"receive_secret": "รับ (ลับ)",
	"send":           "ส่ง",
	"send_secret":    "ส่ง (ลับ)",
	"command":        "คำสั่ง",
	"rule":           "ระเบียบ",
	"regulation":     "ข้อบังคับ",
	"policy":         "นโยบาย",
	"other":          "อื่นๆ",
}

type aeopdStore struct {
	mu      sync.Mutex
	pool    *sql.DB
	tokens  map[string]aeopdToken
	lastTry time.Time
}

type aeopdToken struct {
	name    string
	expires time.Time
}

var aeopd = &aeopdStore{tokens: make(map[string]aeopdToken)}

// database returns the shared pool, opening it lazily. Failures are
// logged at most once a minute so an unreachable database does not flood
// the log.
func (s *aeopdStore) database() (*sql.DB, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if aeopdDSN == "" {
		log.Printf("aeopd: AEOPD_DSN is not set; /aeopd endpoints disabled")
		return nil, sql.ErrConnDone
	}
	if s.pool != nil {
		return s.pool, nil
	}
	if time.Since(s.lastTry) < time.Minute {
		return nil, sql.ErrConnDone
	}
	s.lastTry = time.Now()
	pool, err := sql.Open("mysql", aeopdDSN)
	if err != nil {
		return nil, err
	}
	pool.SetMaxOpenConns(2)
	pool.SetConnMaxIdleTime(time.Minute)
	if err := pool.Ping(); err != nil {
		pool.Close()
		return nil, err
	}
	s.pool = pool
	log.Printf("aeopd: database connected")
	return s.pool, nil
}

func (s *aeopdStore) issueToken(name string) string {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	token := hex.EncodeToString(buf)
	s.mu.Lock()
	defer s.mu.Unlock()
	// Opportunistic sweep of expired tokens.
	now := time.Now()
	for t, v := range s.tokens {
		if v.expires.Before(now) {
			delete(s.tokens, t)
		}
	}
	s.tokens[token] = aeopdToken{name: name, expires: now.Add(aeopdTokenTTL)}
	return token
}

func (s *aeopdStore) validToken(token string) (aeopdToken, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.tokens[token]
	if !ok || v.expires.Before(time.Now()) {
		if ok {
			delete(s.tokens, token)
		}
		return aeopdToken{}, false
	}
	return v, true
}

func aeopdWriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func handleAEOPDLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "post only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		aeopdWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request"})
		return
	}
	db, err := aeopd.database()
	if err != nil {
		aeopdWriteJSON(w, http.StatusBadGateway, map[string]string{"error": "database unreachable"})
		return
	}
	var hash, firstName, lastName string
	err = db.QueryRowContext(r.Context(),
		"SELECT password, first_name, last_name FROM users WHERE username = ?", req.Username).
		Scan(&hash, &firstName, &lastName)
	if err == nil {
		err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password))
	}
	// Uniform failure path: never reveal whether the user exists.
	if err != nil {
		time.Sleep(aeopdLoginSlow)
		aeopdWriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "ชื่อผู้ใช้หรือรหัสผ่านไม่ถูกต้อง"})
		return
	}
	token := aeopd.issueToken(firstName + " " + lastName)
	if token == "" {
		aeopdWriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "token issue failed"})
		return
	}
	aeopdWriteJSON(w, http.StatusOK, map[string]string{"token": token, "name": firstName + " " + lastName})
}

type aeopdDocRow struct {
	Type  string `json:"type"`
	Label string `json:"label"`
	Date  string `json:"date"`
	From  string `json:"from"`
	To    string `json:"to"`
	Topic string `json:"topic"`
}

func handleAEOPDDocs(w http.ResponseWriter, r *http.Request) {
	token, ok := aeopd.validToken(r.URL.Query().Get("token"))
	if !ok {
		aeopdWriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "session expired"})
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	db, err := aeopd.database()
	if err != nil {
		aeopdWriteJSON(w, http.StatusBadGateway, map[string]string{"error": "database unreachable"})
		return
	}
	ctx := r.Context()
	var total int
	if err := db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM shooting_range_documents WHERE status != 'cancelled'").Scan(&total); err != nil {
		aeopdWriteJSON(w, http.StatusBadGateway, map[string]string{"error": "query failed"})
		return
	}
	pages := (total + aeopdPageRows - 1) / aeopdPageRows
	if pages == 0 {
		pages = 1
	}
	if page > pages {
		page = pages
	}
	rows, err := db.QueryContext(ctx, `
		SELECT doc_type, DATE_FORMAT(doc_date, '%d-%m-%Y'), sender, IFNULL(receiver, ''), topic
		FROM shooting_range_documents
		WHERE status != 'cancelled'
		ORDER BY created_at DESC, id DESC
		LIMIT ? OFFSET ?`, aeopdPageRows, (page-1)*aeopdPageRows)
	if err != nil {
		aeopdWriteJSON(w, http.StatusBadGateway, map[string]string{"error": "query failed"})
		return
	}
	defer rows.Close()
	docs := make([]aeopdDocRow, 0, aeopdPageRows)
	for rows.Next() {
		var row aeopdDocRow
		if err := rows.Scan(&row.Type, &row.Date, &row.From, &row.To, &row.Topic); err != nil {
			aeopdWriteJSON(w, http.StatusBadGateway, map[string]string{"error": "query failed"})
			return
		}
		row.Label = aeopdDocTypeLabels[row.Type]
		if row.Label == "" {
			row.Label = row.Type
		}
		docs = append(docs, row)
	}
	aeopdWriteJSON(w, http.StatusOK, map[string]any{
		"page":  page,
		"pages": pages,
		"total": total,
		"name":  token.name,
		"rows":  docs,
	})
}

func registerAEOPDHandlers() {
	http.HandleFunc("/aeopd/login", handleAEOPDLogin)
	http.HandleFunc("/aeopd/docs", handleAEOPDDocs)
}
