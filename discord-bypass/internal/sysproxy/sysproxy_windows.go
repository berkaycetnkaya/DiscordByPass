//go:build windows

package sysproxy

import (
	"log"
	"os/exec"
)

const internetSettingsKey = `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`

// Enable sets the Windows system proxy to the local bypass server
func Enable(proxyAddr string) {
	// Set ProxyServer
	cmd1 := exec.Command("reg", "add", internetSettingsKey, "/v", "ProxyServer", "/t", "REG_SZ", "/d", proxyAddr, "/f")
	if err := cmd1.Run(); err != nil {
		log.Printf("[-] ProxyServer reg eklenemedi: %v", err)
	}

	// Set ProxyEnable = 1
	cmd2 := exec.Command("reg", "add", internetSettingsKey, "/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "1", "/f")
	if err := cmd2.Run(); err != nil {
		log.Printf("[-] ProxyEnable reg eklenemedi: %v", err)
	} else {
		log.Println("[+] Windows Sistem Vekil Sunucusu (Proxy) otomatik aktif edildi.")
	}
}

// Disable restores Windows system proxy settings
func Disable() {
	cmd := exec.Command("reg", "add", internetSettingsKey, "/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "0", "/f")
	if err := cmd.Run(); err != nil {
		log.Printf("[-] Proxy devre dışı bırakılamadı: %v", err)
	} else {
		log.Println("[+] Windows Sistem Vekil Sunucusu kapatıldı, normal ayarlara dönüldü.")
	}
}
