package utils

import (
	"net/netip"
	"strings"
	"testing"
)

func TestResolve_DirectClient_IgnoresXFF(t *testing.T) {
	resolver, err := NewIPResolver("")
	if err != nil {
		t.Fatalf("NewIPResolver: %v", err)
	}

	ip, err := resolver.Resolve("1.2.3.4:5678", "9.9.9.9, 8.8.8.8")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := ip.String(); got != "1.2.3.4" {
		t.Fatalf("expected 1.2.3.4, got %s", got)
	}
}

func TestResolve_TrustedProxy_UsesXFF(t *testing.T) {
	resolver, err := NewIPResolver("10.0.0.0/8")
	if err != nil {
		t.Fatalf("NewIPResolver: %v", err)
	}

	ip, err := resolver.Resolve("10.1.2.3:5678", "1.2.3.4")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := ip.String(); got != "1.2.3.4" {
		t.Fatalf("expected 1.2.3.4, got %s", got)
	}
}

func TestResolve_TrustedProxy_ChainRightToLeft(t *testing.T) {
	resolver, err := NewIPResolver("10.0.0.0/8,192.168.0.0/16")
	if err != nil {
		t.Fatalf("NewIPResolver: %v", err)
	}

	ip, err := resolver.Resolve("10.1.2.3:5678", "1.2.3.4, 192.168.1.1, 10.0.0.5")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := ip.String(); got != "1.2.3.4" {
		t.Fatalf("expected 1.2.3.4, got %s", got)
	}
}

func TestResolve_SpoofedXFF_IgnoredWhenUntrusted(t *testing.T) {
	resolver, err := NewIPResolver("10.0.0.0/8")
	if err != nil {
		t.Fatalf("NewIPResolver: %v", err)
	}

	ip, err := resolver.Resolve("1.2.3.4:5678", "9.9.9.9")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := ip.String(); got != "1.2.3.4" {
		t.Fatalf("expected 1.2.3.4, got %s", got)
	}
}

func TestResolve_IPv6LoopbackUnmapped(t *testing.T) {
	resolver, err := NewIPResolver("")
	if err != nil {
		t.Fatalf("NewIPResolver: %v", err)
	}

	ip, err := resolver.Resolve("[::1]:5678", "")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !ip.Is4() || ip.String() != "127.0.0.1" {
		t.Fatalf("expected 127.0.0.1 (Is4), got %s (Is4=%v)", ip.String(), ip.Is4())
	}
}

func TestResolve_BadRemoteAddr_Error(t *testing.T) {
	resolver, err := NewIPResolver("")
	if err != nil {
		t.Fatalf("NewIPResolver: %v", err)
	}

	if _, err := resolver.Resolve("not-an-ip", ""); err == nil {
		t.Fatal("expected error for unparseable remote address")
	}
}

func TestHash_Deterministic(t *testing.T) {
	resolver, err := NewIPResolver("")
	if err != nil {
		t.Fatalf("NewIPResolver: %v", err)
	}

	addr, err := netip.ParseAddr("1.2.3.4")
	if err != nil {
		t.Fatalf("ParseAddr: %v", err)
	}

	h1 := resolver.Hash(addr, []byte("pepper"))
	h2 := resolver.Hash(addr, []byte("pepper"))

	if h1 != h2 {
		t.Fatalf("expected deterministic hash, got %s vs %s", h1, h2)
	}
	if len(h1) != 64 {
		t.Fatalf("expected 64-char hex, got %d chars", len(h1))
	}
	if strings.ToLower(h1) != h1 {
		t.Fatalf("expected lowercase hex, got %s", h1)
	}
}

func TestHash_DifferentPepperDifferentHash(t *testing.T) {
	resolver, err := NewIPResolver("")
	if err != nil {
		t.Fatalf("NewIPResolver: %v", err)
	}

	addr, err := netip.ParseAddr("1.2.3.4")
	if err != nil {
		t.Fatalf("ParseAddr: %v", err)
	}

	h1 := resolver.Hash(addr, []byte("pepper-one"))
	h2 := resolver.Hash(addr, []byte("pepper-two"))

	if h1 == h2 {
		t.Fatalf("expected different hashes for different peppers, got %s", h1)
	}
}

func TestHash_IPv6_BucketsTo64(t *testing.T) {
	resolver, err := NewIPResolver("")
	if err != nil {
		t.Fatalf("NewIPResolver: %v", err)
	}

	a1, err := netip.ParseAddr("2001:db8::1")
	if err != nil {
		t.Fatalf("ParseAddr a1: %v", err)
	}
	a2, err := netip.ParseAddr("2001:db8::ffff")
	if err != nil {
		t.Fatalf("ParseAddr a2: %v", err)
	}
	a3, err := netip.ParseAddr("2001:db9::1")
	if err != nil {
		t.Fatalf("ParseAddr a3: %v", err)
	}

	h1 := resolver.Hash(a1, []byte("pepper"))
	h2 := resolver.Hash(a2, []byte("pepper"))
	h3 := resolver.Hash(a3, []byte("pepper"))

	if h1 != h2 {
		t.Fatalf("expected same /64 hash, got %s vs %s", h1, h2)
	}
	if h1 == h3 {
		t.Fatalf("expected different hash for different /64, got %s", h1)
	}
}

func TestNewIPResolver_InvalidCIDR_Error(t *testing.T) {
	if _, err := NewIPResolver("not-a-cidr"); err == nil {
		t.Fatal("expected error for invalid CIDR")
	}
}
