package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type VerifyRequest struct {
	Code string `json:"code"`
}

type VerifyResponse struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"` // エラーメッセージ用フィールド追加
}

func VerifyHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
			return
		}

		var req VerifyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondVerifyError(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		if req.Code == "" {
			respondVerifyError(w, "Code is required", http.StatusBadRequest)
			return
		}

		var expiresAt time.Time
		var maxUses, usesCount int
		err := db.QueryRow("SELECT expires_at, max_uses, uses_count FROM serial_codes WHERE code = ?", req.Code).Scan(&expiresAt, &maxUses, &usesCount)
		if err != nil {
			if err == sql.ErrNoRows {
				respondVerifyResult(w, false, "Code not found")
				return
			}
			respondVerifyError(w, "Database query error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// 有効期限の確認
		if time.Now().After(expiresAt) {
			respondVerifyResult(w, false, fmt.Sprintf("Code expired at %s", expiresAt.Format(time.RFC3339)))
			return
		}

		// 使用回数の確認
		if usesCount >= maxUses {
			respondVerifyResult(w, false, fmt.Sprintf("Code has reached its maximum usage limit of %d", maxUses))
			return
		}

		// 使用回数をインクリメント
		stmt, err := db.Prepare("UPDATE serial_codes SET uses_count = uses_count + 1 WHERE code = ?")
		if err != nil {
			respondVerifyError(w, "Database error (prepare update): "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer stmt.Close()

		_, err = stmt.Exec(req.Code)
		if err != nil {
			respondVerifyError(w, "Database error (exec update): "+err.Error(), http.StatusInternalServerError)
			return
		}

		respondVerifyResult(w, true, "Code is valid")
	}
}

func respondVerifyResult(w http.ResponseWriter, valid bool, message string) {
	res := VerifyResponse{
		Valid:   valid,
		Message: message,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func respondVerifyError(w http.ResponseWriter, errorMessage string, statusCode int) {
	res := VerifyResponse{
		Valid:   false,
		Message: "Verification failed",
		Error:   errorMessage,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(res)
}
