package server_test

import (
	"net/http"
	"os"
	"strings"
	"testing"
)

// TestAlertDismissJSExists checks that design/base/18-alert-dismiss.js exists.
// Acceptance item 4.
func TestAlertDismissJSExists(t *testing.T) {
	data, err := os.ReadFile("../../design/base/18-alert-dismiss.js")
	if err != nil {
		t.Fatalf("18-alert-dismiss.js does not exist; %v", err)
	}
	body := string(data)

	// The script must listen for clicks on [data-dismiss] and remove the alert.
	if !strings.Contains(body, "[data-dismiss]") {
		t.Error("alert dismiss script must target [data-dismiss] elements")
	}
	if !strings.Contains(body, ".sw-alert") {
		t.Error("alert dismiss script must reference .sw-alert")
	}
	if !strings.Contains(body, "remove()") {
		t.Error("alert dismiss script must call remove() to delete the alert from DOM")
	}
}

// TestAlertDismissJSInBundledSamewayJS checks that 18-alert-dismiss.js is
// bundled into /design/sameway.js. Acceptance item 4.
func TestAlertDismissJSInBundledSamewayJS(t *testing.T) {
	a, h := newApp(t)

	jsResp := get(t, h, "/design/sameway.js")
	wantStatus(t, jsResp, http.StatusOK)
	jsBody := jsResp.Body.String()

	if !strings.Contains(jsBody, "18-alert-dismiss.js") {
		t.Fatalf("bundled sameway.js must include 18-alert-dismiss.js; body starts:\n%s", truncate(jsBody))
	}
	if !strings.Contains(jsBody, "[data-dismiss]") {
		t.Fatalf("bundled sameway.js must contain the [data-dismiss] logic from 18-alert-dismiss.js")
	}

	_ = a // app is opened but we only need the handler for the request
}
