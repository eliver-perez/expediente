//go:build !windows && !darwin

package network

func publishEndpoint(string, string) error { return nil }
