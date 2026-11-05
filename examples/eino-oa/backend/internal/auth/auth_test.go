package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestNormalizeScopes(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{"", "read", true},
		{"read", "read", true},
		{"write", "read,write", true},
		{"read,write", "read,write", true},
		{"read write", "read,write", true},
		{"read,admin", "", false},
	}
	for _, tt := range tests {
		got, err := normalizeScopes(tt.input)
		if tt.ok && err != nil {
			t.Fatalf("normalizeScopes(%q) error = %v", tt.input, err)
		}
		if !tt.ok {
			if err == nil {
				t.Fatalf("normalizeScopes(%q) expected error", tt.input)
			}
			continue
		}
		if got != tt.want {
			t.Fatalf("normalizeScopes(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestScopesToSetWriteImpliesRead(t *testing.T) {
	p := &Principal{Method: MethodPAT, Scopes: scopesToSet("write")}
	if !p.HasScope(ScopeWrite) || !p.HasScope(ScopeRead) {
		t.Fatal("write scope should imply read")
	}
	ro := &Principal{Method: MethodPAT, Scopes: scopesToSet("read")}
	if ro.HasScope(ScopeWrite) {
		t.Fatal("read token must not have write scope")
	}
	session := &Principal{Method: MethodSession}
	if !session.HasScope(ScopeWrite) {
		t.Fatal("session should have full access")
	}
}

func TestVerifyPKCE(t *testing.T) {
	verifier := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~"
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])

	if !verifyPKCE(verifier, challenge) {
		t.Fatal("expected PKCE verification to pass")
	}
	if verifyPKCE(verifier, challenge+"x") {
		t.Fatal("expected tampered challenge to fail")
	}
	if verifyPKCE("short", challenge) {
		t.Fatal("expected too-short verifier to fail")
	}
}

func TestPasswordHash(t *testing.T) {
	hash, err := hashPassword("s3cret-pass")
	if err != nil {
		t.Fatal(err)
	}
	if !checkPassword(hash, "s3cret-pass") {
		t.Fatal("password should verify")
	}
	if checkPassword(hash, "wrong-pass") {
		t.Fatal("wrong password must not verify")
	}

	short, err := hashPassword("123456")
	if err != nil {
		t.Fatalf("short password should be allowed: %v", err)
	}
	if !checkPassword(short, "123456") {
		t.Fatal("short password should verify")
	}
}
