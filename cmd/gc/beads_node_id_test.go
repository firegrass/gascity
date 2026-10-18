package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBeadsNodeIDFollowsTheSiteBindingThenTheDirectoryName(t *testing.T) {
	root := t.TempDir()
	city := filepath.Join(root, "proofcity")
	if err := os.MkdirAll(filepath.Join(city, ".gc"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := beadsNodeIDForCity(city); got != "proofcity" {
		t.Fatalf("no site binding: node id = %q, want the directory name", got)
	}
	bound := filepath.Join(root, "bound")
	if err := os.MkdirAll(filepath.Join(bound, ".gc"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bound, ".gc", "site.toml"), []byte("workspace_name = \"p620\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := beadsNodeIDForCity(bound); got != "p620" {
		t.Fatalf("site binding: node id = %q, want p620", got)
	}
	if got := beadsNodeIDForCity(""); got != "" {
		t.Fatalf("empty city path: node id = %q, want empty", got)
	}
}

func TestCityRuntimeEnvCarriesTheBeadsNodeID(t *testing.T) {
	city := filepath.Join(t.TempDir(), "factory-a")
	if err := os.MkdirAll(filepath.Join(city, ".gc"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(city, ".gc", "site.toml"), []byte("workspace_name = \"factory-a\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := cityRuntimeEnvMapForCity(city)
	if env[beadsNodeIDEnv] != "factory-a" {
		t.Fatalf("BEADS_NODE_ID = %q, want factory-a; env=%v", env[beadsNodeIDEnv], env)
	}
	if env["GC_CITY"] != city || env["GC_CITY_PATH"] != city {
		t.Fatalf("identity anchors changed: %v", env)
	}
	explicit := withBeadsNodeID(map[string]string{beadsNodeIDEnv: "pinned"}, city)
	if explicit[beadsNodeIDEnv] != "pinned" {
		t.Fatalf("an explicit value must win: %v", explicit)
	}
}

func TestCityRuntimeEnvCarriesTheCityNameOverAnAmbientOne(t *testing.T) {
	// One supervisor, two cities: the supervisor's environment says "proof",
	// this city is "p620". Everything this city starts must say p620.
	t.Setenv(cityNameEnv, "proof")
	city := filepath.Join(t.TempDir(), "city")
	if err := os.MkdirAll(filepath.Join(city, ".gc"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(city, ".gc", "site.toml"), []byte("workspace_name = \"p620\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := cityRuntimeEnvMapForCity(city)
	if env[cityNameEnv] != "p620" {
		t.Fatalf("GC_CITY_NAME = %q, want p620; env=%v", env[cityNameEnv], env)
	}
	if env[beadsNodeIDEnv] != env[cityNameEnv] {
		t.Fatalf("node id %q and city name %q must agree", env[beadsNodeIDEnv], env[cityNameEnv])
	}
	merged := mergeOrderExecEnv([]string{cityNameEnv + "=proof", "PATH=/bin"}, []string{cityNameEnv + "=" + env[cityNameEnv]})
	count := 0
	for _, entry := range merged {
		if entry == cityNameEnv+"=proof" {
			t.Fatalf("an order kept the ambient city name: %v", merged)
		}
		if entry == cityNameEnv+"=p620" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("an order must see exactly one GC_CITY_NAME=p620, got %d: %v", count, merged)
	}
	explicit := withCityName(map[string]string{cityNameEnv: "pinned"}, city)
	if explicit[cityNameEnv] != "pinned" {
		t.Fatalf("an explicit value must win: %v", explicit)
	}
}
