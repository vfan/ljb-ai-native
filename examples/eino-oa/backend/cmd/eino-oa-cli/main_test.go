package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteCredentialsPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "credentials.json")
	c := &credentials{
		Server:       "http://localhost:8080",
		AccessToken:  "eo_at_test",
		RefreshToken: "eo_rt_test",
		ExpiresAt:    time.Now().Add(time.Hour),
		Scope:        "read,write",
	}
	if err := writeCredentials(path, c); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("credentials mode = %o, want 600", mode)
	}

	// 已存在的文件也必须被收紧到 0600，而不是沿用旧权限。
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeCredentials(path, c); err != nil {
		t.Fatal(err)
	}
	info, _ = os.Stat(path)
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("credentials mode after rewrite = %o, want 600", mode)
	}
}

func TestNewPKCE(t *testing.T) {
	verifier, challenge, err := newPKCE()
	if err != nil {
		t.Fatal(err)
	}
	if len(verifier) < 43 || len(verifier) > 128 {
		t.Fatalf("verifier length = %d", len(verifier))
	}
	if len(challenge) != 43 { // SHA-256 base64url 无填充固定 43 字符
		t.Fatalf("challenge length = %d", len(challenge))
	}
	if verifier == challenge {
		t.Fatal("verifier and challenge should differ")
	}
}

func TestNormalizeServer(t *testing.T) {
	if got := normalizeServer("  http://localhost:8080/// "); got != "http://localhost:8080" {
		t.Fatalf("normalizeServer = %q", got)
	}
	if got := normalizeServer(""); got != "http://localhost:8080" {
		t.Fatalf("default server = %q", got)
	}
}
