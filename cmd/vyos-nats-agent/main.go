package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/routerarchitects/olg-server-vyos-client-natagent/internal/config"
)

func main() {
	var configPath string
	var validateConfig bool
	var printEffectiveConfig bool

	flag.StringVar(&configPath, "config", "", "path to config yaml")
	flag.BoolVar(&validateConfig, "validate-config", false, "validate config and exit")
	flag.BoolVar(&printEffectiveConfig, "print-effective-config", false, "print loaded app config and converted agentcore config as JSON")
	flag.Parse()

	cfg, resolvedPath, err := config.LoadResolved(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	agentCoreCfg, err := cfg.ToAgentCoreConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to convert config to agentcore.Config: %v\n", err)
		os.Exit(1)
	}

	if printEffectiveConfig {
		if err := printJSON("resolved_config_path", resolvedPath); err != nil {
			fmt.Fprintf(os.Stderr, "failed to print resolved path: %v\n", err)
			os.Exit(1)
		}
		if err := printJSON("loaded_app_config", cfg); err != nil {
			fmt.Fprintf(os.Stderr, "failed to print loaded app config: %v\n", err)
			os.Exit(1)
		}
		if err := printJSON("converted_agentcore_config", agentCoreCfg); err != nil {
			fmt.Fprintf(os.Stderr, "failed to print converted agentcore config: %v\n", err)
			os.Exit(1)
		}
	}

	if validateConfig {
		fmt.Println("configuration valid")
		return
	}

	fmt.Println("phase 1 complete: config loader available; agent runtime not implemented yet")
}

func printJSON(label string, v any) error {
	payload, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("%s:\n%s\n", label, payload)
	return nil
}
