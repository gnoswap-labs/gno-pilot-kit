package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var reader = bufio.NewReader(os.Stdin)

func prompt(label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("%s: ", label)
	}
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

func run(args ...string) error {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func mustRun(args ...string) {
	if err := run(args...); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	fmt.Println("=== gno-pilot-kit ===")
	fmt.Println()

	cfg := loadConfig()
	gnoRoot := resolveGnoRoot(cfg)
	projectRoot, _ := filepath.Abs(".")

	fmt.Printf("Project : %s\n", projectRoot)
	fmt.Printf("GnoRoot : %s\n", gnoRoot)

	nodeName := prompt("Node directory name", "node")
	nodeDir := filepath.Join(projectRoot, nodeName)
	fmt.Printf("Path    : %s\n", nodeDir)
	fmt.Println()

	fmt.Println("1) Start node    (setup if not initialized, start if already set up)")
	fmt.Println("2) Reset node    (delete db & reset validator state to block 1)")
	fmt.Println("3) Dry-run test  (update admin address & run gnoswap patch script)")
	fmt.Println()
	choice := prompt("Choice", "1")
	fmt.Println()

	genesisPath := filepath.Join(nodeDir, "config", "genesis.json")

	switch choice {
	case "2":
		resetNode(nodeDir)
	case "3":
		fmt.Println("Note: option 3 requires the node to be running. To start the node, select option 1.")
		fmt.Println()
		runDryTest(cfg, nodeDir)
	default:
		if _, err := os.Stat(genesisPath); err == nil {
			fmt.Println("Existing node detected. Starting directly...")
			fmt.Println()
			startNode(nodeDir, genesisPath)
		} else {
			fmt.Println("No existing node found. Running setup...")
			fmt.Println()
			setupNode(nodeDir, cfg)
		}
	}
}
