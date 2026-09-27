package launcher

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBrowserTargetAllowsLocalHTTPAndConfiguredHTTPS(t *testing.T) {
	for _, origin := range []string{"http://127.0.0.1:18090", "http://[::1]:18090", "http://localhost:19090", "https://biblioteca.example"} {
		contents, _ := json.Marshal(map[string]string{"public_url": origin})
		actual, err := ReadTarget(strings.NewReader("\xef\xbb\xbf" + string(contents)))
		if err != nil || actual != origin {
			t.Fatalf("%s: %s, %v", origin, actual, err)
		}
	}
}

func TestBrowserTargetRejectsCommandsCredentialsAndNonlocalHTTP(t *testing.T) {
	for _, origin := range []string{"file:///C:/Windows/System32/cmd.exe", "javascript:alert(1)", "ms-settings:privacy", "http://example.com", "http://localhost.example.com", "https://user:password@example.com", "https://example.com/path", "https://example.com?token=secret", "https://example.com#fragment", "https:///missing-host", "http://127.0.0.1:18090\r\nignored"} {
		contents, _ := json.Marshal(map[string]string{"public_url": origin})
		if _, err := ReadTarget(strings.NewReader(string(contents))); err == nil {
			t.Fatalf("accepted %s", origin)
		}
	}
	for _, contents := range []string{`{"public_url":"http://127.0.0.1:18090","private_key":"unexpected"}`, `{"public_url":"http://127.0.0.1:18090"}{}`, strings.Repeat(" ", 4097)} {
		if _, err := ReadTarget(strings.NewReader(contents)); err == nil {
			t.Fatal("accepted invalid/oversized metadata")
		}
	}
}
