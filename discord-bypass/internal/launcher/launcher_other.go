//go:build !windows

package launcher

import (
	"fmt"
	"os/exec"
	"runtime"
)

// LaunchDiscord attempts to launch Discord on Linux or macOS
func LaunchDiscord(proxyAddr string) (bool, string) {
	proxyFlag := fmt.Sprintf("--proxy-server=http://%s", proxyAddr)

	if runtime.GOOS == "darwin" {
		// macOS
		cmd := exec.Command("open", "-a", "Discord", "--args", proxyFlag)
		if err := cmd.Start(); err == nil {
			return true, "Discord (macOS) başlatıldı."
		}
	} else {
		// Linux
		cmd := exec.Command("discord", proxyFlag)
		if err := cmd.Start(); err == nil {
			return true, "Discord (Linux) başlatıldı."
		}
	}

	return false, "Discord uygulaması sistemde bulunamadı."
}
