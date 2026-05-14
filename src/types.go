package main

import (
	"net"
	"sync"
	"time"
)

type SIPEndpoint struct {
	Protocol string // "udp", "tcp", "tls", "ws", "wss"
	Host     string
	Port     int
}

type dnsResolver struct {
	mu    sync.Mutex
	cache map[string]*dnsEntry
}

type dnsEntry struct {
	ips       []net.IP
	updatedAt time.Time
	nextIndex int
}
