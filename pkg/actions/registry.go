package actions

import (
	"fmt"
	"sort"
	"sync"
)

var (
	mu                sync.RWMutex
	catalog           = make(map[string]Action)
	runtimeTargetType string
)

// SetTargetType configures which ZOA target type (rc or mc) this process serves.
// Must be called once at startup from ZOA_TARGET_TYPE.
// When empty, all registered TAs remain visible (used in unit tests).
func SetTargetType(target string) {
	mu.Lock()
	defer mu.Unlock()
	runtimeTargetType = target
}

// RuntimeTargetType returns the target type (rc or mc) configured at process startup.
func RuntimeTargetType() string {
	mu.RLock()
	defer mu.RUnlock()
	return runtimeTargetType
}

func Register(a Action) {
	mu.Lock()
	defer mu.Unlock()

	name := a.Metadata().Name
	if _, exists := catalog[name]; exists {
		panic(fmt.Sprintf("action %q already registered", name))
	}
	catalog[name] = a
}

func allowedOnRuntimeTarget(meta ActionMetadata) bool {
	target := runtimeTargetType
	if target == "" {
		return true
	}
	for _, t := range meta.TargetTypes {
		if t == target {
			return true
		}
	}
	return false
}

// Get returns a registered action visible on this target type.
func Get(name string) (Action, bool) {
	mu.RLock()
	defer mu.RUnlock()

	a, ok := catalog[name]
	if !ok || !allowedOnRuntimeTarget(a.Metadata()) {
		return nil, false
	}
	return a, true
}

// GetCatalog returns an action from the full catalog, ignoring target type filtering.
func GetCatalog(name string) (Action, bool) {
	mu.RLock()
	defer mu.RUnlock()

	a, ok := catalog[name]
	return a, ok
}

// List returns registered actions visible on this target type.
func List() []Action {
	mu.RLock()
	defer mu.RUnlock()

	out := make([]Action, 0, len(catalog))
	for _, a := range catalog {
		if allowedOnRuntimeTarget(a.Metadata()) {
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Metadata().Name < out[j].Metadata().Name
	})
	return out
}

// ListCatalog returns all registered actions regardless of target type.
func ListCatalog() []Action {
	mu.RLock()
	defer mu.RUnlock()

	out := make([]Action, 0, len(catalog))
	for _, a := range catalog {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Metadata().Name < out[j].Metadata().Name
	})
	return out
}
