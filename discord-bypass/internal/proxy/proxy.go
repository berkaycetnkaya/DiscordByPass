package proxy

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"discord-bypass/internal/doh"
	"discord-bypass/internal/dpi"
	"discord-bypass/internal/rules"
)

// Server is a dual-protocol (HTTP CONNECT & SOCKS5) proxy with DPI bypass
type Server struct {
	addr         string
	resolver     *doh.Resolver
	listener     net.Listener
	mu           sync.Mutex
	running      bool
	bypassedReqs uint64
	totalBytes   uint64
	activeConns  int64
	startTime    time.Time
}

// NewServer creates a new proxy server
func NewServer(addr string, resolver *doh.Resolver) *Server {
	return &Server{
		addr:      addr,
		resolver:  resolver,
		startTime: time.Now(),
	}
}

// Start runs the proxy listener
func (s *Server) Start() error {
	l, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.listener = l
	s.running = true

	for s.running {
		conn, err := l.Accept()
		if err != nil {
			if !s.running {
				break
			}
			continue
		}
		go s.handleConnection(conn)
	}
	return nil
}

// Stop stops the server
func (s *Server) Stop() {
	s.running = false
	if s.listener != nil {
		s.listener.Close()
	}
}

func (s *Server) handleConnection(clientConn net.Conn) {
	atomic.AddInt64(&s.activeConns, 1)
	defer atomic.AddInt64(&s.activeConns, -1)
	defer clientConn.Close()

	reader := bufio.NewReader(clientConn)
	peek, err := reader.Peek(1)
	if err != nil {
		return
	}

	if peek[0] == 0x05 {
		// SOCKS5 Protocol
		s.handleSOCKS5(clientConn, reader)
	} else {
		// HTTP Protocol (CONNECT or normal GET/web)
		s.handleHTTP(clientConn, reader)
	}
}

// handleHTTP handles HTTP CONNECT tunnels or serves web dashboard
func (s *Server) handleHTTP(clientConn net.Conn, reader *bufio.Reader) {
	req, err := http.ReadRequest(reader)
	if err != nil {
		return
	}

	if req.Method == http.MethodConnect {
		// HTTP CONNECT tunnel
		target := req.Host
		s.tunnel(clientConn, target, "HTTP CONNECT")
		return
	}

	// Web dashboard / health check request
	s.serveDashboard(clientConn, req)
}

