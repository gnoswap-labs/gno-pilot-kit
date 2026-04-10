package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// runDryTest performs the gnoswap dry-run test sequence:
//  1. Update admin address in tests/scripts/config/default.mk
//  2. Run tests/scripts/patch-admin-address.sh with the admin address
func runDryTest(cfg Config, nodeDir string) {
	if cfg.GnoswapTests == "" {
		fmt.Fprintln(os.Stderr, "error: gnoswap_tests is not set in pilot.toml")
		os.Exit(1)
	}

	fmt.Println("=== Dry-run test ===")
	fmt.Println()
	fmt.Println("  Steps:")
	fmt.Println("  1. Update ADDR_ADMIN in tests/scripts/config/default.mk")
	fmt.Println("  2. Run tests/scripts/patch-admin-address.sh")
	fmt.Println("  3. Run make faucet-admin ENV=local")
	fmt.Println("  4. Remove test packages")
	fmt.Println("  5. Deploy packages (make deploy ENV=local)")
	fmt.Println()

	// Step 1: update all .mk config files (default.mk, local.mk, etc.)
	configDir := filepath.Join(cfg.GnoswapTests, "scripts", "config")
	entries, err := os.ReadDir(configDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading config dir %s: %v\n", configDir, err)
		os.Exit(1)
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".mk") {
			updateAdminInMakefile(filepath.Join(configDir, e.Name()), cfg.AdminNewAddress)
		}
	}

	// Step 2: patch OLD_ADDRESS in the script, then run it with admin_new_address
	scriptPath := filepath.Join(cfg.GnoswapTests, "scripts", "patch-admin-address.sh")
	patchOldAddressInScript(scriptPath, cfg.AdminOldAddress)

	fmt.Printf("Running: bash scripts/patch-admin-address.sh %s\n", cfg.AdminNewAddress)
	fmt.Println()

	cmd := exec.Command("bash", "scripts/patch-admin-address.sh", cfg.AdminNewAddress)
	cmd.Dir = cfg.GnoswapTests
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Step 3: faucet admin account with ugnot
	fmt.Println("Running: make faucet-admin ENV=local")
	fmt.Println()

	faucetCmd := exec.Command("make", "faucet-admin", "ENV=local")
	faucetCmd.Dir = cfg.GnoswapTests
	faucetCmd.Stdout = os.Stdout
	faucetCmd.Stderr = os.Stderr
	if err := faucetCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Step 4: remove test packages
	fmt.Println("Removing test packages...")
	fmt.Println()

	removeCmd := exec.Command("make", "remove-test")
	removeCmd.Dir = cfg.GnoswapTests
	removeCmd.Stdout = os.Stdout
	removeCmd.Stderr = os.Stderr
	if err := removeCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Step 5: deploy packages
	fmt.Println("Deploying packages...")
	fmt.Println()

	deployCmd := exec.Command("make", "deploy", "ENV=local")
	deployCmd.Dir = cfg.GnoswapTests
	deployCmd.Stdout = os.Stdout
	deployCmd.Stderr = os.Stderr
	if err := deployCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("Dry-run test complete.")
}

// patchOldAddressInScript replaces the value of OLD_ADDRESS in patch-admin-address.sh with addr.
func patchOldAddressInScript(path, addr string) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %v\n", path, err)
		os.Exit(1)
	}

	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "OLD_ADDRESS=") {
			lines[i] = `OLD_ADDRESS="` + addr + `"`
			break
		}
	}

	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "error writing %s: %v\n", path, err)
		os.Exit(1)
	}
	fmt.Printf("  ✓ updated OLD_ADDRESS in %s\n", path)
}

// updateAdminInMakefile replaces the values of ADDR_GNOSWAP, ADDR_ADMIN, ADDR_TEST
// in the given Makefile with addr.
func updateAdminInMakefile(path, addr string) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %v\n", path, err)
		os.Exit(1)
	}

	targets := map[string]bool{
		"ADDR_ADMIN":      true,
		"ADDR_GNOSWAP":    true,
		"ADDR_TEST":       true,
		"ADDR_TEST_ADMIN": true,
	}

	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		for key := range targets {
			if strings.HasPrefix(trimmed, key) {
				parts := strings.SplitN(line, ":=", 2)
				if len(parts) == 2 {
					lines[i] = parts[0] + ":= " + addr
				}
				break
			}
		}
	}

	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing %s: %v\n", path, err)
		os.Exit(1)
	}
	fmt.Printf("  ✓ updated %s\n", path)
}
