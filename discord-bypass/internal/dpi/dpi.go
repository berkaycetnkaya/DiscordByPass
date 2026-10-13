package dpi

import (
	"bytes"
	"net"
	"time"
)

// RecordSplitConn wraps an outbound TCP connection to split TLS ClientHello
// across multiple RFC-compliant TLS records, bypassing ISP SNI DPI filters.
type RecordSplitConn struct {
	net.Conn
	handshakeDone bool
	splitSize     int
	pauseDuration time.Duration
}

// NewRecordSplitConn wraps a connection with TLS record fragmentation capability
func NewRecordSplitConn(c net.Conn) *RecordSplitConn {
	if tcpConn, ok := c.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
	}

	return &RecordSplitConn{
		Conn:          c,
		splitSize:     40, // 40 bytes leaves SNI completely in the second record
		pauseDuration: 10 * time.Millisecond,
	}
}

func (c *RecordSplitConn) Write(b []byte) (int, error) {
	if !c.handshakeDone && len(b) > 5 && b[0] == 0x16 && b[1] == 0x03 {
		// Valid TLS Handshake record
		recLen := int(b[3])<<8 | int(b[4])
		if len(b) >= 5+recLen {
			handshakeData := b[5 : 5+recLen]

			split := c.splitSize
			if split > len(handshakeData)-10 {
				split = len(handshakeData) / 2
			}

			part1 := handshakeData[:split]
			part2 := handshakeData[split:]

			// TLS Record 1: Contains first chunk (Handshake type 0x01, random, etc. No SNI!)
			r1 := []byte{0x16, 0x03, 0x01, byte(len(part1) >> 8), byte(len(part1) & 0xff)}
			r1 = append(r1, part1...)

			// TLS Record 2: Contains remaining chunk (including SNI)
			r2 := []byte{0x16, 0x03, 0x01, byte(len(part2) >> 8), byte(len(part2) & 0xff)}
			r2 = append(r2, part2...)

			c.handshakeDone = true

			// Write Record 1
			if _, err := c.Conn.Write(r1); err != nil {
				return 0, err
			}

			// Brief pause to ensure separate TCP packets
			time.Sleep(c.pauseDuration)

			// Write Record 2
			if _, err := c.Conn.Write(r2); err != nil {
				return len(r1), err
			}

			// Report original length consumed
			return len(b), nil
		}
	}

	// Normal write
	return c.Conn.Write(b)
}

// ExtractSNI extracts the Server Name Indication (SNI) from a raw TLS ClientHello
func ExtractSNI(data []byte) string {
	if len(data) < 5 || data[0] != 0x16 {
		return ""
	}
	// Handshake data begins at offset 5
	pos := 5
	if len(data) < pos+4 || data[pos] != 0x01 { // 0x01 = ClientHello
		return ""
	}
	pos += 4 // Skip Handshake Type (1) + Handshake Length (3)
	pos += 2 // Skip Client Version (2)
	pos += 32 // Skip Random (32)

	if len(data) < pos+1 {
		return ""
	}
	sessionIDLen := int(data[pos])
	pos += 1 + sessionIDLen

	if len(data) < pos+2 {
		return ""
	}
	cipherSuitesLen := int(data[pos])<<8 | int(data[pos+1])
	pos += 2 + cipherSuitesLen

	if len(data) < pos+1 {
		return ""
	}
	compressionMethodsLen := int(data[pos])
	pos += 1 + compressionMethodsLen

	if len(data) < pos+2 {
		return ""
	}
	extensionsLen := int(data[pos])<<8 | int(data[pos+1])
	pos += 2

	end := pos + extensionsLen
	if end > len(data) {
		end = len(data)
	}

	for pos+4 <= end {
		extType := int(data[pos])<<8 | int(data[pos+1])
		extLen := int(data[pos+2])<<8 | int(data[pos+3])
		pos += 4

		if extType == 0x0000 { // server_name extension
			if pos+2 <= end {
				// Server name list length (2 bytes)
				sniPos := pos + 2
				for sniPos+3 <= pos+extLen && sniPos+3 <= end {
					nameType := data[sniPos]
					nameLen := int(data[sniPos+1])<<8 | int(data[sniPos+2])
					sniPos += 3
					if nameType == 0x00 && sniPos+nameLen <= end { // host_name
						return string(data[sniPos : sniPos+nameLen])
					}
					sniPos += nameLen
				}
			}
		}
		pos += extLen
	}
	return ""
}

// ContainsDomain searches for domain bytes in packet
func ContainsDomain(data []byte, domain string) bool {
	return bytes.Contains(bytes.ToLower(data), []byte(stringsToLower(domain)))
}

func stringsToLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
