// Package handler содержит HTTP-хендлеры и вспомогательные функции
// для формирования JSON-ответов.
package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// writeJSON кодирует значение в JSON и пишет ответ с Content-Type.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		if err := json.NewEncoder(w).Encode(v); err != nil {
			slog.Error("encode response failed", "error", err)
		}
	}
}

// writeError пишет текстовый ответ с указанным HTTP-статусом.
func writeError(w http.ResponseWriter, status int, msg string) {
	http.Error(w, msg, status)
}
