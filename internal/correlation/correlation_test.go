package correlation

import (
	"testing"

	"github.com/hrodrig/kzero/internal/config"
)

func TestClientID_trimAndEmpty(t *testing.T) {
	t.Parallel()
	if got := ClientID(nil); got != "" {
		t.Fatalf("nil cfg: got %q", got)
	}
	cfg := &config.Config{Client: config.ClientConfig{ID: "  pilot  "}}
	if got := ClientID(cfg); got != "pilot" {
		t.Fatalf("got %q", got)
	}
}

func TestAppendEnv(t *testing.T) {
	t.Parallel()
	base := []string{"HOME=/tmp"}
	got := AppendEnv(&config.Config{}, base)
	if got[0] != "HOME=/tmp" {
		t.Fatalf("got %v", got)
	}
	got = AppendEnv(&config.Config{Client: config.ClientConfig{ID: "cbpi-dev"}}, base)
	var hasClient bool
	for _, e := range got {
		if e == "KZERO_CLIENT_ID=cbpi-dev" {
			hasClient = true
		}
	}
	if !hasClient {
		t.Fatalf("missing KZERO_CLIENT_ID in %v", got)
	}
}

func TestLogPrefix(t *testing.T) {
	t.Parallel()
	if LogPrefix(&config.Config{}) != "" {
		t.Fatal("expected empty prefix")
	}
	if got := LogPrefix(&config.Config{Client: config.ClientConfig{ID: "a"}}); got != "client_id=a " {
		t.Fatalf("got %q", got)
	}
	if got := LogPrefix(&config.Config{Client: config.ClientConfig{ID: "a b"}}); got != `client_id="a b" ` {
		t.Fatalf("got %q", got)
	}
}
