package connections

import (
    "encoding/json"
    "errors"
    "net/http"
    "strconv"

    "github.com/go-chi/chi/v5"
)

type Handler struct {
    service *Service
}

func NewHandler(service *Service) *Handler {
    return &Handler{service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
    items, err := h.service.List()
    if err != nil {
        writeError(w, err)
        return
    }
    writeJSON(w, http.StatusOK, items)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
    if err != nil {
        writeError(w, errBadRequest("无效的连接 ID。"))
        return
    }
    item, err := h.service.Find(id)
    if err != nil {
        writeError(w, err)
        return
    }
    if item == nil {
        writeJSON(w, http.StatusNotFound, map[string]any{"message": "连接未找到。"})
        return
    }
    writeJSON(w, http.StatusOK, item)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    var input UpsertInput
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        writeError(w, err)
        return
    }
    item, err := h.service.Create(input)
    if err != nil {
        writeError(w, err)
        return
    }
    writeJSON(w, http.StatusCreated, map[string]any{"message": "连接创建成功。", "connection": item})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
    if err != nil {
        writeError(w, errBadRequest("无效的连接 ID。"))
        return
    }
    var input UpsertInput
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        writeError(w, err)
        return
    }
    item, err := h.service.Update(id, input)
    if err != nil {
        writeError(w, err)
        return
    }
    writeJSON(w, http.StatusOK, map[string]any{"message": "连接更新成功。", "connection": item})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
    if err != nil {
        writeError(w, errBadRequest("无效的连接 ID。"))
        return
    }
    if err := h.service.Delete(id); err != nil {
        writeError(w, err)
        return
    }
    writeJSON(w, http.StatusOK, map[string]any{"message": "连接删除成功。"})
}

func (h *Handler) TestSaved(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
    if err != nil {
        writeError(w, errBadRequest("无效的连接 ID。"))
        return
    }
    connection, err := h.service.GetDecrypted(id)
    if err != nil {
        writeError(w, err)
        return
    }
    if connection == nil {
        writeJSON(w, http.StatusNotFound, map[string]any{"message": "连接未找到。"})
        return
    }
    latency, err := h.service.Test(r.Context(), UpsertInput{Host: connection.Host, Port: connection.Port, Username: connection.Username, AuthMethod: connection.AuthMethod, Password: connection.Password, PrivateKey: connection.PrivateKey, Passphrase: connection.Passphrase})
    if err != nil {
        writeError(w, err)
        return
    }
    writeJSON(w, http.StatusOK, map[string]any{"success": true, "message": "连接测试成功。", "latency": latency})
}

func (h *Handler) TestUnsaved(w http.ResponseWriter, r *http.Request) {
    var input UpsertInput
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        writeError(w, err)
        return
    }
    latency, err := h.service.Test(r.Context(), input)
    if err != nil {
        writeError(w, err)
        return
    }
    writeJSON(w, http.StatusOK, map[string]any{"success": true, "message": "连接测试成功。", "latency": latency})
}

type appError struct {
    Message string
    Status  int
}

func (e appError) Error() string { return e.Message }

func errBadRequest(message string) error { return appError{Message: message, Status: http.StatusBadRequest} }

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
