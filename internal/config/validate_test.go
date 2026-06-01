package config

import (
	"strings"
	"testing"
)

/*
TC-CONFIG-VALIDATE-001
Type: Positive
Title: Default configure mode is placeholder
Summary:
Loads the default application config.
The configure backend should default to placeholder mode so CI and
local non-VyOS runs remain safe.

Validates:
  - default configure mode is placeholder
  - default config validates successfully
*/
func TestDefaultConfigureModeIsPlaceholder(t *testing.T) {
	cfg := DefaultAppConfig()
	if cfg.Agent.Configure.Mode != "placeholder" {
		t.Fatalf("default configure mode got=%q want=placeholder", cfg.Agent.Configure.Mode)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("default config should validate: %v", err)
	}
}

/*
TC-CONFIG-VALIDATE-002
Type: Positive
Title: Supported configure modes validate
Summary:
Checks each supported configure backend mode.
Both placeholder and real modes should pass validation because they
are valid runtime choices.

Validates:
  - placeholder configure mode is accepted
  - real configure mode is accepted
*/
func TestValidateConfigureModeAcceptsSupportedValues(t *testing.T) {
	for _, mode := range []string{"placeholder", "real"} {
		t.Run(mode, func(t *testing.T) {
			cfg := DefaultAppConfig()
			cfg.Agent.Configure.Mode = mode
			if err := cfg.Validate(); err != nil {
				t.Fatalf("configure mode %q should validate: %v", mode, err)
			}
		})
	}
}

/*
TC-CONFIG-VALIDATE-003
Type: Negative
Title: Unknown configure mode is rejected
Summary:
Sets configure mode to an unsupported value.
Validation should fail fast with an error that identifies the
invalid configure mode field.

Validates:
  - unknown configure mode returns an error
  - error mentions agent.configure.mode
*/
func TestValidateConfigureModeRejectsUnknownValue(t *testing.T) {
	cfg := DefaultAppConfig()
	cfg.Agent.Configure.Mode = "bogus"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "agent.configure.mode") {
		t.Fatalf("error %q does not mention agent.configure.mode", err.Error())
	}
}
