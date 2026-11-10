//go:build !windows

package network

func publishEndpoint(string, string) error { return nil }
