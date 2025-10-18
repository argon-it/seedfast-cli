// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

package manifest

import "sync"

var (
	// Global singleton cache for the manifest.
	// Lives only in process memory and is cleared when CLI exits.
	globalCache     *Manifest
	globalCacheLock sync.RWMutex
)

// GetCached returns the cached manifest from RAM, or nil if not cached.
func GetCached() *Manifest {
	globalCacheLock.RLock()
	defer globalCacheLock.RUnlock()
	return globalCache
}

// SetCached stores the manifest in RAM.
func SetCached(m *Manifest) {
	globalCacheLock.Lock()
	defer globalCacheLock.Unlock()
	globalCache = m
}

// ClearCache removes the manifest from RAM (primarily for testing).
func ClearCache() {
	globalCacheLock.Lock()
	defer globalCacheLock.Unlock()
	globalCache = nil
}
