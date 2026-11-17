//go:build windows

package store

// Scaffold: Windows ACL hardening lands with the full store port.
func privateHome(string) error { return nil }

func tightenHome(string) {}
