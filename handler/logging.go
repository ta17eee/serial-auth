package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// LoggingMiddleware はリクエストの情報をログに出力するミドルウェアです。
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := &loggingResponseWriter{ResponseWriter: w, buf: &bytes.Buffer{}}

		next.ServeHTTP(lrw, r)

		duration := time.Since(start)
		logMessage := fmt.Sprintf("[%s] %s %s %d %s", r.Method, r.RequestURI, r.Proto, lrw.statusCode, duration)

		// コンテキストから認証ステータスを取得
		if authStatus, ok := r.Context().Value(AuthStatusKey).(string); ok {
			logMessage += fmt.Sprintf(" Auth: %s", authStatus)
		}

		// レスポンスボディの内容に基づいて追加情報をログに出力
		if strings.HasPrefix(r.RequestURI, "/api/verify") {
			var resp VerifyResponse
			if err := json.Unmarshal(lrw.buf.Bytes(), &resp); err == nil {
				logMessage += fmt.Sprintf(" VerifySuccess: %t", resp.Valid)
			}
		} else if strings.HasPrefix(r.RequestURI, "/api/create") && lrw.statusCode == http.StatusCreated {
			var resp CreateResponse
			if err := json.Unmarshal(lrw.buf.Bytes(), &resp); err == nil {
				created := resp.Error == ""
				logMessage += fmt.Sprintf(" CreatedSuccess: %t", created)
			}
		} else if strings.HasPrefix(r.RequestURI, "/api/serials") && lrw.statusCode == http.StatusOK {
			var resp ListSerialsResponse
			if err := json.Unmarshal(lrw.buf.Bytes(), &resp); err == nil {
				logMessage += fmt.Sprintf(" SerialsCount: %d", resp.Count)
			}
		}

		log.Println(logMessage)
	})
}

// loggingResponseWriter はステータスコードとレスポンスボディをキャプチャするためのResponseWriterラッパーです。
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	buf        *bytes.Buffer // レスポンスボディをキャプチャするバッファ
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// Write はレスポンスボディをバッファに書き込み、元のResponseWriterにも書き込みます。
func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	if lrw.statusCode == 0 {
		lrw.statusCode = http.StatusOK
	}
	// バッファにも書き込む
	lrw.buf.Write(b)
	return lrw.ResponseWriter.Write(b)
}
