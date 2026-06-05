package probe

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hrodrig/kzero/internal/config"
)

type cacheEntry struct {
	OKAt time.Time `json:"ok_at"`
	Key  string    `json:"key"`
}

// CacheKey fingerprints the probe pipeline and checks for a cluster.
func CacheKey(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}
	var parts []string
	parts = append(parts, cfg.Cluster.Name)
	for _, s := range cfg.InfraProbe.Pipeline.Up {
		parts = append(parts, "up:"+s.Ref)
	}
	for _, s := range cfg.InfraProbe.Pipeline.Down {
		parts = append(parts, "down:"+s.Ref)
	}
	for _, c := range cfg.InfraProbe.Checks {
		if c.PVCBound != "" {
			parts = append(parts, "pvc:"+c.PVCBound)
		}
		if c.ReleaseReady {
			parts = append(parts, "release_ready")
		}
	}
	return strings.Join(parts, "|")
}

func cachePath(cfg *config.Config) (string, error) {
	dir, err := cacheDir(cfg)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("infra probe cache dir: %w", err)
	}
	return filepath.Join(dir, "probe-cache.json"), nil
}

func cacheDir(cfg *config.Config) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("infra probe cache: no config")
	}
	if dir := strings.TrimSpace(cfg.Run.ProbeCacheDir); dir != "" {
		return dir, nil
	}
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "kzero", "probe"), nil
}

// IsFresh reports whether a successful probe is still within cache_ttl.
func IsFresh(cfg *config.Config) (bool, error) {
	if cfg == nil || cfg.InfraProbe.CacheTTL <= 0 {
		return false, nil
	}
	path, err := cachePath(cfg)
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return false, nil
	}
	key := CacheKey(cfg)
	if entry.Key != key {
		return false, nil
	}
	return time.Since(entry.OKAt) < cfg.InfraProbe.CacheTTL, nil
}

// WriteOK stores a successful probe timestamp.
func WriteOK(cfg *config.Config) error {
	if cfg == nil || cfg.InfraProbe.CacheTTL <= 0 {
		return nil
	}
	path, err := cachePath(cfg)
	if err != nil {
		return err
	}
	entry := cacheEntry{OKAt: time.Now().UTC(), Key: CacheKey(cfg)}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
