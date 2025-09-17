// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package server

import (
	"net/http"
)

func PanicRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { // ハンドラ内の処理でパニックが起きたときに呼び出される関数
			if err := recover(); err != nil {
				http.Error(w, "Internal Error", http.StatusInternalServerError)
				// slog.Error("panic recovered", "error", err)
			}
		}()
		next.ServeHTTP(w, r) // ベースハンドラに制御を渡す
	})
}
