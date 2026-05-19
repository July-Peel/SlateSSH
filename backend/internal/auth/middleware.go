package auth

import "net/http"

// RequireAuth 用于校验请求是否已有登录用户。
// 输入参数：service 表示service 参数；next 表示下一个 HTTP 处理器。
// 输出参数：返回 http.Handler。
func RequireAuth(service *Service, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service.CurrentUser(r) == nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"message": "未授权。"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
