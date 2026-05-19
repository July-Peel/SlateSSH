package rdp

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	"slatessh/backend/internal/auth"
	"slatessh/backend/internal/connections"
	"slatessh/backend/internal/models"
)

type Handler struct {
	upgrader           websocket.Upgrader
	connectionsService *connections.Service
	authService        *auth.Service
	guacdHost          string
	guacdPort          int
}

type tunnelOptions struct {
	Width    string
	Height   string
	DPI      string
	Timezone string
}

// NewHandler 用于创建对应模块的 HTTP 处理器。
// 输入参数：connectionsService 表示connectionsService 参数；authService 表示authService 参数；guacdHost 表示guacdHost 参数；guacdPort 表示guacdPort 参数。
// 输出参数：返回 *Handler。
func NewHandler(connectionsService *connections.Service, authService *auth.Service, guacdHost string, guacdPort int) *Handler {
	if strings.TrimSpace(guacdHost) == "" {
		guacdHost = "guacd"
	}
	if guacdPort <= 0 {
		guacdPort = 4822
	}
	return &Handler{
		upgrader:           websocket.Upgrader{Subprotocols: []string{"guacamole"}, CheckOrigin: func(r *http.Request) bool { return true }},
		connectionsService: connectionsService,
		authService:        authService,
		guacdHost:          guacdHost,
		guacdPort:          guacdPort,
	}
}

// ServeHTTP 用于处理 WebSocket 或隧道 HTTP 请求。
// 输入参数：w 表示HTTP 响应写入器；r 表示HTTP 请求对象。
// 输出参数：无。
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.authService.CurrentUser(r) == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	connectionID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || connectionID <= 0 {
		http.Error(w, "invalid connection id", http.StatusBadRequest)
		return
	}

	connection, err := h.connectionsService.GetDecrypted(connectionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if connection == nil {
		http.Error(w, "connection not found", http.StatusNotFound)
		return
	}
	if strings.ToUpper(connection.Type) != "RDP" {
		http.Error(w, "connection is not RDP", http.StatusBadRequest)
		return
	}

	browser, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer browser.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	guacd, err := net.DialTimeout("tcp", net.JoinHostPort(h.guacdHost, strconv.Itoa(h.guacdPort)), 10*time.Second)
	if err != nil {
		_ = browser.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseTryAgainLater, err.Error()))
		return
	}
	defer guacd.Close()

	reader := bufio.NewReader(guacd)
	options := parseOptions(r)
	if err := h.handshake(guacd, reader, connection, options); err != nil {
		_ = browser.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, err.Error()))
		return
	}

	_ = h.connectionsService.TouchLastConnected(connectionID)

	var writeMu sync.Mutex
	errCh := make(chan error, 2)

	go func() {
		errCh <- copyGuacdToBrowser(ctx, reader, browser, &writeMu)
	}()
	go func() {
		errCh <- copyBrowserToGuacd(ctx, browser, guacd, &writeMu)
	}()

	<-errCh
}

// handshake 用于与 guacd 完成 RDP 协议握手。
// 输入参数：conn 表示网络连接；reader 表示缓冲读取器；connection 表示连接配置；options 表示隧道显示选项。
// 输出参数：返回 error；error 表示执行失败原因。
func (h *Handler) handshake(conn net.Conn, reader *bufio.Reader, connection *models.DecryptedConnection, options tunnelOptions) error {
	if _, err := io.WriteString(conn, formatInstruction("select", "rdp")); err != nil {
		return err
	}

	args, raw, err := readInstruction(reader)
	if err != nil {
		return err
	}
	if len(args) == 0 || args[0] != "args" {
		return fmt.Errorf("unexpected guacd handshake response: %s", raw)
	}

	if _, err := io.WriteString(conn, formatInstruction("size", options.Width, options.Height, options.DPI)); err != nil {
		return err
	}
	if _, err := io.WriteString(conn, formatInstruction("audio")); err != nil {
		return err
	}
	if _, err := io.WriteString(conn, formatInstruction("video")); err != nil {
		return err
	}
	if _, err := io.WriteString(conn, formatInstruction("image", "image/png", "image/jpeg")); err != nil {
		return err
	}
	if _, err := io.WriteString(conn, formatInstruction("timezone", options.Timezone)); err != nil {
		return err
	}

	values := make([]string, 0, len(args)-1)
	for _, name := range args[1:] {
		values = append(values, parameterValue(name, connection, options))
	}
	if _, err := io.WriteString(conn, formatInstruction(append([]string{"connect"}, values...)...)); err != nil {
		return err
	}
	return nil
}

// copyGuacdToBrowser 用于将 guacd 指令转发给浏览器。
// 输入参数：ctx 表示上下文对象；reader 表示缓冲读取器；browser 表示浏览器 WebSocket 连接；writeMu 表示写锁。
// 输出参数：返回 error；error 表示执行失败原因。
func copyGuacdToBrowser(ctx context.Context, reader *bufio.Reader, browser *websocket.Conn, writeMu *sync.Mutex) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		_, raw, err := readInstruction(reader)
		if err != nil {
			return err
		}
		writeMu.Lock()
		err = browser.WriteMessage(websocket.TextMessage, []byte(raw))
		writeMu.Unlock()
		if err != nil {
			return err
		}
	}
}

