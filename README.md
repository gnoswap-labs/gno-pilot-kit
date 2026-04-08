# gno-pilot-kit

A CLI toolkit for quickly setting up and running a local `gnoland` node, and testing [GnoSwap](https://github.com/gnoswap-labs/gnoswap) package deployments.

## Overview

`gno-pilot-kit` automates the repetitive steps involved in:

- Bootstrapping a new `gnoland` node (secrets, config, genesis)
- Starting or resetting an existing node
- Patching admin addresses and deploying GnoSwap packages to a local node

It is designed for developers who want a reproducible local environment without manually running a chain of `gnoland` CLI commands.

## Prerequisites

| Requirement | Notes |
|---|---|
| Go 1.22+ | Required to build the tool |
| `gnoland` | Must be built and available in `$PATH` |
| [Gno repository](https://github.com/gnolang/gno) | Local clone required |
| [GnoSwap repository](https://github.com/gnoswap-labs/gnoswap) | Required only for dry-run testing (option 3) |
| `make`, `bash` | Required for dry-run testing |

### Build `gnoland`

```sh
cd /path/to/gno
make install
```

## Installation

```sh
git clone https://github.com/aronpark/gno-pilot-kit
cd gno-pilot-kit
go build -o gno-pilot-kit .
```

## Configuration

Copy the example config and fill in your local paths:

```sh
cp pilot.toml.example pilot.toml
```

`pilot.toml` is gitignored — each developer maintains their own copy.

### `pilot.toml` reference

```toml
# Path to the gno repository root (where go.work is located)
gno_root = "/path/to/gno"

# Path to the gnoswap tests directory (required for dry-run option)
gnoswap_tests = "/path/to/gnoswap/tests"

# Admin address patching (required for dry-run option)
admin_old_address = "g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5"  # address currently in GnoSwap source
admin_new_address = "g18xk5cn802htdg4zqxy5y6lcjdtcgxpyzh85ce3"  # your local wallet address

# Persistent peers — add one [[peers]] block per validator
# [[peers]]
# id      = "nodeID@ip:port"   # P2P connection string
# pub_key = ""                 # validator public key (added to genesis)
# address = ""                 # wallet address (added to genesis balances)
```


## Usage

```sh
./gno-pilot-kit
```

The tool displays an interactive menu:

```
=== gno-pilot-kit ===

Project : /path/to/gno-pilot-kit
GnoRoot : /path/to/gno

Node directory name [node]:

1) Start node    (setup if not initialized, start if already set up)
2) Reset node    (delete db & reset validator state to block 1)
3) Dry-run test  (update admin address & run gnoswap patch script)

Choice [1]:
```

---

### Option 1 — Start node

**First run (node not initialized):** Runs a 6-step setup wizard:

```
[1/6] Generating secrets...
       → gnoland secrets init

[2/6] Config setup...
       → gnoland config init
       → Prompts for P2P port (default: 26656) and RPC port (default: 26657)

[3/6] Validator info:
       → Reads priv_validator_key.json and displays address and public key

[4/6] Genesis setup...
       → Prompts for chain ID (default: dev) and node name
       → Loads genesis.json from project root (if present) or uses a minimal template
       → Writes node/config/genesis.json with this node as a validator

[5/6] Persistent peers setup...
       → Configures p2p.persistent_peers from pilot.toml [[peers]] entries

[6/6] Setup complete!
       → Prompts to start the node immediately
```

**Subsequent runs (node already initialized):** Skips setup and starts the node directly.

The node starts with:
```sh
gnoland start -data-dir <nodeDir> -genesis <genesisPath> -skip-failing-genesis-txs
```

#### Using a custom genesis

Place a `genesis.json` file in the project root before running setup. The tool will load it as the base genesis template and merge in the validator information. This is useful for seeding the chain with pre-deployed packages or specific balances.

---

### Option 2 — Reset node

Wipes all block data so the node can restart from block 1, while preserving the node's identity (validator keys and config):

- Deletes `<nodeDir>/db/`
- Deletes `<nodeDir>/wal/`
- Resets `<nodeDir>/secrets/priv_validator_state.json` to height 0

A confirmation prompt is shown before any deletion.

---

### Option 3 — Dry-run test (GnoSwap deployment)

Requires a running node (start it first with option 1). Automates the GnoSwap package deployment workflow in 5 steps:

```
1. Update ADDR_ADMIN in tests/scripts/config/default.mk
2. Patch OLD_ADDRESS in tests/scripts/patch-admin-address.sh → run it with admin_new_address
3. make faucet-admin ENV=local     (fund admin account with ugnot)
4. make remove-test                (remove previously deployed test packages)
5. make deploy ENV=local           (deploy GnoSwap packages)
```

Configure `admin_old_address` and `admin_new_address` in `pilot.toml` before running this option.

---

## Multi-validator setup

To connect multiple nodes in a local validator set:

1. Run option 1 on each machine to generate validator keys.
2. Collect each node's `address` and `pub_key` (printed in step [3/6]).
3. Add peer entries to `pilot.toml` on each machine:

```toml
[[peers]]
id      = "abc123...@192.168.1.10:26656"
pub_key = "<base64-pubkey>"
address = "g1..."

[[peers]]
id      = "def456...@192.168.1.11:26656"
pub_key = "<base64-pubkey>"
address = "g1..."
```

4. Re-run setup — peers will be added to `p2p.persistent_peers` and their keys will be included in genesis validators.

---

## Node directory layout

After setup, the node directory (`node/` by default) contains:

```
node/
├── config/
│   ├── config.toml          # Tendermint configuration
│   └── genesis.json         # Chain genesis file
├── secrets/
│   ├── priv_validator_key.json    # Validator signing key
│   ├── priv_validator_state.json  # Validator consensus state
│   └── node_key.json              # P2P node identity key
├── db/                      # LevelDB block databases (created on first start)
└── wal/                     # Write-ahead logs (created on first start)
```

The `node*/` directories are gitignored.