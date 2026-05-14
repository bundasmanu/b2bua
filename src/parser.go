package main

import (
	"net"
	"strconv"
)

func getSIPEndpoint(addr string) SIPEndpoint {
	// Format: [protocol:]host[:port]
	// Examples: udp:xpto.com:5060, xpto.com:5070, xpto.com
	
	cfg := SIPEndpoint{
		Protocol: "udp",
		Port:     5060,
	}
	
	parts := splitAddr(addr)
	
	if len(parts) == 0 {
		cfg.Host = "127.0.0.1"
		return cfg
	}
	
	// Check if first part is a protocol
	if isProtocol(parts[0]) {
		cfg.Protocol = parts[0]
		parts = parts[1:]
	}
	
	// Remaining parts: [host] [port]
	if len(parts) > 0 {
		cfg.Host = parts[0]
	}
	if len(parts) > 1 {
		cfg.Port = parsePort(parts[1])
	}
	
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	
	return cfg
}

func splitAddr(addr string) []string {
	var parts []string
	var current string
	
	for _, ch := range addr {
		if ch == ':' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func isProtocol(s string) bool {
	switch s {
	case "udp", "tcp", "tls", "ws", "wss":
		return true
	}
	return false
}

func parsePort(port string) int {
	if port == "" {
		return 5060
	}
	p, err := strconv.Atoi(port)
	if err != nil || p <= 0 || p > 65535 {
		return 5060
	}
	return p
}

func parseHostPort(value, defaultPort string) (string, string) {
	host, port, err := net.SplitHostPort(value)
	if err == nil {
		return host, port
	}
	return value, defaultPort
}

func ipMatches(remote net.IP, ips []net.IP) bool {
	for _, ip := range ips {
		if ip.Equal(remote) {
			return true
		}
	}
	return false
}
