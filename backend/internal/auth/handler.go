package auth

import (
    "encoding/json"
    "errors"
    "net/http"
)

type Handler struct {
    service *Service
}

func NewHandler(service *Service) *Handler {
    return &Handler{service: service}
}

func (h *Handler) NeedsSetup(w http.ResponseWriter, r *http.Request) {
    needsSetup, err := h.service.NeedsSetup()
    if err != nil {
        writeError(w, err)
        return
    }
    writeJSON(w, http.StatusOK, map[string]any{"needsSetup": needsSetup})
}

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

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
    if err := h.service.Logout(r); err != nil {
        writeError(w, err)
        return
    }
    writeJSON(w, http.StatusOK, map[string]any{"message": "已登出。"})
}

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
    user := h.service.CurrentUser(r)
    if user == nil {
        writeJSON(w, http.StatusOK, map[string]any{"isAuthenticated": false})
        return
    }
    writeJSON(w, http.StatusOK, map[string]any{"isAuthenticated": true, "user": user})
}

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

func writeJSON(w http.ResponseWriter, status int, data any) {
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, err error) {
    var appErr appError
    if errors.As(err, &appErr) {
        writeJSON(w, appErr.Status, map[string]any{"message": appErr.Message})
        return
    }
    writeJSON(w, http.StatusInternalServerError, map[string]any{"message": err.Error()})
}
