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
	if got := AppendEnv(&config.Config{}, base); len(got) != 1 {
		t.Fatalf("empty client: got %v", got)
	}
	got := AppendEnv(&config.Config{Client: config.ClientConfig{ID: "cbpi-dev"}}, base)
	if len(got) != 2 || got[1] != "KZERO_CLIENT_ID=cbpi-dev" {
		t.Fatalf("got %v", got)
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
