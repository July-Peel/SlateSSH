package ws

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pkg/sftp"
	gossh "golang.org/x/crypto/ssh"

	"slatessh/backend/internal/auth"
	"slatessh/backend/internal/connections"
	"slatessh/backend/internal/models"
	sshsession "slatessh/backend/internal/sshsession"
	"slatessh/backend/internal/status"
)

type Hub struct {
	upgrader           websocket.Upgrader
	connectionsService *connections.Service
	authService        *auth.Service
	monitor            *status.Monitor
	mu                 sync.Mutex
	writeMu            sync.Mutex
	sessions           map[string]*ClientSession
}

type ClientSession struct {
	ID           string
	Socket       *websocket.Conn
	SSHClient    *gossh.Client
	Shell        io.ReadWriteCloser
	ShellInput   io.WriteCloser
	SFTPClient   *sftp.Client
	Connection   *models.DecryptedConnection
	ConnectionID int64
	TabID        string
	Closed       bool
	cancel       context.CancelFunc
}

type shellResizer interface {
	Resize(cols, rows int) error
}

type Message struct {
	Type      string         `json:"type"`
	SessionID string         `json:"sessionId,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
}

const maxInlineEditableBytes = 1024 * 1024

// NewHub 用于执行 NewHub 相关后端逻辑。
// 输入参数：connectionsService 表示connectionsService 参数；authService 表示authService 参数。
// 输出参数：返回 *Hub。
func NewHub(connectionsService *connections.Service, authService *auth.Service) *Hub {
	return &Hub{
		upgrader:           websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		connectionsService: connectionsService,
		authService:        authService,
		monitor:            status.NewMonitor(),
		sessions:           make(map[string]*ClientSession),
	}
}

// ServeHTTP 用于处理 WebSocket 或隧道 HTTP 请求。
// 输入参数：w 表示HTTP 响应写入器；r 表示HTTP 请求对象。
// 输出参数：无。
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.authService.CurrentUser(r) == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	socket, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer socket.Close()

	for {
		var message Message
		if err := socket.ReadJSON(&message); err != nil {
			h.closeSocketSessions(socket)
			return
		}
		if err := h.handleMessage(r.Context(), socket, message); err != nil {
			_ = h.write(socket, Message{Type: "error", SessionID: message.SessionID, Payload: map[string]any{"message": err.Error()}})
		}
	}
}

// ServeDownload 用于处理 SFTP 文件下载请求。
// 输入参数：w 表示HTTP 响应写入器；r 表示HTTP 请求对象。
// 输出参数：无。
func (h *Hub) ServeDownload(w http.ResponseWriter, r *http.Request) {
	if h.authService.CurrentUser(r) == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	sessionID := r.URL.Query().Get("sessionId")
	targetPath := r.URL.Query().Get("path")
	if sessionID == "" || targetPath == "" {
		http.Error(w, "missing sessionId or path", http.StatusBadRequest)
		return
	}

	session := h.getSession(sessionID)
	if session == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	file, err := session.SFTPClient.Open(targetPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err == nil && stat.Size() >= 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
	}

	filename := filepath.Base(targetPath)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(filename)))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	_, _ = io.Copy(w, file)
}

// ServeUpload 用于处理 SFTP 文件上传请求。
// 输入参数：w 表示HTTP 响应写入器；r 表示HTTP 请求对象。
// 输出参数：无。
func (h *Hub) ServeUpload(w http.ResponseWriter, r *http.Request) {
	if h.authService.CurrentUser(r) == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseMultipartForm(256 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sessionID := r.FormValue("sessionId")
	targetDir := r.FormValue("path")
	if sessionID == "" {
		http.Error(w, "missing sessionId", http.StatusBadRequest)
		return
	}
	if targetDir == "" {
		targetDir = "."
	}

	session := h.getSession(sessionID)
	if session == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	target := joinPath(targetDir, header.Filename)
	remote, err := session.SFTPClient.Create(target)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer remote.Close()

	written, err := io.Copy(remote, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"uploaded": true, "path": target, "bytes": written})
}

// handleMessage 用于分发前端 WebSocket 消息到对应处理方法。
// 输入参数：ctx 表示上下文对象；socket 表示WebSocket 连接；message 表示前端消息。
// 输出参数：返回 error；error 表示执行失败原因。
func (h *Hub) handleMessage(ctx context.Context, socket *websocket.Conn, message Message) error {
	switch message.Type {
	case "ssh:connect":
		return h.handleSSHConnect(ctx, socket, message)
	case "ssh:input":
		return h.handleSSHInput(message)
	case "ssh:resize":
		return h.handleSSHResize(message)
	case "ssh:disconnect":
		h.closeSession(message.SessionID, "manual disconnect", true)
		return nil
	case "sftp:readdir":
		return h.handleSFTPReaddir(message)
	case "sftp:readfile":
		return h.handleSFTPReadFile(message)
	case "sftp:writefile":
		return h.handleSFTPWriteFile(message)
	case "sftp:mkdir":
		return h.handleSFTPMkdir(message)
	case "sftp:rmdir":
		return h.handleSFTPRmdir(message)
	case "sftp:unlink":
		return h.handleSFTPUnlink(message)
	case "sftp:rename":
		return h.handleSFTPRename(message)
	case "status:refresh":
		return h.handleStatusRefresh(message)
	default:
		return fmt.Errorf("unsupported message type: %s", message.Type)
	}
}

// handleSSHConnect 用于建立 SSH 会话并创建前端标签会话。
// 输入参数：ctx 表示上下文对象；socket 表示WebSocket 连接；message 表示前端消息。
// 输出参数：返回 error；error 表示执行失败原因。
func (h *Hub) handleSSHConnect(ctx context.Context, socket *websocket.Conn, message Message) error {
	connectionID, _ := int64FromPayload(message.Payload, "connectionId")
	if connectionID <= 0 {
		return fmt.Errorf("missing connectionId")
	}

	connection, err := h.connectionsService.GetDecrypted(connectionID)
	if err != nil {
		return err
	}
	if connection == nil {
		return fmt.Errorf("connection not found")
	}

	client, err := sshsession.Dial(connection)
	if err != nil {
		return err
	}
	shell, input, err := sshsession.OpenShell(client, 120, 32)
	if err != nil {
		_ = client.Close()
		return err
	}
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		_ = shell.Close()
		_ = client.Close()
		return err
	}

	sessionID := uuid.NewString()
	sessionCtx, cancel := context.WithCancel(ctx)
	session := &ClientSession{
		ID:           sessionID,
		Socket:       socket,
		SSHClient:    client,
		Shell:        shell,
		ShellInput:   input,
		SFTPClient:   sftpClient,
		Connection:   connection,
		ConnectionID: connectionID,
		TabID:        sessionID,
		cancel:       cancel,
	}

	h.mu.Lock()
	h.sessions[sessionID] = session
	h.mu.Unlock()

	_ = h.connectionsService.TouchLastConnected(connectionID)
	_ = h.write(socket, Message{Type: "ssh:connected", SessionID: sessionID, Payload: map[string]any{"connectionId": connectionID, "sessionId": sessionID, "resolvedHostIp": connection.Host}})

	go h.streamShell(sessionCtx, session)
	go h.pollStatus(sessionCtx, session)
	return nil
}

// streamShell 用于持续读取 SSH Shell 输出并推送到前端。
// 输入参数：ctx 表示上下文对象；session 表示客户端会话。
// 输出参数：无。
func (h *Hub) streamShell(ctx context.Context, session *ClientSession) {
	buffer := make([]byte, 32*1024)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		count, err := session.Shell.Read(buffer)
		if count > 0 {
			encoded := base64.StdEncoding.EncodeToString(buffer[:count])
			_ = h.write(session.Socket, Message{Type: "ssh:output", SessionID: session.ID, Payload: map[string]any{"data": encoded, "encoding": "base64"}})
		}
		if err != nil {
			reason := "shell exited"
			if err != io.EOF {
				reason = err.Error()
			}
			h.closeSession(session.ID, reason, true)
			return
		}
	}
}

// pollStatus 用于定时采集服务器状态并推送到前端。
// 输入参数：ctx 表示上下文对象；session 表示客户端会话。
// 输出参数：无。
func (h *Hub) pollStatus(ctx context.Context, session *ClientSession) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		if err := h.pushStatus(session); err != nil {
			_ = h.write(session.Socket, Message{Type: "status_error", SessionID: session.ID, Payload: map[string]any{"message": err.Error()}})
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// pushStatus 用于采集一次服务器状态并写入 WebSocket。
// 输入参数：session 表示客户端会话。
// 输出参数：返回 error；error 表示执行失败原因。
func (h *Hub) pushStatus(session *ClientSession) error {
	current, err := h.monitor.Fetch(session.ID, session.SSHClient)
	if err != nil {
		return err
	}
	return h.write(session.Socket, Message{Type: "status_update", SessionID: session.ID, Payload: toMap(current)})
}

// handleSSHInput 用于将前端输入写入 SSH Shell。
// 输入参数：message 表示前端消息。
// 输出参数：返回 error；error 表示执行失败原因。
func (h *Hub) handleSSHInput(message Message) error {
	session := h.getSession(message.SessionID)
	if session == nil {
		return fmt.Errorf("session not found")
	}
	data, _ := stringFromPayload(message.Payload, "data")
	if data == "" {
		return nil
	}
	_, err := io.WriteString(session.ShellInput, data)
	return err
}

// handleSSHResize 用于处理前端终端尺寸变化并同步远端 PTY。
// 输入参数：message 表示前端消息。
// 输出参数：返回 error；error 表示执行失败原因。
func (h *Hub) handleSSHResize(message Message) error {
	session := h.getSession(message.SessionID)
	if session == nil {
		return fmt.Errorf("session not found")
	}
	cols, ok := intFromPayload(message.Payload, "cols")
	if !ok || cols <= 0 {
		return nil
	}
	rows, ok := intFromPayload(message.Payload, "rows")
	if !ok || rows <= 0 {
		return nil
	}
	resizer, ok := session.Shell.(shellResizer)
	if !ok {
		return nil
	}
	return resizer.Resize(cols, rows)
}

// handleStatusRefresh 用于处理手动刷新服务器状态请求。
// 输入参数：message 表示前端消息。
// 输出参数：返回 error；error 表示执行失败原因。
func (h *Hub) handleStatusRefresh(message Message) error {
	session := h.getSession(message.SessionID)
	if session == nil {
		return fmt.Errorf("session not found")
	}
	return h.pushStatus(session)
}

// handleSFTPReaddir 用于读取远端目录并返回文件列表。
// 输入参数：message 表示前端消息。
// 输出参数：返回 error；error 表示执行失败原因。
func (h *Hub) handleSFTPReaddir(message Message) error {
	session := h.getSession(message.SessionID)
	if session == nil {
		return fmt.Errorf("session not found")
	}
	target, _ := stringFromPayload(message.Payload, "path")
	if target == "" {
		target = "."
	}
	displayPath := target
	if realPath, err := session.SFTPClient.RealPath(target); err == nil && realPath != "" {
		displayPath = realPath
	}
	entries, err := session.SFTPClient.ReadDir(target)
	if err != nil {
		return err
	}
	items := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		items = append(items, map[string]any{
			"filename": entry.Name(),
			"isDir":    entry.IsDir(),
			"size":     entry.Size(),
			"mode":     entry.Mode().String(),
			"mtime":    entry.ModTime().Unix(),
		})
	}
	return h.write(session.Socket, Message{Type: "sftp:readdir:result", SessionID: session.ID, Payload: map[string]any{"path": displayPath, "entries": items}})
}

// handleSFTPReadFile 用于读取远端文本文件并返回可编辑内容。
// 输入参数：message 表示前端消息。
// 输出参数：返回 error；error 表示执行失败原因。
func (h *Hub) handleSFTPReadFile(message Message) error {
	session := h.getSession(message.SessionID)
	if session == nil {
		return fmt.Errorf("session not found")
	}
	target, _ := stringFromPayload(message.Payload, "path")
	stat, err := session.SFTPClient.Stat(target)
	if err != nil {
		return err
	}
	if stat.IsDir() {
		return fmt.Errorf("cannot open directory as file")
	}
	if stat.Size() > maxInlineEditableBytes {
		return h.write(session.Socket, Message{Type: "sftp:readfile:blocked", SessionID: session.ID, Payload: map[string]any{"path": target, "reason": "file too large for inline editing", "size": stat.Size()}})
	}

	file, err := session.SFTPClient.Open(target)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	if !isLikelyText(data) {
		return h.write(session.Socket, Message{Type: "sftp:readfile:blocked", SessionID: session.ID, Payload: map[string]any{"path": target, "reason": "binary file cannot be opened inline", "size": len(data)}})
	}
	return h.write(session.Socket, Message{Type: "sftp:readfile:result", SessionID: session.ID, Payload: map[string]any{"path": target, "content": string(data), "size": len(data)}})
}

// handleSFTPWriteFile 用于将编辑后的内容写入远端文件。
// 输入参数：message 表示前端消息。
// 输出参数：返回 error；error 表示执行失败原因。
func (h *Hub) handleSFTPWriteFile(message Message) error {
	session := h.getSession(message.SessionID)
	if session == nil {
		return fmt.Errorf("session not found")
	}
	target, _ := stringFromPayload(message.Payload, "path")
	content, _ := stringFromPayload(message.Payload, "content")
	file, err := session.SFTPClient.Create(target)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write([]byte(content)); err != nil {
		return err
	}
	return h.write(session.Socket, Message{Type: "sftp:writefile:result", SessionID: session.ID, Payload: map[string]any{"path": target, "saved": true, "bytes": len(content)}})
}

// handleSFTPMkdir 用于创建远端目录。
// 输入参数：message 表示前端消息。
// 输出参数：返回 error；error 表示执行失败原因。
func (h *Hub) handleSFTPMkdir(message Message) error {
	session := h.getSession(message.SessionID)
	if session == nil {
		return fmt.Errorf("session not found")
	}
	target, _ := stringFromPayload(message.Payload, "path")
	if err := session.SFTPClient.MkdirAll(target); err != nil {
		return err
	}
	return h.write(session.Socket, Message{Type: "sftp:mkdir:result", SessionID: session.ID, Payload: map[string]any{"path": target, "created": true}})
}

// handleSFTPRmdir 用于删除远端目录。
// 输入参数：message 表示前端消息。
// 输出参数：返回 error；error 表示执行失败原因。
func (h *Hub) handleSFTPRmdir(message Message) error {
	session := h.getSession(message.SessionID)
	if session == nil {
		return fmt.Errorf("session not found")
	}
	target, _ := stringFromPayload(message.Payload, "path")
	if err := session.SFTPClient.RemoveDirectory(target); err != nil {
		return err
	}
	return h.write(session.Socket, Message{Type: "sftp:rmdir:result", SessionID: session.ID, Payload: map[string]any{"path": target, "removed": true}})
}

// handleSFTPUnlink 用于删除远端文件。
// 输入参数：message 表示前端消息。
// 输出参数：返回 error；error 表示执行失败原因。
func (h *Hub) handleSFTPUnlink(message Message) error {
	session := h.getSession(message.SessionID)
	if session == nil {
		return fmt.Errorf("session not found")
	}
	target, _ := stringFromPayload(message.Payload, "path")
	if err := session.SFTPClient.Remove(target); err != nil {
		return err
	}
	return h.write(session.Socket, Message{Type: "sftp:unlink:result", SessionID: session.ID, Payload: map[string]any{"path": target, "removed": true}})
}

// handleSFTPRename 用于重命名或移动远端文件路径。
// 输入参数：message 表示前端消息。
// 输出参数：返回 error；error 表示执行失败原因。
func (h *Hub) handleSFTPRename(message Message) error {
	session := h.getSession(message.SessionID)
	if session == nil {
		return fmt.Errorf("session not found")
	}
	oldPath, _ := stringFromPayload(message.Payload, "oldPath")
	newPath, _ := stringFromPayload(message.Payload, "newPath")
	if err := session.SFTPClient.Rename(oldPath, newPath); err != nil {
		return err
	}
	return h.write(session.Socket, Message{Type: "sftp:rename:result", SessionID: session.ID, Payload: map[string]any{"oldPath": oldPath, "newPath": newPath, "renamed": true}})
}

// getSession 用于根据会话 ID 获取当前客户端会话。
// 输入参数：id 表示标识符。
// 输出参数：返回 *ClientSession。
func (h *Hub) getSession(id string) *ClientSession {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.sessions[id]
}

// closeSocketSessions 用于关闭指定 WebSocket 关联的全部 SSH 会话。
// 输入参数：socket 表示WebSocket 连接。
// 输出参数：无。
func (h *Hub) closeSocketSessions(socket *websocket.Conn) {
	h.mu.Lock()
	ids := make([]string, 0)
	for id, session := range h.sessions {
		if session.Socket == socket {
			ids = append(ids, id)
		}
	}
	h.mu.Unlock()
	for _, id := range ids {
		h.closeSession(id, "socket closed", false)
	}
}

// closeSession 用于释放 SSH、SFTP 和会话上下文资源。
// 输入参数：id 表示标识符；reason 表示reason 参数；notify 表示notify 参数。
// 输出参数：无。
func (h *Hub) closeSession(id, reason string, notify bool) {
	h.mu.Lock()
	session := h.sessions[id]
	if session == nil || session.Closed {
		h.mu.Unlock()
		return
	}
	session.Closed = true
	delete(h.sessions, id)
	h.mu.Unlock()

	if session.cancel != nil {
		session.cancel()
	}
	if session.SFTPClient != nil {
		_ = session.SFTPClient.Close()
	}
	if session.Shell != nil {
		_ = session.Shell.Close()
	}
	if session.SSHClient != nil {
		_ = session.SSHClient.Close()
	}

	if notify {
		_ = h.write(session.Socket, Message{Type: "ssh:disconnected", SessionID: id, Payload: map[string]any{"reason": reason, "autoCloseTab": true}})
	}
}

// write 用于向 WebSocket 写入 JSON 消息。
// 输入参数：socket 表示WebSocket 连接；message 表示前端消息。
// 输出参数：返回 error；error 表示执行失败原因。
func (h *Hub) write(socket *websocket.Conn, message Message) error {
	h.writeMu.Lock()
	defer h.writeMu.Unlock()
	socket.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return socket.WriteJSON(message)
}

// stringFromPayload 用于从消息载荷中读取字符串字段。
// 输入参数：payload 表示消息载荷；key 表示键名。
// 输出参数：返回 string, bool。
func stringFromPayload(payload map[string]any, key string) (string, bool) {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return "", false
	}
	value, ok := raw.(string)
	return value, ok
}

// intFromPayload 用于从消息载荷中读取整数字段。
// 输入参数：payload 表示消息载荷；key 表示键名。
// 输出参数：返回 int, bool。
func intFromPayload(payload map[string]any, key string) (int, bool) {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return 0, false
	}
	switch value := raw.(type) {
	case float64:
		return int(value), true
	case int:
		return value, true
	case int64:
		return int(value), true
	case string:
		parsed, err := strconv.Atoi(value)
		return parsed, err == nil
	default:
		return 0, false
	}
}

// int64FromPayload 用于从消息载荷中读取 64 位整数字段。
// 输入参数：payload 表示消息载荷；key 表示键名。
// 输出参数：返回 int64, bool。
func int64FromPayload(payload map[string]any, key string) (int64, bool) {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return 0, false
	}
	switch value := raw.(type) {
	case float64:
		return int64(value), true
	case int64:
		return value, true
	case int:
		return int64(value), true
	case string:
		parsed, err := strconv.ParseInt(value, 10, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

// toMap 用于将服务器状态结构转换为通用 JSON 映射。
// 输入参数：status 表示HTTP 状态码。
// 输出参数：返回 map[string]any。
func toMap(status models.ServerStatus) map[string]any {
	raw, _ := json.Marshal(status)
	result := map[string]any{}
	_ = json.Unmarshal(raw, &result)
	return result
}

// isLikelyText 用于判断文件内容是否适合按文本内联编辑。
// 输入参数：data 表示响应数据。
// 输出参数：返回 bool。
func isLikelyText(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	var binaryCount int
	sample := data
	if len(sample) > 8192 {
		sample = sample[:8192]
	}
	for _, b := range sample {
		if b == 0 {
			return false
		}
		if b < 7 || (b > 14 && b < 32) {
			binaryCount++
		}
	}
	return float64(binaryCount)/float64(len(sample)) < 0.05
}

// joinPath 用于拼接远端路径片段。
// 输入参数：base 表示基础路径；name 表示字段名或参数名。
// 输出参数：返回 string。
func joinPath(base, name string) string {
	if base == "" || base == "." {
		return "./" + strings.TrimPrefix(name, "./")
	}
	base = strings.TrimSuffix(base, "/")
	return base + "/" + strings.TrimPrefix(name, "./")
}
