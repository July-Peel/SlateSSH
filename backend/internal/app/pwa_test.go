package app

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"slatessh/backend/internal/config"
)

func TestPublicPWARoutes(t *testing.T) {
	root := t.TempDir()
	for name, body := range map[string]string{"index.html": "<!doctype html><title>SlateSSH</title>", "sw.js": "self.addEventListener('fetch', () => {});"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	app := &App{cfg: config.Config{FrontendDir: root}}
	handler := app.routes()
	for _, check := range []struct{ path, contentType, body string }{
		{"/sw.js", "application/javascript", "self.addEventListener"},
		{"/", "text/html", "<!doctype html>"},
	} {
		t.Run(check.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest("GET", check.path, nil))
			if response.Code != 200 || !strings.HasPrefix(response.Header().Get("Content-Type"), check.contentType) || !strings.Contains(response.Body.String(), check.body) {
				t.Fatalf("unexpected PWA response: %d %s %s", response.Code, response.Header(), response.Body.String())
			}
			if response.Header().Get("Cache-Control") != "no-cache" {
				t.Fatal("PWA entry points must revalidate")
			}
		})
	}
}
