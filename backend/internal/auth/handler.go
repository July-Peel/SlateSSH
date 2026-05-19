package auth

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Handler struct {
	service *Service
}

// NewHandler 用于创建对应模块的 HTTP 处理器。
// 输入参数：service 表示service 参数。
// 输出参数：返回 *Handler。
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// NeedsSetup 用于返回系统是否需要初始化管理员账号。
// 输入参数：w 表示HTTP 响应写入器；r 表示HTTP 请求对象。
// 输出参数：无。
func (h *Handler) NeedsSetup(w http.ResponseWriter, r *http.Request) {
	needsSetup, err := h.service.NeedsSetup()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"needsSetup": needsSetup})
}

// SetupAdmin 用于处理初始化管理员账号请求。
// 输入参数：w 表示HTTP 响应写入器；r 表示HTTP 请求对象。
// 输出参数：无。
func (h *Handler) SetupAdmin(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Username        string `json:"username"`
		Password        string `json:"password"`
		ConfirmPassword string `json:"confirmPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, err)
		return
	}
	if err := h.service.SetupAdmin(payload.Username, payload.Password, payload.ConfirmPassword); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": "初始化成功。"})
}

// Login 用于处理用户登录请求并写入会话。
// 输入参数：w 表示HTTP 响应写入器；r 表示HTTP 请求对象。
// 输出参数：无。
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Username   string `json:"username"`
		Password   string `json:"password"`
		RememberMe bool   `json:"rememberMe"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, err)
		return
	}
	user, err := h.service.Login(w, r, payload.Username, payload.Password, payload.RememberMe)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "登录成功。", "user": user})
}

// Logout 用于处理用户退出登录请求并清理会话。
// 输入参数：w 表示HTTP 响应写入器；r 表示HTTP 请求对象。
// 输出参数：无。
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Logout(r); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "已登出。"})
}

// Status 用于返回当前登录状态和用户信息。
// 输入参数：w 表示HTTP 响应写入器；r 表示HTTP 请求对象。
// 输出参数：无。
func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	user := h.service.CurrentUser(r)
	if user == nil {
		writeJSON(w, http.StatusOK, map[string]any{"isAuthenticated": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"isAuthenticated": true, "user": user})
}

// ChangePassword 用于处理当前用户修改密码请求。
// 输入参数：w 表示HTTP 响应写入器；r 表示HTTP 请求对象。
// 输出参数：无。
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, err)
		return
	}
	if err := h.service.ChangePassword(r, payload.CurrentPassword, payload.NewPassword); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "密码修改成功。"})
}

// writeJSON 用于写入 JSON HTTP 响应。
// 输入参数：w 表示HTTP 响应写入器；status 表示HTTP 状态码；data 表示响应数据。
// 输出参数：无。
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeError 用于写入错误 HTTP 响应。
// 输入参数：w 表示HTTP 响应写入器；err 表示err 参数。
// 输出参数：无。
func writeError(w http.ResponseWriter, err error) {
	var appErr appError
	if errors.As(err, &appErr) {
		writeJSON(w, appErr.Status, map[string]any{"message": appErr.Message})
		return
	}
	writeJSON(w, http.StatusInternalServerError, map[string]any{"message": err.Error()})
}
