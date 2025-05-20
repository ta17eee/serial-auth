package handler

import (
	"database/sql"
	"encoding/json"
	"math/rand"
	"net/http"
	"time"
)

// CreateRequest はシリアルコード発行リクエストの構造体です。
// Codeが空の場合、ランダムなコードが生成されます。
type CreateRequest struct {
	Code string `json:"code,omitempty"`
}

// CreateResponse はシリアルコード発行レスポンスの構造体です。
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

// CreateHandler は /api/create へのリクエストを処理します。
func CreateHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
			return
		}

		var req CreateRequest
		// リクエストボディが空でない場合のみデコードを試みる
		if r.ContentLength > 0 {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}
		}

		var codeToInsert string
		if req.Code != "" {
			codeToInsert = req.Code
		} else {
			codeToInsert = generateRandomCode(codeLength)
		}

		// データベースにシリアルコードを挿入
		stmt, err := db.Prepare("INSERT INTO serial_codes (code) VALUES (?)")
		if err != nil {
			respondWithError(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer stmt.Close()

		_, err = stmt.Exec(codeToInsert)
		if err != nil {
			// UNIQUE制約違反の可能性などを考慮
			respondWithError(w, "Could not create serial code: "+err.Error(), http.StatusInternalServerError)
			return
		}

		resp := CreateResponse{
			Code:    codeToInsert,
			Message: "Serial code created successfully",
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
