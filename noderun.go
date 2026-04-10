package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type PrivValidatorKey struct {
	Address string `json:"address"`
	PubKey  struct {
		Type  string `json:"@type"`
		Value string `json:"value"`
	} `json:"pub_key"`
}

func readValidatorKey(secretsDir string) PrivValidatorKey {
	data, err := os.ReadFile(filepath.Join(secretsDir, "priv_validator_key.json"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading validator key: %v\n", err)
		os.Exit(1)
	}
	var key PrivValidatorKey
	if err := json.Unmarshal(data, &key); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing validator key: %v\n", err)
		os.Exit(1)
	}
	return key
}

// loadRootGenesis loads genesis.json from the project root if it exists,
// otherwise falls back to baseGenesis.
func loadRootGenesis() map[string]any {
	data, err := os.ReadFile("genesis.json")
	if err != nil {
		fmt.Println("  genesis.json not found in project root — using minimal template")
		return baseGenesis()
	}
	var genesis map[string]any
	if err := json.Unmarshal(data, &genesis); err != nil {
		fmt.Fprintf(os.Stderr, "  warning: failed to parse genesis.json: %v — using minimal template\n", err)
		return baseGenesis()
	}
	fmt.Println("  ✓ loaded genesis.json from project root")
	return genesis
}

// baseGenesis returns a minimal but valid genesis template with all required params.
func baseGenesis() map[string]any {
	return map[string]any{
		"genesis_time": "2024-01-01T00:00:00Z",
		"chain_id":     "dev",
		"consensus_params": map[string]any{
			"Block": map[string]string{
				"MaxTxBytes":    "1000000",
				"MaxDataBytes":  "2000000",
				"MaxBlockBytes": "0",
				"MaxGas":        "3000000000",
				"TimeIotaMS":    "100",
			},
		},
		"validators": []any{},
		"app_state": map[string]any{
			"@type":    "/gno.GenesisState",
			"balances": []any{},
			"txs":      []any{},
			"auth": map[string]any{
				"params": map[string]any{
					"max_memo_bytes":              "65536",
					"tx_sig_limit":                "7",
					"tx_size_cost_per_byte":       "10",
					"sig_verify_cost_ed25519":     "590",
					"sig_verify_cost_secp256k1":   "1000",
					"gas_price_change_compressor": "10",
					"target_gas_ratio":            "70",
					"initial_gasprice": map[string]any{
						"gas":   "1000",
						"price": "1ugnot",
					},
					"unrestricted_addrs": nil,
					"fee_collector":      "g17xpfvakm2amg962yls6f84z3kell8c5lr9lr2e",
				},
			},
			"bank": map[string]any{
				"params": map[string]any{
					"restricted_denoms": []any{},
				},
			},
			"vm": map[string]any{
				"params": map[string]any{
					"sysnames_pkgpath":      "gno.land/r/sys/names",
					"syscla_pkgpath":        "gno.land/r/sys/cla",
					"chain_domain":          "gno.land",
					"default_deposit":       "600000000ugnot",
					"storage_price":         "100ugnot",
					"storage_fee_collector": "g1c9stkafpvcwez2efq3qtfuezw4zpaux3tvxggk",
				},
				"realm_params": nil,
			},
		},
	}
}

