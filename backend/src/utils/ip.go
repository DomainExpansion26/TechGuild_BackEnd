package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strings"
)

// IPResolver resolves the true client IP from a request's remote address and
// its X-Forwarded-For header, honouring a configurable set of trusted proxy
// CIDRs. It also derives a stable, privacy-preserving hash of a resolved IP.
//
// The zero value is not usable; construct with NewIPResolver.
type IPResolver struct {
	trusted []netip.Prefix
}

// NewIPResolver parses a comma-separated list of trusted proxy CIDRs.
// An empty (or whitespace-only) string yields a resolver that trusts no proxy,
// so X-Forwarded-For is always ignored. Any unparseable CIDR is an error, so
// misconfiguration fails closed rather than silently trusting a bad range.
func NewIPResolver(trustedCIDRsCSV string) (*IPResolver, error) {
	r := &IPResolver{}

	csv := strings.TrimSpace(trustedCIDRsCSV)
	if csv == "" {
		return r, nil
	}

	for _, raw := range strings.Split(csv, ",") {
		part := strings.TrimSpace(raw)
		if part == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(part)
		if err != nil {
			return nil, fmt.Errorf("invalid trusted proxy CIDR %q: %w", part, err)
		}
		r.trusted = append(r.trusted, prefix.Masked())
	}

	return r, nil
}

// Resolve returns the client IP for a request.
//
// remoteAddr is the listener's view of the peer ("ip:port" or bare "ip").
// If that peer is not inside a trusted CIDR it is returned directly and
// forwardedFor is ignored (a client cannot spoof its own address). If the peer
// IS trusted, forwardedFor is parsed right-to-left and the first address that
// is not itself trusted is returned; if none qualifies, the peer's address is
// returned. An unparseable remoteAddr is an error.
func (r *IPResolver) Resolve(remoteAddr, forwardedFor string) (netip.Addr, error) {
	remoteIP, err := parseHostIP(remoteAddr)
	if err != nil {
		return netip.Addr{}, err
	}
	remoteIP = normalizeAddr(remoteIP)

	if !r.isTrusted(remoteIP) {
		return remoteIP, nil
	}

	parts := strings.Split(forwardedFor, ",")
	for i := len(parts) - 1; i >= 0; i-- {
		candidate := strings.TrimSpace(parts[i])
		if candidate == "" {
			continue
		}
		addr, err := netip.ParseAddr(candidate)
		if err != nil {
			continue
		}
		addr = normalizeAddr(addr)
		if !r.isTrusted(addr) {
			return addr, nil
		}
	}

	return remoteIP, nil
}

// Hash returns a lowercase hex, 64-character HMAC-SHA256 of the address keyed
// with pepper. IPv4 addresses are hashed directly. IPv6 addresses are first
// bucketed to their /64 prefix so that clients rotating within a subnet map to
// the same key.
func (r *IPResolver) Hash(addr netip.Addr, pepper []byte) string {
	key := normalizeAddr(addr)
	if key.Is6() {
		key = netip.PrefixFrom(key, 64).Masked().Addr()
	}

	mac := hmac.New(sha256.New, pepper)
	mac.Write(key.AsSlice())
	return hex.EncodeToString(mac.Sum(nil))
}

func (r *IPResolver) isTrusted(addr netip.Addr) bool {
	for _, prefix := range r.trusted {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

// parseHostIP extracts an IP from "ip:port", "ip", "[ip]:port" or "[ip]".
func parseHostIP(raw string) (netip.Addr, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return netip.Addr{}, errors.New("empty remote address")
	}

	if host, _, err := net.SplitHostPort(s); err == nil {
		if ip, perr := netip.ParseAddr(host); perr == nil {
			return ip, nil
		}
	}

	trimmed := s
	if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
		trimmed = trimmed[1 : len(trimmed)-1]
	}
	if ip, err := netip.ParseAddr(trimmed); err == nil {
		return ip, nil
	}

	return netip.Addr{}, fmt.Errorf("cannot parse remote address %q", raw)
}

// normalizeAddr maps IPv4-mapped IPv6 addresses back to IPv4 and folds the
// IPv6 loopback ::1 onto the IPv4 loopback 127.0.0.1.
func normalizeAddr(addr netip.Addr) netip.Addr {
	if addr == netip.IPv6Loopback() {
		return netip.AddrFrom4([4]byte{127, 0, 0, 1})
	}
	return addr.Unmap()
}
