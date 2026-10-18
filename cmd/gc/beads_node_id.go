package main

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/gastownhall/gascity/internal/config"
	"github.com/gastownhall/gascity/internal/fsys"
)

// beadsNodeIDEnv is the Beads replica identity. bd reads it (config node_id,
// env BEADS_NODE_ID) when it grants a lease and when `bd reclaim` decides
// whether a stale lease was granted by this replica; with no id, every lease
// is reclaimable from anywhere. gc sets it to the city's workspace name so
// every bd subprocess and every agent session of this factory grants and
// reclaims under one id, scoped to the factory even if a store is ever read
// from another machine (PROOF ADR-0239, decision 4).
//
// The native store (internal/beads, the library binding) cannot take the
// value: beads v1.3.1 reads node_id only through its CLI config
// initialisation, and the library's WithNodeID lives in an internal package.
// Leases the native store grants therefore carry no granted_node until the
// library exposes a setter; see nativeDoltOpenEnvKeys.
const beadsNodeIDEnv = "BEADS_NODE_ID"

// cityNameEnv carries the same workspace name to everything a city starts.
// gc already gives pack commands GC_CITY_NAME (runDiscoveredCommand); orders,
// bd subprocesses, hooks and agent sessions start from the city runtime map
// instead and used to inherit whatever the supervisor's environment held. One
// supervisor serving two cities then handed both the same name, and a pack
// that builds an identity from it (the PROOF claim seat
// gascity/<factory>/<rig>) took the other city's identity. Projecting it per
// city makes the supervisor-wide value irrelevant (PROOF ADR-0239, M3).
const cityNameEnv = "GC_CITY_NAME"

var beadsNodeIDs sync.Map // cityPath -> node id

// beadsNodeIDForCity returns the Beads node id for cityPath: the city's
// workspace name as the site binding declares it (.gc/site.toml
// workspace_name), else the city directory's name — the fallback
// config.EffectiveCityName uses when nothing is declared. The value is read
// once per process and city; a change to the site binding is a restart.
func beadsNodeIDForCity(cityPath string) string {
	cityPath = strings.TrimSpace(cityPath)
	if cityPath == "" {
		return ""
	}
	if cached, ok := beadsNodeIDs.Load(cityPath); ok {
		return cached.(string)
	}
	id := readBeadsNodeID(fsys.OSFS{}, cityPath)
	beadsNodeIDs.Store(cityPath, id)
	return id
}

func readBeadsNodeID(fs fsys.FS, cityPath string) string {
	fallback := filepath.Base(filepath.Clean(cityPath))
	binding, err := config.LoadSiteBinding(fs, cityPath)
	if err != nil || binding == nil {
		return fallback
	}
	if name := strings.TrimSpace(binding.WorkspaceName); name != "" {
		return name
	}
	return fallback
}

// withBeadsNodeID adds the city's node id to a runtime env map when one is
// known and the map does not already carry an explicit value.
func withBeadsNodeID(env map[string]string, cityPath string) map[string]string {
	if env == nil {
		return env
	}
	if _, set := env[beadsNodeIDEnv]; set {
		return env
	}
	if id := beadsNodeIDForCity(cityPath); id != "" {
		env[beadsNodeIDEnv] = id
	}
	return env
}

// withCityName adds the city's workspace name as GC_CITY_NAME — the same value
// as the Beads node id — when one is known and the map does not already carry
// an explicit value. An ambient GC_CITY_NAME is not an explicit value: the map
// is built from the city alone, so the city's own name always reaches it.
func withCityName(env map[string]string, cityPath string) map[string]string {
	if env == nil {
		return env
	}
	if _, set := env[cityNameEnv]; set {
		return env
	}
	if name := beadsNodeIDForCity(cityPath); name != "" {
		env[cityNameEnv] = name
	}
	return env
}
