package licensing

// Distributed with this client release. Key rotation requires an authenticated
// application/configuration update; keys in remote payloads are never trusted.
const ProviderURL = "https://aibid.adariel.com"

func ProviderKey() TrustKey {
	return TrustKey{Kid: "lic-prod-2026-a", PublicKey: "PiN24pE0T3oIK1z9TKbq9mzDlclHvwdqqQZuIKDV3AY", Environment: "production", Purpose: "license"}
}
func ProviderOptions() Options {
	return Options{ServerURL: ProviderURL, TrustedKeys: []TrustKey{ProviderKey()}, RefreshHours: 24}
}
