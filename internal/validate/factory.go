package validate

import (
	"sync"

	"github.com/hrodrig/kzero/internal/executor"
)

var (
	defaultClientFactoryMu sync.RWMutex
	defaultClientFactory   ClientFactory = executor.NewKubernetesClient
)

// ClientFactoryDefault returns the process-wide client factory (thread-safe).
func ClientFactoryDefault() ClientFactory {
	defaultClientFactoryMu.RLock()
	defer defaultClientFactoryMu.RUnlock()
	return defaultClientFactory
}

// SwapDefaultClientFactory replaces the default factory (tests). Returns the previous value for restore.
func SwapDefaultClientFactory(f ClientFactory) ClientFactory {
	defaultClientFactoryMu.Lock()
	defer defaultClientFactoryMu.Unlock()
	old := defaultClientFactory
	defaultClientFactory = f
	return old
}
