package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"discord-bypass/internal/doh"
	"discord-bypass/internal/launcher"
	"discord-bypass/internal/proxy"
	"discord-bypass/internal/sysproxy"
)

const (
	version = "1.0.2"
	banner  = `
=============================================================
  ____  _                       _   ____                              
 |  _ \(_)___  ___ ___  _ __ __| | | __ ) _   _ _ __   __ _ ___ ___  
 | | | | / __|/ __/ _ \| '__/ _` + "`" + ` | |  _ \| | | | '_ \ / _` + "`" + ` / __/ __| 
 | |_| | \__ \ (_| (_) | | | (_| | | |_) | |_| | |_) | (_| \__ \__ \ 
 |____/|_|___/\___\___/|_|  \__,_| |____/ \__, | .__/ \__,_|___/___/ 
                                          |___/|_|                   
                     TURKEY DPI BYPASS v%s
=============================================================
`
)

func main() {
	port := flag.Int("port", 8085, "Yerel vekil sunucu portu (HTTP & SOCKS5)")
	bind := flag.String("bind", "127.0.0.1", "Dinlenecek IP adresi")
	launchApp := flag.Bool("launch", true, "Discord uygulamasını otomatik başlat")
	useSysProxy := flag.Bool("sysproxy", true, "Windows sistem vekilini otomatik aç/kapat")
	flag.Parse()

	// Setup logging to both stdout and bypass.log
	exePath, err := os.Executable()
	var logDir string
	if err == nil {
		logDir = filepath.Dir(exePath)
	} else {
		logDir = "."
	}
	logFile, err := os.OpenFile(filepath.Join(logDir, "bypass.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		defer logFile.Close()
		multi := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(multi)
	}

	fmt.Printf(banner, version)
	fmt.Printf(" [•] İşletim Sistemi: %s (%s)\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf(" [•] Yerel Vekil:      http://%s:%d\n", *bind, *port)
	fmt.Printf(" [•] Kontrol Paneli:   http://%s:%d/\n", *bind, *port)
	fmt.Printf(" [•] Mod:              RFC-Uyumlu TLS Parçalama (0ms Ek Gecikme, VPN Yok)\n")
	fmt.Println("-------------------------------------------------------------")

	// Initialize DoH resolver
	resolver := doh.NewResolver()
	log.Println("[+] Cloudflare & Google DNS-over-HTTPS (DoH) çözücü hazır.")

	// Start Proxy Server
	addr := fmt.Sprintf("%s:%d", *bind, *port)
	srv := proxy.NewServer(addr, resolver)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[-] Proxy panic kurtarıldı: %v", r)
			}
		}()
		if err := srv.Start(); err != nil {
			log.Fatalf("[-] Sunucu başlatılamadı: %v", err)
		}
	}()

	log.Printf("[+] Proxy sunucusu %s adresinde dinliyor (HTTP CONNECT & SOCKS5)...", addr)

	// Automatically enable Windows System Proxy if requested
	if *useSysProxy {
		sysproxy.Enable(addr)
		defer sysproxy.Disable()
	}

	// Launch Discord
	if *launchApp {
		ok, msg := launcher.LaunchDiscord(addr)
		if ok {
			log.Printf("[+] %s", msg)
		} else {
			log.Printf("[i] %s", msg)
			fmt.Println("\n>>> İPUCU: Discord'u vekil sunucuyla manuel çalıştırmak için:")
			fmt.Printf("    Discord.exe --proxy-server=\"http://%s:%d\"\n\n", *bind, *port)
		}
	}

	fmt.Println("\n[!] Tünel devrede. Discord'u dilediğiniz gibi kullanabilirsiniz.")
	fmt.Println("[!] Kapatmak için terminalde CTRL + C tuşlarına basabilirsiniz.")
	fmt.Println("-------------------------------------------------------------")

	// Wait for terminate signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n[*] Uygulama kapatılıyor...")
	if *useSysProxy {
		sysproxy.Disable()
	}
	srv.Stop()
	fmt.Println("[*] Görüşmek üzere!")
}
