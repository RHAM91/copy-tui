package updater

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"
)

// The test vector of the catalog integration guide.
const (
	vecKey = "A6EHv/POEL4dcN0Y50vAmWfk1jCbpQ1fHdyGZBJVMbg="
	vecSha = "a948904f2f0f479b8f8197694b30184b0d2ed1c1cd2a1ec0fb85d299a192a447"
	vecSig = "MHyP3cImcjTO5Ext9OyQylQN+FSBrQx4DgWgihS8Sgl2bUrHXKCSAXgkpsoBJXxZEiZ878a2X5N2i5zaLYm+Dw=="
)

func TestVerifyAcceptsTestVector(t *testing.T) {
	if err := Verify(vecKey, "demo-app", "1.2.3", "linux", "amd64", vecSha, vecSig); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyRejectsAlteredInputs(t *testing.T) {
	other, _, _ := ed25519.GenerateKey(nil)
	tests := []struct {
		name                       string
		key, app, ver, os, arch, h string
	}{
		{"wrong key", base64.StdEncoding.EncodeToString(other), "demo-app", "1.2.3", "linux", "amd64", vecSha},
		{"wrong version", vecKey, "demo-app", "1.2.4", "linux", "amd64", vecSha},
		{"wrong app", vecKey, "other", "1.2.3", "linux", "amd64", vecSha},
		{"wrong arch", vecKey, "demo-app", "1.2.3", "linux", "arm64", vecSha},
		{"flipped hash", vecKey, "demo-app", "1.2.3", "linux", "amd64", "b" + vecSha[1:]},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Verify(tt.key, tt.app, tt.ver, tt.os, tt.arch, tt.h, vecSig); err == nil {
				t.Fatal("Verify accepted an altered input")
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.10.0", "1.9.0", 1},
		{"1.0.0", "1.0.0-rc.1", 1},
		{"1.0.0-rc.1", "1.0.0", -1},
		{"1.0.0", "v1.0.0", 0},
		{"0.1.0", "0.2.0", -1},
	}
	for _, tt := range tests {
		if got, err := CompareVersions(tt.a, tt.b); err != nil || got != tt.want {
			t.Errorf("CompareVersions(%q, %q) = %d, %v; want %d", tt.a, tt.b, got, err, tt.want)
		}
	}
	if _, err := CompareVersions("dev", "1.0.0"); err == nil {
		t.Error("CompareVersions accepted \"dev\"")
	}
}

func TestCheckURLRejectsPlainHTTPOutsideLoopback(t *testing.T) {
	if err := (Config{BaseURL: "http://updates.example.com"}).checkURL(); err == nil {
		t.Fatal("plain HTTP to a remote host was accepted")
	}
	if err := (Config{BaseURL: "http://127.0.0.1:8080"}).checkURL(); err != nil {
		t.Fatal(err)
	}
}
