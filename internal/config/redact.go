package config

import "strings"

const redactedValue = "********"

func (c AppConfig) Redacted() AppConfig {
	out := c

	out.AgentCore.NATS.Password = redactString(out.AgentCore.NATS.Password)
	out.AgentCore.NATS.Token = redactString(out.AgentCore.NATS.Token)
	out.AgentCore.NATS.CredentialsFile = redactString(out.AgentCore.NATS.CredentialsFile)
	out.AgentCore.NATS.NKeySeedFile = redactString(out.AgentCore.NATS.NKeySeedFile)
	out.AgentCore.NATS.UserJWTFile = redactString(out.AgentCore.NATS.UserJWTFile)

	return out
}

func redactString(v string) string {
	if strings.TrimSpace(v) == "" {
		return ""
	}
	return redactedValue
}
