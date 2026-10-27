package httpapi

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestLocalFolderAuthorizationRejectsBroadOrRemotePaths(t *testing.T) {
	for _, path := range []string{`\\server\share\docs`, `C:\`, `C:relative`, `C:\docs\bad` + "\n" + `name`, `D:\\`, `/tmp/docs`} {
		if localFolderAuthorizationPath(path) {
			t.Fatalf("accepted unsafe path %q", path)
		}
	}
	if !localFolderAuthorizationPath(`C:\Users\Ada\Documents`) {
		t.Fatal("rejected local directory")
	}
	for _, test := range []struct {
		remote, host string
		allowed      bool
	}{
		{"127.0.0.1:53210", "127.0.0.1:18090", true},
		{"[::1]:53210", "localhost:18090", true},
		{"192.0.2.15:53210", "127.0.0.1:18090", false},
		{"127.0.0.1:53210", "192.0.2.15:18090", false},
	} {
		r := httptest.NewRequest("POST", "http://"+test.host+"/api/v1/local-folder-authorizations", nil)
		r.RemoteAddr = test.remote
		if localOnly(r) != test.allowed {
			t.Fatalf("remote=%s host=%s", test.remote, test.host)
		}
	}
	proxy := httptest.NewRequest("POST", "http://127.0.0.1:18090/api/v1/local-folder-authorizations", nil)
	proxy.RemoteAddr = "127.0.0.1:53210"
	proxy.Header.Set("X-Forwarded-For", "192.0.2.15")
	if localOnly(proxy) {
		t.Fatal("reverse proxy request accepted as local")
	}
}

func TestFolderAuthorizationTicketExpiry(t *testing.T) {
	server := &Server{folderAuth: map[string]*folderAuthorization{"expired": {expires: time.Now().Add(-time.Second)}, "pending": {expires: time.Now().Add(time.Minute)}}}
	server.expireFolderTickets(time.Now())
	if server.folderAuth["expired"] != nil || server.folderAuth["pending"] == nil {
		t.Fatal("ticket expiry not enforced")
	}
}
