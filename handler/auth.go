package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

// contextKey はコンテキストのキーの型です。
type contextKey string

// AuthStatusKey は認証ステータスをコンテキストに保存するためのキーです。
const AuthStatusKey contextKey = "authStatus"

// Config は設定ファイルの内容を保持する構造体です。
type Config struct {
	AdminToken string `json:"admin_token"`
}

var appConfig Config

// LoadConfig は設定ファイルから設定を読み込みます。
func LoadConfig(filePath string) error {
	configFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer configFile.Close()

	byteValue, err := ioutil.ReadAll(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	err = json.Unmarshal(byteValue, &appConfig)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if appConfig.AdminToken == "" {
		return fmt.Errorf("admin_token is not set in config file")
	}
	log.Println("Configuration loaded successfully.")
	return nil
}

// AuthMiddleware はリクエストヘッダーの X-Admin-Token を検証するミドルウェアです。
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Admin-Token")

		if token == "" {
			// トークンが存在しない場合
			ctx := context.WithValue(r.Context(), AuthStatusKey, "reject (missing token)")
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized: Missing X-Admin-Token header\n"))

			// 認証失敗でもnext.ServeHTTP()を呼び出してLoggingMiddlewareに到達させる
			// ただし、ハンドラ側で認証状態をチェックして処理をスキップする
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		if token != appConfig.AdminToken {
			// トークンが無効な場合
			ctx := context.WithValue(r.Context(), AuthStatusKey, fmt.Sprintf("reject (invalid token: %s)", token))
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Forbidden: Invalid X-Admin-Token\n"))

			// 認証失敗でもnext.ServeHTTP()を呼び出してLoggingMiddlewareに到達させる
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// 認証成功時
		ctx := context.WithValue(r.Context(), AuthStatusKey, "accept")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
