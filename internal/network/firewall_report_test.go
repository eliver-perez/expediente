package network

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestWindowsFirewallEffectiveRuleDiagnostics(t *testing.T) {
	const binary = `C:\Program Files\AIBID-Test\gestor-documental.exe`
	valid := map[string]any{
		"Enabled": true, "Allow": true, "Inbound": true, "EdgeBlocked": true,
		"Profiles": "Domain, Private", "Protocol": "TCP", "Port": "18090", "Program": binary, "Service": "AIBIDTest", "Remote": "LocalSubnet",
	}
	cases := []struct {
		key   string
		value any
		state string
	}{
		{"", nil, "managed"}, {"Enabled", false, "mismatch"}, {"Allow", false, "mismatch"},
		{"Port", "9999", "mismatch"}, {"Remote", "Any", "mismatch"}, {"Profiles", "Any", "mismatch"},
		{"Service", "OtherService", "mismatch"}, {"Program", `C:\Other.exe`, "mismatch"}, {"Protocol", "UDP", "mismatch"},
		{"missing", nil, "missing"}, {"public", nil, "public_network"}, {"policy", nil, "policy_restricted"},
	}
	for _, c := range cases {
		t.Run(c.key+c.state, func(t *testing.T) {
			rule := map[string]any{}
			for key, value := range valid {
				rule[key] = value
			}
			report := map[string]any{"Profiles": []any{map[string]any{"Name": "Domain", "Enabled": true, "LocalRulesBlocked": c.key == "policy"}}, "Categories": []string{"DomainAuthenticated"}, "Rule": rule}
			switch c.key {
			case "missing":
				report["Rule"] = nil
			case "public":
				report["Categories"] = []string{"Public"}
			case "", "policy":
			default:
				rule[c.key] = c.value
			}
			data, _ := json.Marshal(report)
			result := describeWindowsFirewall(unknownFirewall(), string(data), "18090", binary, "AIBIDTest")
			if result.State != c.state {
				t.Fatal(result.State, "expected", c.state)
			}
			if strings.Contains(result.Message, binary) {
				t.Fatal("raw command data exposed")
			}
		})
	}
	if describeWindowsFirewall(unknownFirewall(), "invalid", "18090", binary, "AIBIDTest").State != "unknown" {
		t.Fatal("invalid probe result trusted")
	}
}
