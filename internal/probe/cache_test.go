package probe

import (
	"testing"
	"time"

	"github.com/hrodrig/kzero/internal/config"
)

func TestCacheKey_stable(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Cluster: config.ClusterConfig{Name: "c1"},
		InfraProbe: config.InfraProbeConfig{
			Pipeline: config.ProbePipelineConfig{
				Up: []config.PipelineStep{{Ref: "release.ns/a"}},
			},
			Checks: []config.ProbeCheck{{PVCBound: "ns/pvc"}},
		},
	}
	k1 := CacheKey(cfg)
	k2 := CacheKey(cfg)
	if k1 != k2 || k1 == "" {
		t.Fatalf("unexpected key %q", k1)
	}
}

func TestIsFresh_andWriteOK(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := &config.Config{
		Cluster: config.ClusterConfig{Name: "c1"},
		Run:     config.RunConfig{ProbeCacheDir: dir},
		InfraProbe: config.InfraProbeConfig{
			CacheTTL: 30 * time.Minute,
			Pipeline: config.ProbePipelineConfig{
				Up: []config.PipelineStep{{Ref: "release.ns/a"}},
			},
		},
	}
	fresh, err := IsFresh(cfg)
	if err != nil || fresh {
		t.Fatalf("expected not fresh initially, fresh=%v err=%v", fresh, err)
	}
	if err := WriteOK(cfg); err != nil {
		t.Fatalf("WriteOK: %v", err)
	}
	fresh, err = IsFresh(cfg)
	if err != nil || !fresh {
		t.Fatalf("expected fresh after write, fresh=%v err=%v", fresh, err)
	}
	cfg.InfraProbe.Pipeline.Up[0].Ref = "changed"
	fresh, err = IsFresh(cfg)
	if err != nil || fresh {
		t.Fatalf("key change should invalidate cache, fresh=%v err=%v", fresh, err)
	}
}
