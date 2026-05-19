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

// NewHandler 用于创建对应模块的 HTTP 处理器。
// 输入参数：service 表示service 参数。
// 输出参数：返回 *Handler。
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List 用于返回列表数据。
// 输入参数：w 表示HTTP 响应写入器；r 表示HTTP 请求对象。
// 输出参数：无。
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// Get 用于根据请求参数返回单条数据。
// 输入参数：w 表示HTTP 响应写入器；r 表示HTTP 请求对象。
// 输出参数：无。
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

// Create 用于创建新数据记录。
// 输入参数：w 表示HTTP 响应写入器；r 表示HTTP 请求对象。
// 输出参数：无。
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

// Update 用于更新已有数据记录。
// 输入参数：w 表示HTTP 响应写入器；r 表示HTTP 请求对象。
// 输出参数：无。
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

// Delete 用于删除指定数据记录。
// 输入参数：w 表示HTTP 响应写入器；r 表示HTTP 请求对象。
// 输出参数：无。
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

// TestSaved 用于测试已保存的连接配置。
// 输入参数：w 表示HTTP 响应写入器；r 表示HTTP 请求对象。
// 输出参数：无。
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
	latency, err := h.service.Test(r.Context(), UpsertInput{Type: connection.Type, Host: connection.Host, Port: connection.Port, Username: connection.Username, AuthMethod: connection.AuthMethod, Password: connection.Password, PrivateKey: connection.PrivateKey, Passphrase: connection.Passphrase})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "message": "连接测试成功。", "latency": latency})
}

// TestUnsaved 用于测试尚未保存的连接配置。
// 输入参数：w 表示HTTP 响应写入器；r 表示HTTP 请求对象。
// 输出参数：无。
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

// Error 用于执行 Error 相关后端逻辑。
// 输入参数：无。
// 输出参数：返回 string。
func (e appError) Error() string { return e.Message }

// errBadRequest 用于创建请求参数错误。
// 输入参数：message 表示前端消息。
// 输出参数：返回 error { return appError；error 表示执行失败原因。
func errBadRequest(message string) error {
	return appError{Message: message, Status: http.StatusBadRequest}
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
