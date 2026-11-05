package handler

import (
	"testing"
	"time"
)

func TestValidateLoopbackRedirect(t *testing.T) {
	valid := []string{
		"http://127.0.0.1:12345/callback",
		"http://localhost:8080/callback",
		"http://[::1]:9000/callback",
	}
	for _, raw := range valid {
		if err := validateLoopbackRedirect(raw); err != nil {
			t.Fatalf("validateLoopbackRedirect(%q) = %v, want nil", raw, err)
		}
	}

	invalid := []string{
		"https://127.0.0.1:12345/callback", // 只允许本机 http
		"http://evil.com/callback",
		"http://127.0.0.1:12345/other",
		"http://user@127.0.0.1:12345/callback",
		"http://127.0.0.1:12345/callback#fragment",
		"javascript:alert(1)",
	}
	for _, raw := range invalid {
		if err := validateLoopbackRedirect(raw); err == nil {
			t.Fatalf("validateLoopbackRedirect(%q) expected error", raw)
		}
	}
}

func TestNormalizeScopeList(t *testing.T) {
	got, err := normalizeScopeList([]string{"read", "write"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "read" || got[1] != "write" {
		t.Fatalf("unexpected scopes: %#v", got)
	}

	got, err = normalizeScopeList(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "read" {
		t.Fatalf("default scope should be read, got %#v", got)
	}

	if _, err := normalizeScopeList([]string{"admin"}); err == nil {
		t.Fatal("unknown scope should fail")
	}
}

func TestLoginLimiter(t *testing.T) {
	l := newLoginLimiter(2, time.Minute)
	if !l.Allow("1.2.3.4") || !l.Allow("1.2.3.4") {
		t.Fatal("first two attempts should be allowed")
	}
	if l.Allow("1.2.3.4") {
		t.Fatal("third attempt should be blocked")
	}
	if !l.Allow("5.6.7.8") {
		t.Fatal("different key should not be blocked")
	}
	l.Reset("1.2.3.4")
	if !l.Allow("1.2.3.4") {
		t.Fatal("attempt should be allowed after reset")
	}
}