func writeGenesis(path string, content map[string]any) {
	data, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func startNode(nodeDir, genesisPath string) {
	fmt.Printf("Running: gnoland start -data-dir %s -genesis %s -skip-failing-genesis-txs\n", nodeDir, genesisPath)
	fmt.Println()
	mustRun("gnoland", "start", "-data-dir", nodeDir, "-genesis", genesisPath, "-skip-failing-genesis-txs")
}

func resetNode(nodeDir string) {
	fmt.Println("=== Reset node ===")
	fmt.Println()
	fmt.Printf("  node dir: %s\n", nodeDir)
	fmt.Println()

	confirm := prompt("Reset? This will delete the entire node directory. (y/n)", "n")
	if confirm != "y" {
		fmt.Println("Cancelled.")
		return
	}

	if err := os.RemoveAll(nodeDir); err != nil {
		fmt.Fprintf(os.Stderr, "error removing node dir: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  ✓ %s removed\n", nodeDir)
	fmt.Println()
	fmt.Println("Node reset complete. Run option 1 to reinitialize.")
}

func setupNode(nodeDir string, cfg Config) {
	secretsDir := filepath.Join(nodeDir, "secrets")
	configDir := filepath.Join(nodeDir, "config")
	configPath := filepath.Join(configDir, "config.toml")
	genesisPath := filepath.Join(configDir, "genesis.json")

	// ── Step 1: Generate secrets ──────────────────────────────────
	fmt.Println("[1/6] Generating secrets...")
	if err := os.MkdirAll(secretsDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	mustRun("gnoland", "secrets", "init", "-data-dir", secretsDir)
	fmt.Println()

	// ── Step 2: Initialize config ─────────────────────────────────
	fmt.Println("[2/6] Config setup...")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	mustRun("gnoland", "config", "init", "-config-path", configPath)

	p2pPort := prompt("P2P port", "26656")
	rpcPort := prompt("RPC port", "26657")

	mustRun("gnoland", "config", "set", "p2p.laddr", fmt.Sprintf("tcp://0.0.0.0:%s", p2pPort), "-config-path", configPath)
	mustRun("gnoland", "config", "set", "rpc.laddr", fmt.Sprintf("tcp://0.0.0.0:%s", rpcPort), "-config-path", configPath)
	mustRun("gnoland", "config", "set", "consensus.priv_validator.local_signer", "priv_validator_key.json", "-config-path", configPath)
	mustRun("gnoland", "config", "set", "consensus.priv_validator.sign_state", "priv_validator_state.json", "-config-path", configPath)
	fmt.Println()

	// ── Step 3: Read validator key from secrets ──────────────────
	fmt.Println("[3/6] Validator info:")
	myKey := readValidatorKey(secretsDir)
	fmt.Printf("  address : %s\n", myKey.Address)
	fmt.Printf("  pub_key : %s\n", myKey.PubKey.Value)
	fmt.Println()

	// ── Step 4: Write genesis ─────────────────────────────────────
	fmt.Println("[4/6] Genesis setup...")

	chainID := prompt("Chain ID", "dev")
	myName := prompt("This node name", "testnode")

	genesis := loadRootGenesis()
	genesis["chain_id"] = chainID
	genesis["genesis_time"] = "2024-01-01T00:00:00Z"
	validators := []map[string]any{
		{
			"address": myKey.Address,
			"pub_key": map[string]string{
				"@type": "/tm.PubKeyEd25519",
				"value": myKey.PubKey.Value,
			},
			"power": "10",
			"name":  myName,
		},
	}
	for i, p := range cfg.Peers {
		if p.PubKey != "" && p.Address != "" {
			validators = append(validators, map[string]any{
				"address": p.Address,
				"pub_key": map[string]string{
					"@type": "/tm.PubKeyEd25519",
					"value": p.PubKey,
				},
				"power": "10",
				"name":  fmt.Sprintf("peer-%d", i+1),
			})
			fmt.Printf("  ✓ added peer validator: %s\n", p.Address)
		}
	}
	genesis["validators"] = validators

	writeGenesis(genesisPath, genesis)
	fmt.Printf("  ✓ genesis saved: %s\n", genesisPath)
	fmt.Printf("  → To add other validators, edit the file directly:\n")
	fmt.Printf("    %s\n", genesisPath)
	fmt.Println()

	// ── Step 5: Persistent peers ──────────────────────────────────
	fmt.Println("[5/6] Persistent peers setup...")
	if len(cfg.Peers) > 0 {
		peerIDs := make([]string, 0, len(cfg.Peers))
		for _, p := range cfg.Peers {
			if p.ID != "" {
				peerIDs = append(peerIDs, p.ID)
			}
		}
		if len(peerIDs) > 0 {
			peersVal := strings.Join(peerIDs, ",")
			mustRun("gnoland", "config", "set", "p2p.persistent_peers", peersVal, "-config-path", configPath)
			fmt.Printf("  ✓ persistent_peers: %s\n", peersVal)
		}
	} else {
		fmt.Println("  (no peers configured in pilot.toml — skipping)")
	}
	fmt.Println()

	// ── Step 6: Done and start node ───────────────────────────────
	fmt.Println("[6/6] Setup complete!")
	fmt.Println()

	ans := prompt("Start node now? (y/n)", "y")
	if ans != "n" {
		fmt.Println()
		startNode(nodeDir, genesisPath)
	} else {
		fmt.Println()
		fmt.Println("Start node manually:")
		fmt.Printf("  gnoland start -data-dir %s -genesis %s -skip-failing-genesis-txs\n", nodeDir, genesisPath)
	}
}
