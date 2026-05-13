package auth

import "net/http"

func RequireAuth(service *Service, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if service.CurrentUser(r) == nil {
            writeJSON(w, http.StatusUnauthorized, map[string]any{"message": "未授权。"})
            return
        }
        next.ServeHTTP(w, r)
    })
}