// copyBrowserToGuacd 用于将浏览器 Guacamole 指令转发给 guacd。
// 输入参数：ctx 表示上下文对象；browser 表示浏览器 WebSocket 连接；guacd 表示guacd 网络连接；writeMu 表示写锁。
// 输出参数：返回 error；error 表示执行失败原因。
func copyBrowserToGuacd(ctx context.Context, browser *websocket.Conn, guacd net.Conn, writeMu *sync.Mutex) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		messageType, data, err := browser.ReadMessage()
		if err != nil {
			return err
		}
		if messageType != websocket.TextMessage {
			continue
		}
		message := string(data)
		if isTunnelPing(message) {
			writeMu.Lock()
			err = browser.WriteMessage(websocket.TextMessage, data)
			writeMu.Unlock()
			if err != nil {
				return err
			}
			continue
		}
		if strings.HasPrefix(message, "0.") {
			continue
		}
		if _, err := io.WriteString(guacd, message); err != nil {
			return err
		}
	}
}

// isTunnelPing 用于识别 Guacamole WebSocketTunnel 内部心跳指令。
// 输入参数：message 表示前端消息。
// 输出参数：返回 bool。
func isTunnelPing(message string) bool {
	elements, _, err := readInstruction(bufio.NewReader(strings.NewReader(message)))
	return err == nil && len(elements) >= 2 && elements[0] == "" && elements[1] == "ping"
}

// readInstruction 用于读取并解析一条 Guacamole 协议指令。
// 输入参数：reader 表示缓冲读取器。
// 输出参数：返回 []string, string, error；error 表示执行失败原因。
func readInstruction(reader *bufio.Reader) ([]string, string, error) {
	var elements []string
	var raw strings.Builder

	for {
		lengthText, err := reader.ReadString('.')
		if err != nil {
			return nil, raw.String(), err
		}
		raw.WriteString(lengthText)
		lengthText = strings.TrimSuffix(lengthText, ".")
		length, err := strconv.Atoi(lengthText)
		if err != nil {
			return nil, raw.String(), err
		}

		buffer := make([]byte, length+1)
		if _, err := io.ReadFull(reader, buffer); err != nil {
			return nil, raw.String(), err
		}
		raw.Write(buffer)
		elements = append(elements, string(buffer[:length]))

		terminator := buffer[length]
		if terminator == ';' {
			return elements, raw.String(), nil
		}
		if terminator != ',' {
			return nil, raw.String(), fmt.Errorf("invalid guacamole instruction terminator %q", terminator)
		}
	}
}

// formatInstruction 用于按 Guacamole 协议格式编码指令。
// 输入参数：elements 表示协议元素列表。
// 输出参数：返回 string。
func formatInstruction(elements ...string) string {
	parts := make([]string, 0, len(elements))
	for _, element := range elements {
		parts = append(parts, strconv.Itoa(len(element))+"."+element)
	}
	return strings.Join(parts, ",") + ";"
}

// parseOptions 用于从 RDP 隧道请求中解析显示参数。
// 输入参数：r 表示HTTP 请求对象。
// 输出参数：返回 tunnelOptions。
func parseOptions(r *http.Request) tunnelOptions {
	query := r.URL.Query()
	return tunnelOptions{
		Width:    firstNonEmpty(query.Get("width"), "1280"),
		Height:   firstNonEmpty(query.Get("height"), "720"),
		DPI:      firstNonEmpty(query.Get("dpi"), "96"),
		Timezone: firstNonEmpty(query.Get("timezone"), "Asia/Shanghai"),
	}
}

// parameterValue 用于根据 guacd 请求的参数名返回 RDP 连接参数值。
// 输入参数：name 表示字段名或参数名；connection 表示连接配置；options 表示隧道显示选项。
// 输出参数：返回 string。
func parameterValue(name string, connection *models.DecryptedConnection, options tunnelOptions) string {
	switch strings.ToLower(name) {
	case "hostname":
		return connection.Host
	case "port":
		return strconv.Itoa(connection.Port)
	case "username":
		return connection.Username
	case "password":
		return connection.Password
	case "domain":
		return ""
	case "security":
		return "any"
	case "ignore-cert":
		return "true"
	case "enable-wallpaper", "enable-theming", "enable-font-smoothing", "enable-full-window-drag", "enable-desktop-composition", "enable-menu-animations":
		return "true"
	case "resize-method":
		return "reconnect"
	case "width":
		return options.Width
	case "height":
		return options.Height
	case "dpi":
		return options.DPI
	case "color-depth":
		return "32"
	case "timezone":
		return options.Timezone
	case "disable-audio":
		return "true"
	case "enable-drive", "create-drive-path", "enable-printing", "enable-audio-input", "read-only":
		return "false"
	default:
		return ""
	}
}

// firstNonEmpty 用于返回第一个非空字符串或默认值。
// 输入参数：value 表示输入值；fallback 表示默认值。
// 输出参数：返回 string。
func firstNonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
