package sandboxes

// Capability exposes configuration for hostlier package add without selecting a primary builder.
func Capability(kind string) map[string]any {
	c := Defaults(kind)
	return map[string]any{
		"version": 1, "name": "sandbox-" + kind, "hooks": []string{"project-contribute"},
		"configSchema": map[string]any{
			"type": "object", "additionalProperties": false, "required": []string{"tokenSecret"},
			"oneOf": []any{
				map[string]any{"required": []string{"image"}, "not": map[string]any{"required": []string{"archive"}}},
				map[string]any{"required": []string{"archive"}, "not": map[string]any{"required": []string{"image"}}},
			},
			"properties": map[string]any{
				"image":       map[string]any{"type": "string", "minLength": 1, "maxLength": 1024, "description": "Registry image built from this package's matching runtime Dockerfile. Choose image or archive."},
				"archive":     map[string]any{"type": "string", "minLength": 1, "maxLength": 1024, "description": "Project-relative docker save archive of the matching runtime. Choose image or archive."},
				"name":        stringField(c.Name, "Sidecar name; unique within the application."),
				"port":        map[string]any{"type": "integer", "minimum": 1024, "maximum": 65535, "default": c.Port, "description": "Loopback API port in the shared application network namespace."},
				"tokenSecret": map[string]any{"type": "string", "pattern": "^[a-z][a-z0-9-]{0,62}$", "description": "Existing Hostlier secret name holding a random token of at least 32 characters. Never enter the token here."},
				"urlEnv":      stringField(c.URLEnv, "Parent application environment variable for the HTTP base URL."),
				"tokenEnv":    stringField(c.TokenEnv, "Parent application environment variable for the API token."),
				"cpu":         map[string]any{"type": "string", "pattern": "^[1-8]$", "default": c.CPU},
				"memory":      map[string]any{"type": "string", "enum": []string{"2Gi", "3Gi", "4Gi", "6Gi", "8Gi", "16Gi"}, "default": c.Memory},
			},
		},
		"outputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
	}
}

// stringField describes a public setting whose semantic validation also runs inside the executable.
func stringField(value, description string) map[string]any {
	return map[string]any{"type": "string", "minLength": 1, "maxLength": 128, "default": value, "description": description}
}