// handleSOCKS5 handles SOCKS5 handshake and connection requests
func (s *Server) handleSOCKS5(clientConn net.Conn, reader *bufio.Reader) {
	// Step 1: Read version and auth methods
	header := make([]byte, 2)
	if _, err := io.ReadFull(reader, header); err != nil {
		return
	}
	numMethods := int(header[1])
	methods := make([]byte, numMethods)
	if _, err := io.ReadFull(reader, methods); err != nil {
		return
	}

	// Respond with NO AUTH REQUIRED (0x00)
	if _, err := clientConn.Write([]byte{0x05, 0x00}); err != nil {
		return
	}

	// Step 2: Read request details
	reqHeader := make([]byte, 4)
	if _, err := io.ReadFull(reader, reqHeader); err != nil {
		return
	}
	cmd := reqHeader[1]
	addrType := reqHeader[3]

	if cmd != 0x01 { // 0x01 = CONNECT
		clientConn.Write([]byte{0x05, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	var targetHost string
	switch addrType {
	case 0x01: // IPv4
		ipBuf := make([]byte, 4)
		if _, err := io.ReadFull(reader, ipBuf); err != nil {
			return
		}
		targetHost = net.IP(ipBuf).String()
	case 0x03: // Domain name
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(reader, lenBuf); err != nil {
			return
		}
		domainLen := int(lenBuf[0])
		domainBuf := make([]byte, domainLen)
		if _, err := io.ReadFull(reader, domainBuf); err != nil {
			return
		}
		targetHost = string(domainBuf)
	case 0x04: // IPv6
		ipBuf := make([]byte, 16)
		if _, err := io.ReadFull(reader, ipBuf); err != nil {
			return
		}
		targetHost = net.IP(ipBuf).String()
	default:
		clientConn.Write([]byte{0x05, 0x08, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(reader, portBuf); err != nil {
		return
	}
	port := binary.BigEndian.Uint16(portBuf)
	target := fmt.Sprintf("%s:%d", targetHost, port)

	// Send SOCKS5 SUCCESS reply
	if _, err := clientConn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0}); err != nil {
		return
	}

	s.tunnel(clientConn, target, "SOCKS5")
}

func (s *Server) tunnel(clientConn net.Conn, target string, proto string) {
	host, portStr, err := net.SplitHostPort(target)
	if err != nil {
		host = target
		portStr = "443"
	}
	port, _ := strconv.Atoi(portStr)

	isDiscord := rules.IsDiscordDomain(host)

	var targetIP string
	if isDiscord {
		// Use DoH to resolve Discord domains cleanly
		resolvedIP, err := s.resolver.Resolve(host)
		if err != nil {
			log.Printf("[DoH Error] Host '%s' çözülemedi: %v", host, err)
			return
		}
		targetIP = resolvedIP
	} else {
		// Direct DNS for other targets
		targetIP = host
	}

	// Dial target server
	dialTarget := net.JoinHostPort(targetIP, strconv.Itoa(port))
	remoteConn, err := net.DialTimeout("tcp", dialTarget, 5*time.Second)
	if err != nil {
		log.Printf("[Hata] %s sunucusuna bağlanılamadı: %v", dialTarget, err)
		return
	}
	defer remoteConn.Close()

	if tcpConn, ok := remoteConn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
	}

	if proto == "HTTP CONNECT" {
		// Send 200 Connection Established to client
		clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	}

	var outbound net.Conn = remoteConn
	if isDiscord && port == 443 {
		atomic.AddUint64(&s.bypassedReqs, 1)
		log.Printf("⚡ [DPI Bypass] %s -> %s (%s) [TLS Parçalama Aktif]", host, targetIP, proto)
		outbound = dpi.NewRecordSplitConn(remoteConn)
	} else {
		log.Printf("➡️  [%s] %s -> %s (Direkt Geçiş)", proto, host, targetIP)
	}

	// Bidirectional stream
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		n, _ := io.Copy(outbound, clientConn)
		atomic.AddUint64(&s.totalBytes, uint64(n))
		if tc, ok := outbound.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
	}()

	go func() {
		defer wg.Done()
		n, _ := io.Copy(clientConn, outbound)
		atomic.AddUint64(&s.totalBytes, uint64(n))
		if tc, ok := clientConn.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
	}()

	wg.Wait()
}

func (s *Server) serveDashboard(conn net.Conn, req *http.Request) {
	if req.URL.Path == "/ping" {
		conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\nPONG"))
		return
	}

	uptime := time.Since(s.startTime).Round(time.Second)
	bypassed := atomic.LoadUint64(&s.bypassedReqs)
	active := atomic.LoadInt64(&s.activeConns)

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="tr">
<head>
  <meta charset="UTF-8">
  <title>Discord DPI Bypass Kontrol Paneli</title>
  <style>
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      font-family: 'Segoe UI', -apple-system, BlinkMacSystemFont, Roboto, sans-serif;
      background: #0f111a;
      color: #e3e5e8;
      display: flex;
      justify-content: center;
      align-items: center;
      min-height: 100vh;
      padding: 20px;
    }
    .card {
      background: #1e1f29;
      border: 1px solid #2b2d3d;
      border-radius: 16px;
      padding: 32px;
      max-width: 520px;
      width: 100%%;
      box-shadow: 0 12px 32px rgba(0,0,0,0.5);
    }
    .header {
      display: flex;
      align-items: center;
      gap: 14px;
      margin-bottom: 24px;
    }
    .icon {
      width: 48px;
      height: 48px;
      background: #5865F2;
      border-radius: 12px;
      display: flex;
      align-items: center;
      justify-content: center;
      font-weight: bold;
      font-size: 24px;
      color: white;
    }
    h1 { font-size: 22px; color: #fff; }
    .badge {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      background: rgba(35, 165, 89, 0.15);
      color: #23a55a;
      padding: 4px 10px;
      border-radius: 20px;
      font-size: 13px;
      font-weight: 600;
      margin-top: 4px;
    }
    .badge-dot {
      width: 8px;
      height: 8px;
      border-radius: 50%%;
      background: #23a55a;
      box-shadow: 0 0 8px #23a55a;
    }
    .stats {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 12px;
      margin-bottom: 24px;
    }
    .stat-box {
      background: #2b2d3d;
      border-radius: 12px;
      padding: 16px;
      text-align: center;
    }
    .stat-val { font-size: 24px; font-weight: bold; color: #5865F2; }
    .stat-label { font-size: 12px; color: #949ba4; margin-top: 4px; text-transform: uppercase; }
    .info {
      background: #181920;
      border-radius: 12px;
      padding: 16px;
      font-size: 13px;
      line-height: 1.6;
      color: #b5bac1;
      margin-bottom: 20px;
    }
    .info b { color: #fff; }
    .code {
      background: #111217;
      padding: 8px 12px;
      border-radius: 8px;
      font-family: Consolas, monospace;
      color: #00e5a3;
      margin-top: 8px;
      display: block;
      word-break: break-all;
    }
  </style>
</head>
<body>
  <div class="card">
    <div class="header">
      <div class="icon">D</div>
      <div>
        <h1>Discord DPI Bypass</h1>
        <div class="badge"><div class="badge-dot"></div> Tünel Çalışıyor (0ms Ping)</div>
      </div>
    </div>
    <div class="stats">
      <div class="stat-box">
        <div class="stat-val">%d</div>
        <div class="stat-label">Bypass Edilen İstek</div>
      </div>
      <div class="stat-box">
        <div class="stat-val">%d</div>
        <div class="stat-label">Aktif Bağlantı</div>
      </div>
    </div>
    <div class="info">
      <div><b>Çalışma Süresi:</b> %s</div>
      <div><b>Yerel Vekil Adresi:</b> 127.0.0.1:8085</div>
      <div><b>Protokol Desteği:</b> HTTP CONNECT ve SOCKS5 (Otomatik)</div>
      <div style="margin-top: 8px;">Discord uygulamasını çalıştırma komutu:</div>
      <span class="code">Discord.exe --proxy-server="http://127.0.0.1:8085"</span>
    </div>
  </div>
</body>
</html>`, bypassed, active, uptime)

	resp := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/html; charset=utf-8\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", len(html), html)
	conn.Write([]byte(resp))
}
