package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type SerialInfo struct {
	ID        int       `json:"id"`
	Code      string    `json:"code"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	MaxUses   int       `json:"max_uses"`
	UsesCount int       `json:"uses_count"`
	IsValid   bool      `json:"is_valid"`
}

type ListSerialsResponse struct {
	Serials []SerialInfo `json:"serials"`
	Count   int          `json:"count"`
	Error   string       `json:"error,omitempty"`
}

func ListSerialsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respondListError(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
			return
		}

		showAll := r.URL.Query().Get("all")

		var rows *sql.Rows
		var err error

		query := "SELECT id, code, created_at, expires_at, max_uses, uses_count FROM serial_codes"
		args := []interface{}{}

		if showAll == "" || showAll == "false" {
			query += " WHERE expires_at > ? AND uses_count < max_uses"
			args = append(args, time.Now())
		}
		query += " ORDER BY id DESC"

		rows, err = db.Query(query, args...)
		if err != nil {
			respondListError(w, "Database query error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var serials []SerialInfo
		for rows.Next() {
			var s SerialInfo
			var createdAt, expiresAt string // SQLiteはDATETIMEを文字列として返すことがあるため、一度文字列で受ける

			if err := rows.Scan(&s.ID, &s.Code, &createdAt, &expiresAt, &s.MaxUses, &s.UsesCount); err != nil {
				respondListError(w, "Database scan error: "+err.Error(), http.StatusInternalServerError)
				return
			}

			// SQLiteからの日付文字列をtime.Timeにパース (UTCを期待)
			s.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAt) // SQLiteのデフォルトDATETIME形式
			if err != nil {
				s.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
				if err != nil {
					respondListError(w, "Error parsing created_at: "+err.Error(), http.StatusInternalServerError)
					return
				}
			}
			s.ExpiresAt, err = time.Parse("2006-01-02 15:04:05", expiresAt)
			if err != nil {
				s.ExpiresAt, err = time.Parse(time.RFC3339Nano, expiresAt)
				if err != nil {
					respondListError(w, "Error parsing expires_at: "+err.Error(), http.StatusInternalServerError)
					return
				}
			}

			s.IsValid = time.Now().Before(s.ExpiresAt) && s.UsesCount < s.MaxUses
			serials = append(serials, s)
		}

		if err = rows.Err(); err != nil {
			respondListError(w, "Database rows error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		resp := ListSerialsResponse{
			Serials: serials,
			Count:   len(serials),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func respondListError(w http.ResponseWriter, message string, statusCode int) {
	resp := ListSerialsResponse{
		Error: message,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}
