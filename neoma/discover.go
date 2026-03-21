package neoma

import (
	"sync"

	"github.com/MeGaNeKoS/neoma/core"
)

var (
	discoveredMu     sync.RWMutex
	discoveredErrors = map[string][]core.DiscoveredError{}
)

// RegisterDiscoveredErrors merges the given error mappings into the global
// discovered errors registry, keyed by handler or operation name.
func RegisterDiscoveredErrors(errors map[string][]core.DiscoveredError) {
	discoveredMu.Lock()
	defer discoveredMu.Unlock()
	for k, v := range errors {
		discoveredErrors[k] = v
	}
}

func getDiscoveredErrors(handlerName string) []core.DiscoveredError {
	discoveredMu.RLock()
	defer discoveredMu.RUnlock()
	return discoveredErrors[handlerName]
}
