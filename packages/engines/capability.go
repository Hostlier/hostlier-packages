package engines

import "github.com/Usefused/hostlier-packages/sdk/workflow"

// Capability describes the native executable's configuration and authenticated recovery contract.
func (p Package) Capability() workflow.Capability {
	modes := []string{"new-instance", "existing"}
	// PostgreSQL provisions an owned database rather than a Redis-style namespace.
	if p.Engine == "postgres" {
		modes = []string{"new-instance", "new-database"}
	}
	return workflow.Capability{Version: 1, Name: p.Engine, Inspect: true, ConfigSchema: map[string]any{"type": "object", "additionalProperties": false, "required": []string{"mode"}, "properties": map[string]any{"mode": map[string]any{"type": "string", "enum": modes}, "storage": map[string]any{"type": "string"}, "database": map[string]any{"type": "string"}, "keyPrefix": map[string]any{"type": "string"}}}, SecretAliases: []string{"connection", "caCertificatePEM"}, OutputSchema: map[string]any{"type": "object", "additionalProperties": false, "required": []string{"url", "host", "port", "username", "password"}, "properties": outputProperties()}, Outputs: []string{"url", "host", "port", "username", "password"}}
}

// outputProperties describes output types without producing credentials or runtime effects.
func outputProperties() map[string]any {
	properties := map[string]any{}
	// Native engine bindings use string values, including their port.
	for _, name := range []string{"url", "host", "port", "username", "password"} {
		properties[name] = map[string]any{"type": "string"}
	}
	return properties
}
