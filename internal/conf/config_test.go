package conf

import "testing"

func TestLoadUsesDefaultHTTPAddress(t *testing.T) {
	cfg := Load(func(string) string { return "" })

	if cfg.HTTPAddr != "0.0.0.0:8000" {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, "0.0.0.0:8000")
	}
}

func TestLoadUsesConfiguredHTTPAddress(t *testing.T) {
	cfg := Load(func(key string) string {
		if key == "HTTP_ADDR" {
			return "127.0.0.1:9000"
		}
		return ""
	})

	if cfg.HTTPAddr != "127.0.0.1:9000" {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, "127.0.0.1:9000")
	}
}
