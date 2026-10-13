//go:build !windows

package sysproxy

// Enable is a no-op on non-Windows platforms
func Enable(proxyAddr string) {}

// Disable is a no-op on non-Windows platforms
func Disable() {}
