package settings

import (
    "encoding/json"
    "net/http"
)

type Handler struct {
    repo *Repository
}

func NewHandler(repo *Repository) *Handler {
    return &Handler{repo: repo}
}

func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
    settings, err := h.repo.All()
    if err != nil {
        writeJSON(w, http.StatusInternalServerError, map[string]any{"message": err.Error()})
        return
    }
    writeJSON(w, http.StatusOK, settings)
}

func (h *Handler) UpdateMany(w http.ResponseWriter, r *http.Request) {
    payload := map[string]string{}
    if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
        writeJSON(w, http.StatusBadRequest, map[string]any{"message": err.Error()})
        return
    }
    if err := h.repo.SetMany(payload); err != nil {
        writeJSON(w, http.StatusInternalServerError, map[string]any{"message": err.Error()})
        return
    }
    writeJSON(w, http.StatusOK, map[string]any{"message": "设置保存成功。"})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(data)
}
