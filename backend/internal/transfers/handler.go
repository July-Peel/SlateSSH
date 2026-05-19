package transfers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	db *sql.DB
}

// NewHandler 用于创建对应模块的 HTTP 处理器。
// 输入参数：db 表示数据库连接。
// 输出参数：返回 *Handler。
func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

// Initiate 用于创建传输任务记录。
// 输入参数：w 表示HTTP 响应写入器；r 表示HTTP 请求对象。
// 输出参数：无。
func (h *Handler) Initiate(w http.ResponseWriter, r *http.Request) {
	payload := map[string]any{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"message": err.Error()})
		return
	}
	id := uuid.NewString()
	now := time.Now().Unix()
	raw, _ := json.Marshal(payload)
	if _, err := h.db.Exec(`INSERT INTO transfer_tasks (id, payload, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`, id, string(raw), "queued", now, now); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"message": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"id": id, "status": "queued", "payload": payload})
}

// List 用于返回列表数据。
// 输入参数：w 表示HTTP 响应写入器；r 表示HTTP 请求对象。
// 输出参数：无。
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`SELECT id, payload, status, created_at, updated_at FROM transfer_tasks ORDER BY created_at DESC`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"message": err.Error()})
		return
	}
	defer rows.Close()
	tasks := []map[string]any{}
	for rows.Next() {
		var id, payload, status string
		var createdAt, updatedAt int64
		if err := rows.Scan(&id, &payload, &status, &createdAt, &updatedAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"message": err.Error()})
			return
		}
		decoded := map[string]any{}
		_ = json.Unmarshal([]byte(payload), &decoded)
		tasks = append(tasks, map[string]any{"id": id, "payload": decoded, "status": status, "created_at": createdAt, "updated_at": updatedAt})
	}
	writeJSON(w, http.StatusOK, tasks)
}

// Get 用于根据请求参数返回单条数据。
// 输入参数：w 表示HTTP 响应写入器；r 表示HTTP 请求对象。
// 输出参数：无。
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "taskId")
	var payload, status string
	var createdAt, updatedAt int64
	err := h.db.QueryRow(`SELECT payload, status, created_at, updated_at FROM transfer_tasks WHERE id = ?`, id).Scan(&payload, &status, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, map[string]any{"message": "任务不存在。"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"message": err.Error()})
		return
	}
	decoded := map[string]any{}
	_ = json.Unmarshal([]byte(payload), &decoded)
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "payload": decoded, "status": status, "created_at": createdAt, "updated_at": updatedAt})
}

// Cancel 用于取消指定传输任务。
// 输入参数：w 表示HTTP 响应写入器；r 表示HTTP 请求对象。
// 输出参数：无。
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "taskId")
	result, err := h.db.Exec(`UPDATE transfer_tasks SET status = ?, updated_at = ? WHERE id = ?`, "cancelled", time.Now().Unix(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"message": err.Error()})
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"message": err.Error()})
		return
	}
	if affected == 0 {
		writeJSON(w, http.StatusNotFound, map[string]any{"message": "任务不存在。"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "取消成功。"})
}

// writeJSON 用于写入 JSON HTTP 响应。
// 输入参数：w 表示HTTP 响应写入器；status 表示HTTP 状态码；data 表示响应数据。
// 输出参数：无。
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
