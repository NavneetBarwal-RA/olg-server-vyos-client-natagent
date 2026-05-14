package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/routerarchitects/olg-server-vyos-client-natagent/internal/config"
)

func main() {
	var configPath string
	var validateConfig bool
	var printEffectiveConfig bool

	configureUsage()

	flag.StringVar(&configPath, "config", "", "Path to YAML config file")
	flag.BoolVar(&validateConfig, "validate-config", false, "Validate config and exit")
	flag.BoolVar(&printEffectiveConfig, "print-effective-config", false, "Print sanitized effective config and continue")

	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			flag.Usage()
			return
		}
		fmt.Fprintf(os.Stderr, "failed to parse flags: %v\n", err)
		os.Exit(2)
	}

	cfg, _, err := config.LoadResolved(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	if _, err := cfg.ToAgentCoreConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to convert config to agentcore.Config: %v\n", err)
		os.Exit(1)
	}

	if printEffectiveConfig {
		payload, err := config.MarshalRedactedYAML(*cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal effective config yaml: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("effective config:")
		fmt.Print(string(payload))
	}

	if validateConfig {
		fmt.Println("configuration valid")
		return
	}

	fmt.Println("phase 1 complete: config loader available; agent runtime not implemented yet")
}

func configureUsage() {
	flag.CommandLine.SetOutput(io.Discard)
	flag.Usage = func() {
		fmt.Fprintln(os.Stdout, "Usage:")
		fmt.Fprintln(os.Stdout, "  vyos-nats-agent [options]")
		fmt.Fprintln(os.Stdout, "")
		fmt.Fprintln(os.Stdout, "Options:")
		fmt.Fprintln(os.Stdout, "  --config <path>             Path to YAML config file")
		fmt.Fprintln(os.Stdout, "  --validate-config           Validate config and exit")
		fmt.Fprintln(os.Stdout, "  --print-effective-config    Print sanitized effective config and continue")
		fmt.Fprintln(os.Stdout, "  --help                      Show help")
		fmt.Fprintln(os.Stdout, "")
		fmt.Fprintln(os.Stdout, "Config path resolution:")
		fmt.Fprintln(os.Stdout, "  1. --config")
		fmt.Fprintln(os.Stdout, "  2. VYOS_NATS_AGENT_CONFIG")
		fmt.Fprintln(os.Stdout, "  3. /etc/vyos-nats-agent/config.yaml")
		fmt.Fprintln(os.Stdout, "")
		fmt.Fprintln(os.Stdout, "Phase 1 behavior:")
		fmt.Fprintln(os.Stdout, "  This binary only loads, validates, prints, and converts configuration.")
		fmt.Fprintln(os.Stdout, "  It does not connect to NATS or start the agent runtime yet.")
	}
}
