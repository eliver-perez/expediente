package licensing

// Distributed with this client release. Key rotation requires an authenticated
// application/configuration update; keys in remote payloads are never trusted.
const ProviderURL = "https://licensing.aidex.com.mx"

func ProviderKey() TrustKey {
	return TrustKey{Kid: "lic-prod-2026-a", PublicKey: "X8U4PxM_kr3U56k27D6lVkKIEbLUUTAkJs_-jbXdwJM", Environment: "production", Purpose: "license"}
}
func ProviderOptions() Options {
	return Options{ServerURL: ProviderURL, TrustedKeys: []TrustKey{ProviderKey()}, RefreshHours: 24}
}
