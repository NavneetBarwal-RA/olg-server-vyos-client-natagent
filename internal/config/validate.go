package config

import (
	"fmt"
	"strings"
	"time"
)

func (c AppConfig) Validate() error {
	if strings.TrimSpace(c.Agent.Target) == "" {
		return fmt.Errorf("agent.target is required")
	}
	if strings.TrimSpace(c.Agent.StateFile) == "" {
		return fmt.Errorf("agent.state_file is required")
	}
	if c.Agent.Renderer.Mode != "placeholder" {
		return fmt.Errorf("agent.renderer.mode must be placeholder")
	}
	if c.Agent.Apply.Mode != "placeholder" {
		return fmt.Errorf("agent.apply.mode must be placeholder")
	}
	for _, action := range c.Agent.Actions.Enabled {
		if action != "trace" {
			return fmt.Errorf("agent.actions.enabled contains unsupported action %q", action)
		}
	}

	if len(c.AgentCore.NATS.Servers) == 0 {
		return fmt.Errorf("agentcore.nats.servers must not be empty")
	}
	hasServer := false
	for _, server := range c.AgentCore.NATS.Servers {
		if strings.TrimSpace(server) != "" {
			hasServer = true
			break
		}
	}
	if !hasServer {
		return fmt.Errorf("agentcore.nats.servers must contain at least one non-empty server")
	}

	if strings.TrimSpace(c.AgentCore.Subjects.ConfigurePattern) == "" {
		return fmt.Errorf("agentcore.subjects.configure_pattern is required")
	}
	if strings.TrimSpace(c.AgentCore.Subjects.ActionPattern) == "" {
		return fmt.Errorf("agentcore.subjects.action_pattern is required")
	}
	if strings.TrimSpace(c.AgentCore.Subjects.ResultPattern) == "" {
		return fmt.Errorf("agentcore.subjects.result_pattern is required")
	}
	if strings.TrimSpace(c.AgentCore.Subjects.StatusPattern) == "" {
		return fmt.Errorf("agentcore.subjects.status_pattern is required")
	}
	if strings.TrimSpace(c.AgentCore.Subjects.HealthPattern) == "" {
		return fmt.Errorf("agentcore.subjects.health_pattern is required")
	}

	if strings.TrimSpace(c.AgentCore.KV.Bucket) == "" {
		return fmt.Errorf("agentcore.kv.bucket is required")
	}
	if strings.TrimSpace(c.AgentCore.KV.KeyPattern) == "" {
		return fmt.Errorf("agentcore.kv.key_pattern is required")
	}
	if c.AgentCore.KV.History == 0 {
		return fmt.Errorf("agentcore.kv.history must be greater than zero")
	}
	if c.AgentCore.KV.Replicas < 1 {
		return fmt.Errorf("agentcore.kv.replicas must be at least 1")
	}
	if c.AgentCore.Retry.PublishAttempts < 1 {
		return fmt.Errorf("agentcore.retry.publish_attempts must be at least 1")
	}

	if c.AgentCore.Execution.HandlerMode != "sync" {
		return fmt.Errorf("agentcore.execution.handler_mode must be sync")
	}

	requiredDurations := []struct {
		field string
		value string
	}{
		{field: "agentcore.nats.connect_timeout", value: c.AgentCore.NATS.ConnectTimeout},
		{field: "agentcore.nats.reconnect_wait", value: c.AgentCore.NATS.ReconnectWait},
		{field: "agentcore.jetstream.default_timeout", value: c.AgentCore.JetStream.DefaultTimeout},
		{field: "agentcore.timeouts.publish_timeout", value: c.AgentCore.Timeouts.PublishTimeout},
		{field: "agentcore.timeouts.subscribe_timeout", value: c.AgentCore.Timeouts.SubscribeTimeout},
		{field: "agentcore.timeouts.kv_timeout", value: c.AgentCore.Timeouts.KVTimeout},
		{field: "agentcore.timeouts.shutdown_timeout", value: c.AgentCore.Timeouts.ShutdownTimeout},
		{field: "agentcore.timeouts.handler_warn_after", value: c.AgentCore.Timeouts.HandlerWarnAfter},
		{field: "agentcore.retry.publish_backoff", value: c.AgentCore.Retry.PublishBackoff},
	}
	for _, item := range requiredDurations {
		if _, err := parseDurationField(item.field, item.value, false); err != nil {
			return err
		}
	}

	if _, err := parseDurationField("agentcore.kv.ttl", c.AgentCore.KV.TTL, true); err != nil {
		return err
	}

	return nil
}

func parseDurationField(fieldName, raw string, optional bool) (time.Duration, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		if optional {
			return 0, nil
		}
		return 0, fmt.Errorf("%s is required", fieldName)
	}

	d, err := time.ParseDuration(trimmed)
	if err != nil {
		return 0, fmt.Errorf("%s is not a valid duration: %w", fieldName, err)
	}
	return d, nil
}
