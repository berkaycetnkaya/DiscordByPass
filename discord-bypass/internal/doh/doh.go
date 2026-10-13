package doh

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"
)

type dnsResponse struct {
	Status int `json:"Status"`
	Answer []struct {
		Name string `json:"name"`
		Type int    `json:"type"`
		TTL  int    `json:"TTL"`
		Data string `json:"data"`
	} `json:"Answer"`
}

type cacheEntry struct {
	ips     []string
	expires time.Time
}

// Resolver resolves hostnames using DNS-over-HTTPS
type Resolver struct {
	client     *http.Client
	cache      map[string]cacheEntry
	cacheMutex sync.RWMutex
}

// NewResolver creates a new DoH resolver
func NewResolver() *Resolver {
	// Custom transport with direct IP dialing for DoH to avoid DNS recursion
	tr := &http.Transport{
		Proxy:                 nil,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &Resolver{
		client: &http.Client{
			Transport: tr,
			Timeout:   5 * time.Second,
		},
		cache: make(map[string]cacheEntry),
	}
}

// Resolve returns an IP for the given host using DoH with caching
func (r *Resolver) Resolve(host string) (string, error) {
	// Check if already an IP
	if ip := net.ParseIP(host); ip != nil {
		return host, nil
	}

	// Check cache
	r.cacheMutex.RLock()
	if entry, found := r.cache[host]; found {
		if time.Now().Before(entry.expires) && len(entry.ips) > 0 {
			r.cacheMutex.RUnlock()
			return entry.ips[rand.Intn(len(entry.ips))], nil
		}
	}
	r.cacheMutex.RUnlock()

	// Query Cloudflare DoH first
	ips, ttl, err := r.queryDoH(fmt.Sprintf("https://1.1.1.1/dns-query?name=%s&type=A", host))
	if err != nil || len(ips) == 0 {
		// Fallback to Google DoH
		ips, ttl, err = r.queryDoH(fmt.Sprintf("https://8.8.8.8/resolve?name=%s&type=A", host))
	}

	if err != nil || len(ips) == 0 {
		// Last resort: system resolver
		sysIPs, sysErr := net.LookupIP(host)
		if sysErr != nil {
			return "", fmt.Errorf("DoH and system resolution failed: %v", err)
		}
		for _, ip := range sysIPs {
			if ipv4 := ip.To4(); ipv4 != nil {
				ips = append(ips, ipv4.String())
			}
		}
		ttl = 60
	}

	if len(ips) == 0 {
		return "", fmt.Errorf("no IPv4 address found for %s", host)
	}

	if ttl < 30 {
		ttl = 30
	}

	// Cache result
	r.cacheMutex.Lock()
	r.cache[host] = cacheEntry{
		ips:     ips,
		expires: time.Now().Add(time.Duration(ttl) * time.Second),
	}
	r.cacheMutex.Unlock()

	return ips[rand.Intn(len(ips))], nil
}

func (r *Resolver) queryDoH(urlStr string) ([]string, int, error) {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/dns-json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("DoH server returned status %d", resp.StatusCode)
	}

	var dnsResp dnsResponse
	if err := json.NewDecoder(resp.Body).Decode(&dnsResp); err != nil {
		return nil, 0, err
	}

	var ips []string
	ttl := 300
	for _, ans := range dnsResp.Answer {
		if ans.Type == 1 { // Type 1 = A record (IPv4)
			if ip := net.ParseIP(ans.Data); ip != nil {
				ips = append(ips, ans.Data)
				if ans.TTL > 0 && ans.TTL < ttl {
					ttl = ans.TTL
				}
			}
		}
	}
	return ips, ttl, nil
}
