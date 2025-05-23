package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type Config struct {
	AdminToken string `json:"admin_token"`
}

var appConfig Config

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
			http.Error(w, "Unauthorized: Missing X-Admin-Token header", http.StatusUnauthorized)
			log.Printf("Auth failed: Missing X-Admin-Token from %s for %s %s", r.RemoteAddr, r.Method, r.RequestURI)
			return
		}

		if token != appConfig.AdminToken {
			http.Error(w, "Forbidden: Invalid X-Admin-Token", http.StatusForbidden)
			log.Printf("Auth failed: Invalid X-Admin-Token '%s' from %s for %s %s", token, r.RemoteAddr, r.Method, r.RequestURI)
			return
		}

		// 認証成功
		next.ServeHTTP(w, r)
	})
}
