package httpapi

import (
	"gestor-documental/internal/network"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLANPreservesOriginAndCSRFChecks(t *testing.T) {
	fixture := newHTTPFixture(t)
	configuration := fixture.service.Config
	configuration.NetworkMode = "lan"
	configuration.ListenAddress = "0.0.0.0:8090"
	handler := New(fixture.service, configuration, fixture.server.logger).Handler()
	for _, tc := range []struct {
		host, origin, fetch string
		code                int
	}{
		{"127.0.0.1:8090", "http://127.0.0.1:8090", "same-origin", 200},
		{"127.0.0.1:8090", "http://attacker.example", "cross-site", 403},
		{"127.0.0.1:8090", "http://127.0.0.1:8090", "cross-site", 403},
		{"127.0.0.1:8090", "", "", 403},
		{"attacker.example:8090", "http://attacker.example:8090", "same-origin", 400},
	} {
		request := httptest.NewRequest("POST", "http://"+tc.host+"/api/v1/auth/login", strings.NewReader(`{"username":"admin","password":"`+testPassword+`"}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", tc.origin)
		request.Header.Set("Sec-Fetch-Site", tc.fetch)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != tc.code {
			t.Fatal(tc, response.Code, response.Body.String())
		}
		if response.Code == 200 {
			cookie := response.Result().Cookies()[0]
			if !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode {
				t.Fatal("cookie protections lost")
			}
		}
	}
}

func TestExistingLANConnectionCannotKeepAccessAfterLocalSwitch(t *testing.T) {
	fixture := newHTTPFixture(t)
	local := fixture.service.Config
	priorLAN := local
	priorLAN.NetworkMode = "lan"
	priorLAN.ListenAddress = "0.0.0.0:8090"
	handler := New(fixture.service, priorLAN, fixture.server.logger).WithNetwork(network.New("", local)).Handler()
	request := httptest.NewRequest("GET", "http://127.0.0.1:8090/health/live", nil)
	request.RemoteAddr = "192.0.2.4:10000"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 403 || !strings.Contains(response.Body.String(), "NETWORK_LOCAL_ONLY") {
		t.Fatal("old LAN connection survived", response.Code)
	}
}
