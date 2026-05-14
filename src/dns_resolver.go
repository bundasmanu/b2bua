package main

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/emiago/sipgo/sip"
)

func newDNSResolver() *dnsResolver {
	return &dnsResolver{cache: make(map[string]*dnsEntry)}
}

func (r *dnsResolver) resolve(ctx context.Context, host string) ([]net.IP, error) {
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	return ips, nil
}

func (r *dnsResolver) ensureEntry(ctx context.Context, host string) (*dnsEntry, error) {
	r.mu.Lock()
	entry, ok := r.cache[host]
	if ok && len(entry.ips) > 0 {
		r.mu.Unlock()
		return entry, nil
	}
	r.mu.Unlock()

	ips, err := r.resolve(ctx, host)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	entry = &dnsEntry{ips: ips, updatedAt: time.Now()}
	r.cache[host] = entry
	return entry, nil
}

func (r *dnsResolver) refreshEntry(ctx context.Context, host string) (*dnsEntry, error) {
	ips, err := r.resolve(ctx, host)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	entry := r.cache[host]
	if entry == nil {
		entry = &dnsEntry{}
		r.cache[host] = entry
	}
	entry.ips = ips
	entry.updatedAt = time.Now()
	entry.nextIndex = 0
	return entry, nil
}

func (r *dnsResolver) VerifySource(ctx context.Context, source, host string) (bool, error) {
	ipStr, _, err := net.SplitHostPort(source)
	if err != nil {
		ipStr = source
	}
	remoteIP := net.ParseIP(ipStr)
	if remoteIP == nil {
		return false, nil
	}

	entry, err := r.ensureEntry(ctx, host)
	if err != nil {
		return false, err
	}

	if ipMatches(remoteIP, entry.ips) {
		return true, nil
	}

	entry, err = r.refreshEntry(ctx, host)
	if err != nil {
		return false, err
	}

	return ipMatches(remoteIP, entry.ips), nil
}

func (r *dnsResolver) NextRoundRobinURI(ctx context.Context, host, port string) (sip.Uri, error) {
	entry, err := r.ensureEntry(ctx, host)
	if err != nil {
		return sip.Uri{}, err
	}

	if len(entry.ips) == 0 {
		return sip.Uri{}, errors.New("no resolved IPs available for host")
	}

	r.mu.Lock()
	idx := entry.nextIndex % len(entry.ips)
	entry.nextIndex++
	r.mu.Unlock()

	chosen := entry.ips[idx].String()
	uri := sip.Uri{}
	if err := sip.ParseUri("sip:"+chosen+":"+port, &uri); err != nil {
		return sip.Uri{}, err
	}
	return uri, nil
}
