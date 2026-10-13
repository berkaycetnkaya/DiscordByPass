//go:build windows

package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
)

// LaunchDiscord attempts to find and launch Discord on Windows with the proxy flag and environment
func LaunchDiscord(proxyAddr string) (bool, string) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return false, "LOCALAPPDATA ortam değişkeni bulunamadı."
	}

	// Terminate any previous stuck Discord instances
	exec.Command("taskkill", "/F", "/IM", "Discord.exe").Run()

	proxyURL := fmt.Sprintf("http://%s", proxyAddr)
	proxyFlag := fmt.Sprintf("--proxy-server=%s", proxyURL)

	env := append(os.Environ(),
		fmt.Sprintf("HTTP_PROXY=%s", proxyURL),
		fmt.Sprintf("HTTPS_PROXY=%s", proxyURL),
		fmt.Sprintf("http_proxy=%s", proxyURL),
		fmt.Sprintf("https_proxy=%s", proxyURL),
		fmt.Sprintf("ALL_PROXY=%s", proxyURL),
	)

	// Strategy 1: Find the latest app-* directory and launch Discord.exe directly
	// This ensures both Chromium and the native Rust updater (reqwest) receive proxy settings
	discordDir := filepath.Join(localAppData, "Discord")
	entries, err := os.ReadDir(discordDir)
	if err == nil {
		var appDirs []string
		for _, entry := range entries {
			if entry.IsDir() && filepath.HasPrefix(entry.Name(), "app-") {
				appDirs = append(appDirs, entry.Name())
			}
		}

		// Sort to get the latest version (e.g. app-1.0.9059)
		sort.Strings(appDirs)
		if len(appDirs) > 0 {
			latestAppDir := appDirs[len(appDirs)-1]
			appExe := filepath.Join(discordDir, latestAppDir, "Discord.exe")
			if _, err := os.Stat(appExe); err == nil {
				cmd := exec.Command(appExe, proxyFlag)
				cmd.Env = env
				if err := cmd.Start(); err == nil {
					return true, fmt.Sprintf("Discord doğrudan başlatıldı: %s", appExe)
				}
			}
		}
	}

	// Strategy 2: Update.exe fallback with proxy environment
	updaterPath := filepath.Join(localAppData, "Discord", "Update.exe")
	if _, err := os.Stat(updaterPath); err == nil {
		cmd := exec.Command(updaterPath, "--processStart", "Discord.exe")
		cmd.Env = env
		if err := cmd.Start(); err == nil {
			return true, fmt.Sprintf("Discord başlatıldı: %s (Update.exe üzerinden)", updaterPath)
		}
	}

	// Strategy 3: Check Discord Canary / PTB
	canaryDir := filepath.Join(localAppData, "DiscordCanary")
	if entries, err := os.ReadDir(canaryDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() && filepath.HasPrefix(entry.Name(), "app-") {
				appExe := filepath.Join(canaryDir, entry.Name(), "DiscordCanary.exe")
				if _, err := os.Stat(appExe); err == nil {
					cmd := exec.Command(appExe, proxyFlag)
					cmd.Env = env
					if err := cmd.Start(); err == nil {
						return true, fmt.Sprintf("Discord Canary başlatıldı: %s", appExe)
					}
				}
			}
		}
	}

	return false, "Discord masaüstü uygulaması bilgisayarınızda bulunamadı."
}
