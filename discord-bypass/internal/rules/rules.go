package rules

import (
	"strings"
)

var defaultDiscordSuffixes = []string{
	"discord.com",
	"discord.gg",
	"discordapp.com",
	"discordapp.net",
	"discord.media",
	"discordcdn.com",
	"discordstatus.com",
	"discord.co",
	"discord-activities.com",
	"dis.gd",
}

// IsDiscordDomain checks if a given host or domain belongs to Discord
func IsDiscordDomain(host string) bool {
	// Strip port if present
	if idx := strings.IndexByte(host, ':'); idx != -1 {
		host = host[:idx]
	}
	host = strings.ToLower(strings.TrimSpace(host))

	for _, suffix := range defaultDiscordSuffixes {
		if host == suffix || strings.HasSuffix(host, "."+suffix) {
			return true
		}
	}
	return false
}
