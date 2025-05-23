package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"ta17eee-serial-auth/handler"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// ログの出力を log.txt に追記する形に変更
	logFile := "log.txt"
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal("Cannot open log file: ", err)
	}
	log.SetOutput(file)
	// log.SetFlags(log.LstdFlags | log.Lmicroseconds) // 必要に応じて日時とマイクロ秒のフラグを設定

	// 設定ファイルの読み込み
	if err := handler.LoadConfig("config.json"); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	db, err := sql.Open("sqlite3", "./serials.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// ハンドラをmuxに登録
	mux := http.NewServeMux()
	mux.HandleFunc("/api/verify", handler.VerifyHandler(db))

	// /api/create と /api/serials に認証ミドルウェアを適用
	mux.Handle("/api/create", handler.AuthMiddleware(http.HandlerFunc(handler.CreateHandler(db))))
	mux.Handle("/api/serials", handler.AuthMiddleware(http.HandlerFunc(handler.ListSerialsHandler(db))))

	// ロギングミドルウェアを適用 (認証ミドルウェアの後、メインのmuxの前)
	loggedMux := handler.LoggingMiddleware(mux)

	log.Println("Server listening on :8080")
	err = http.ListenAndServe(":8080", loggedMux)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
