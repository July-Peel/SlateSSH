package rdp

import (
	"net/http/httptest"
	"testing"

	"slatessh/backend/internal/models"
)

func TestParseOptionsBuildsIsolatedDrivePath(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/v1/rdp/12/tunnel?width=1440&height=900&dpi=120&timezone=Europe%2FBudapest", nil)
	options := parseOptions(r, "/var/lib/slatessh/rdp-drives", 7, 12)

	if options.Width != "1440" || options.Height != "900" || options.DPI != "120" || options.Timezone != "Europe/Budapest" {
		t.Fatalf("unexpected display options: %+v", options)
	}
	if options.DrivePath != "/var/lib/slatessh/rdp-drives/users/7/connections/12" {
		t.Fatalf("unexpected drive path: %q", options.DrivePath)
	}
}

func TestParameterValueEnablesRdpDriveTransfer(t *testing.T) {
	connection := &models.DecryptedConnection{Connection: models.Connection{Host: "desktop.example", Port: 3389, Username: "alice"}, Password: "secret"}
	options := tunnelOptions{Width: "1280", Height: "720", DPI: "96", Timezone: "UTC", DrivePath: "/var/lib/slatessh/rdp-drives/users/1/connections/2"}

	tests := map[string]string{
		"enable-drive":      "true",
		"create-drive-path": "true",
		"drive-path":        options.DrivePath,
		"drive-name":        "SlateSSH",
		"disable-download":  "false",
		"disable-upload":    "false",
		"hostname":          "desktop.example",
		"username":          "alice",
		"password":          "secret",
	}
	for name, want := range tests {
		if got := parameterValue(name, connection, options); got != want {
			t.Errorf("parameterValue(%q) = %q, want %q", name, got, want)
		}
	}
}
