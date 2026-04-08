package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

const configFileName = "pilot.toml"

// Peer holds information about a validator peer node.
type Peer struct {
	ID      string `toml:"id"`      // node ID for P2P (nodeID@ip:port)
	PubKey  string `toml:"pub_key"` // validator public key (for genesis)
	Address string `toml:"address"` // wallet address (for genesis balances)
}

// Config holds the pilot.toml configuration.
type Config struct {
	GnoRoot         string `toml:"gno_root"`
	GnoswapTests    string `toml:"gnoswap_tests"`
	AdminOldAddress string `toml:"admin_old_address"`
	AdminNewAddress string `toml:"admin_new_address"`
	RPCUrl          string `toml:"rpc_url"`
	ChainID         string `toml:"chain_id"`
	Peers           []Peer `toml:"peers"`
}

func (c Config) rpcUrl() string {
	if c.RPCUrl != "" {
		return c.RPCUrl
	}
	return "localhost:26657"
}

func (c Config) chainID() string {
	if c.ChainID != "" {
		return c.ChainID
	}
	return "dev"
}

// loadConfig reads pilot.toml from the current directory.
// If pilot.toml does not exist, it is created from pilot.toml.example and the user is notified.
// Missing fields fall back to environment variables, then empty string.
func loadConfig() Config {
	var cfg Config

	data, err := os.ReadFile(configFileName)
	if err != nil {
		// pilot.toml not found — create it from the example
		createDefaultConfig()
	} else {
		if _, err := toml.Decode(string(data), &cfg); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to parse %s: %v\n", configFileName, err)
		}
	}

	// Environment variables override config file values
	if root := strings.TrimSpace(os.Getenv("GNOROOT")); root != "" {
		cfg.GnoRoot = root
	}
	if root := strings.TrimSpace(os.Getenv("GNOSWAP_TESTS")); root != "" {
		cfg.GnoswapTests = root
	}

	return cfg
}

// createDefaultConfig copies pilot.toml.example to pilot.toml and prompts the user to edit it.
func createDefaultConfig() {
	example, err := os.ReadFile("pilot.toml.example")
	if err != nil {
		// No example file either — write a minimal template
		example = []byte("# gno-pilot-kit config\ngno_root = \"\"\ngnoswap_tests = \"\"\n")
	}

	if err := os.WriteFile(configFileName, example, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error: could not create %s: %v\n", configFileName, err)
		os.Exit(1)
	}

	fmt.Println("-----------------------------------------------------")
	fmt.Printf("  pilot.toml not found — created from pilot.toml.example\n")
	fmt.Printf("  Please edit pilot.toml with your local paths and re-run.\n")
	fmt.Println("-----------------------------------------------------")
	fmt.Println()
	os.Exit(0)
}

// resolveGnoRoot returns the gno root from config, GNOROOT env, go.work search, or cwd.
func resolveGnoRoot(cfg Config) string {
	if cfg.GnoRoot != "" {
		return cfg.GnoRoot
	}
	if root := strings.TrimSpace(os.Getenv("GNOROOT")); root != "" {
		return root
	}

	// Walk up looking for go.work
	dir, err := filepath.Abs(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	cwd, _ := filepath.Abs(".")
	fmt.Fprintln(os.Stderr, "warning: could not find gno root. Set gno_root in pilot.toml or the GNOROOT env var.")
	return cwd
}
