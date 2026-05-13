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
		errCh <- copyBrowserToGuacd(ctx, browser, guacd)
	}()

	<-errCh
}

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

func copyBrowserToGuacd(ctx context.Context, browser *websocket.Conn, guacd net.Conn) error {
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
		if _, err := io.WriteString(guacd, string(data)); err != nil {
			return err
		}
	}
}

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

func formatInstruction(elements ...string) string {
	parts := make([]string, 0, len(elements))
	for _, element := range elements {
		parts = append(parts, strconv.Itoa(len(element))+"."+element)
	}
	return strings.Join(parts, ",") + ";"
}

func parseOptions(r *http.Request) tunnelOptions {
	query := r.URL.Query()
	return tunnelOptions{
		Width:    firstNonEmpty(query.Get("width"), "1280"),
		Height:   firstNonEmpty(query.Get("height"), "720"),
		DPI:      firstNonEmpty(query.Get("dpi"), "96"),
		Timezone: firstNonEmpty(query.Get("timezone"), "Asia/Shanghai"),
	}
}

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
	case "enable-wallpaper", "enable-theming", "enable-font-smoothing", "enable-full-window-drag":
		return "false"
	case "resize-method":
		return "display-update"
	case "width":
		return options.Width
	case "height":
		return options.Height
	case "dpi":
		return options.DPI
	case "color-depth":
		return "24"
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

func firstNonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
