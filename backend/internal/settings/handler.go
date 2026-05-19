package settings

import (
	"encoding/json"
	"net/http"
)

type Handler struct {
	repo *Repository
}

// NewHandler 用于创建对应模块的 HTTP 处理器。
// 输入参数：repo 表示repo 参数。
// 输出参数：返回 *Handler。
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// GetAll 用于返回全部设置项。
// 输入参数：w 表示HTTP 响应写入器；r 表示HTTP 请求对象。
// 输出参数：无。
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	settings, err := h.repo.All()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

// UpdateMany 用于批量更新设置项。
// 输入参数：w 表示HTTP 响应写入器；r 表示HTTP 请求对象。
// 输出参数：无。
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

// writeJSON 用于写入 JSON HTTP 响应。
// 输入参数：w 表示HTTP 响应写入器；status 表示HTTP 状态码；data 表示响应数据。
// 输出参数：无。
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
