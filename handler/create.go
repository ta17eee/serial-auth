package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type CreateRequest struct {
	Code    string `json:"code,omitempty"`
	Expiry  string `json:"expiry,omitempty"`
	MaxUses int    `json:"max_uses,omitempty"`
}

type CreateResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const codeLength = 12

func generateRandomCode(length int) string {
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func parseExpiry(expiryStr string) (time.Duration, error) {
	if expiryStr == "" {
		return 0, fmt.Errorf("expiry string is empty")
	}

	unit := expiryStr[len(expiryStr)-1:]
	valueStr := expiryStr[:len(expiryStr)-1]

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, fmt.Errorf("invalid expiry value: %s", valueStr)
	}

	switch strings.ToLower(unit) {
	case "d":
		return time.Duration(value) * 24 * time.Hour, nil
	case "h":
		return time.Duration(value) * time.Hour, nil
	default:
		return 0, fmt.Errorf("invalid expiry unit: %s. Use 'd' for days or 'h' for hours", unit)
	}
}

// CreateHandler は /api/create へのリクエストを処理します。
func CreateHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 認証状態をチェック - 認証失敗時は処理をスキップ
		if authStatus, ok := r.Context().Value(AuthStatusKey).(string); ok && strings.HasPrefix(authStatus, "reject") {
			// 認証失敗時は何もしない（AuthMiddlewareで既にレスポンス済み）
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
			return
		}

		var req CreateRequest
		// リクエストボディが空でない場合のみデコードを試みる
		if r.ContentLength > 0 {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				respondWithError(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
				return
			}
		}

		var codeToInsert string
		if req.Code != "" {
			codeToInsert = req.Code
		} else {
			codeToInsert = generateRandomCode(codeLength)
		}

		// 有効期限の設定 (デフォルトは7日)
		expiryDuration := 7 * 24 * time.Hour // 7 days
		if req.Expiry != "" {
			dur, err := parseExpiry(req.Expiry)
			if err != nil {
				respondWithError(w, "Invalid expiry format: "+err.Error(), http.StatusBadRequest)
				return
			}
			expiryDuration = dur
		}
		expiresAt := time.Now().Add(expiryDuration)

		// 最大使用回数の設定 (デフォルトは1回)
		maxUses := 1
		if req.MaxUses > 0 {
			maxUses = req.MaxUses
		}

		// データベースにシリアルコードを挿入
		stmt, err := db.Prepare("INSERT INTO serial_codes (code, expires_at, max_uses) VALUES (?, ?, ?)")
		if err != nil {
			respondWithError(w, "Database error (prepare): "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer stmt.Close()

		_, err = stmt.Exec(codeToInsert, expiresAt, maxUses)
		if err != nil {
			respondWithError(w, "Could not create serial code (exec): "+err.Error(), http.StatusInternalServerError)
			return
		}

		resp := CreateResponse{
			Code:    codeToInsert,
			Message: fmt.Sprintf("Serial code created successfully. Expires at: %s, Max uses: %d", expiresAt.Format(time.RFC3339), maxUses),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}
}

func respondWithError(w http.ResponseWriter, message string, statusCode int) {
	resp := CreateResponse{
		Error: message,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}
