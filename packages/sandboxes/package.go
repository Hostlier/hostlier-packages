// Package sandboxes contributes private browser and desktop sidecars to a Hostlier v2 app.
package sandboxes

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
)

// Generated OCI profiles preserve the reviewed Playwright policy with all outer
// capabilities dropped. The receiving server architecture selects the profile.
//
//go:generate python3 generate_seccomp.py
//go:embed runtime/seccomp-amd64.json
var seccompAMD64 string

//go:embed runtime/seccomp-arm64.json
var seccompARM64 string

// Config contains public deployment choices and stored secret names, never secret values.
type Config struct {
	architecture string // Trusted request metadata, never a package setting.
	Image        string `json:"image,omitempty"`
	Archive      string `json:"archive,omitempty"`
	Name         string `json:"name"`
	Port         int    `json:"port"`
	TokenSecret  string `json:"tokenSecret"`
	URLEnv       string `json:"urlEnv"`
	TokenEnv     string `json:"tokenEnv"`
	CPU          string `json:"cpu"`
	Memory       string `json:"memory"`
}

// Request matches Hostlier's local project-contribute protocol.
type Request struct {
	Version          int             `json:"version"`
	ProjectDir       string          `json:"projectDir"`
	Architecture     string          `json:"architecture"`
	ProvidedBindings []string        `json:"providedBindings,omitempty"`
	Config           json.RawMessage `json:"config"`
}

// Contribution is the additive wire contract; the controller validates every member as a normal sidecar.
type Contribution struct {
	Version  int                       `json:"version"`
	Sidecars map[string]map[string]any `json:"sidecars"`
	Bindings map[string]string         `json:"bindings"`
	Secrets  map[string]string         `json:"secrets"`
}

// Defaults reserves different loopback ports and client bindings for the two packages.
func Defaults(kind string) Config {
	if kind == "browser" {
		return Config{Name: "browser", Port: 9222, URLEnv: "BROWSER_URL", TokenEnv: "BROWSER_TOKEN", CPU: "2", Memory: "2Gi"}
	}
	return Config{Name: "computer", Port: 9333, URLEnv: "COMPUTER_URL", TokenEnv: "COMPUTER_TOKEN", CPU: "2", Memory: "2Gi"}
}

// Serve handles discovery and pure planning without installing software or contacting a server.
func Serve(kind string, args []string, in io.Reader, out io.Writer) error {
	if kind != "browser" && kind != "computer" || len(args) != 1 {
		return errors.New("expected describe or project-contribute")
	}
	if args[0] == "describe" {
		return json.NewEncoder(out).Encode(Capability(kind))
	}
	if args[0] != "project-contribute" {
		return errors.New("expected describe or project-contribute")
	}
	c, err := readConfig(kind, in)
	if err != nil {
		return json.NewEncoder(out).Encode(map[string]any{"version": 1, "error": err.Error()})
	}
	proposal, err := Propose(kind, c)
	if err != nil {
		return json.NewEncoder(out).Encode(map[string]any{"version": 1, "error": err.Error()})
	}
	return json.NewEncoder(out).Encode(proposal)
}

// readConfig also validates direct executable calls that bypass CLI schema validation.
func readConfig(kind string, in io.Reader) (Config, error) {
	c := Defaults(kind)
	data, err := io.ReadAll(io.LimitReader(in, (4<<20)+1))
	if err != nil || len(data) > 4<<20 {
		return c, errors.New("invalid or oversized project request")
	}
	var req Request
	if strictJSON(data, &req) != nil || req.Version != 1 || (req.Architecture != "amd64" && req.Architecture != "arm64") {
		return c, errors.New("expected a version 1 project request for Linux amd64 or arm64")
	}
	if len(req.Config) == 0 || bytes.Equal(bytes.TrimSpace(req.Config), []byte("null")) || strictJSON(req.Config, &c) != nil {
		return c, errors.New("invalid sandbox configuration")
	}
	c.architecture = req.Architecture
	return c, validate(c)
}

// strictJSON prevents unknown fields and trailing documents from silently dropping intent.
func strictJSON(data []byte, out any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return err
	}
	if d.Decode(new(any)) != io.EOF {
		return errors.New("expected one JSON document")
	}
	return nil
}

// validate admits only public image selectors, portable names and bounded runtime allocations.
func validate(c Config) error {
	if (c.Image == "") == (c.Archive == "") {
		return errors.New("select exactly one of image or archive built from the sandbox runtime")
	}
	if c.Archive != "" && (!filepath.IsLocal(c.Archive) || len(c.Archive) > 1024) {
		return errors.New("archive must be a project-relative image archive")
	}
	if len(c.Image) > 1024 || regexp.MustCompile(`[\s\x00]`).MatchString(c.Image) {
		return errors.New("invalid image reference")
	}
	return validateNames(c)
}

// validateNames keeps diagnostics independent of supplied values, including mistaken token literals.
func validateNames(c Config) error {
	name := regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)
	env := regexp.MustCompile(`^[A-Z_][A-Z0-9_]{0,127}$`)
	if !name.MatchString(c.Name) || c.Name == "app" || !name.MatchString(c.TokenSecret) {
		return errors.New("name and tokenSecret must be lowercase identifiers; app is reserved")
	}
	if !env.MatchString(c.URLEnv) || !env.MatchString(c.TokenEnv) || c.URLEnv == c.TokenEnv {
		return errors.New("urlEnv and tokenEnv must be different environment names")
	}
	if c.Port < 1024 || c.Port > 65535 {
		return errors.New("port must be between 1024 and 65535")
	}
	if !regexp.MustCompile(`^[1-8]$`).MatchString(c.CPU) || !regexp.MustCompile(`^(2|3|4|6|8|16)Gi$`).MatchString(c.Memory) {
		return errors.New("cpu must be 1–8 and memory must be 2Gi, 3Gi, 4Gi, 6Gi, 8Gi or 16Gi")
	}
	return nil
}

// Propose delegates all creation, readiness, rollback and cleanup to Hostlier's existing runtime.
func Propose(kind string, c Config) (Contribution, error) {
	profile, ok := map[string]string{"amd64": seccompAMD64, "arm64": seccompARM64}[c.architecture]
	if !ok {
		return Contribution{}, errors.New("sandbox requires Linux amd64 or arm64")
	}
	source := map[string]any{"image": c.Image}
	if c.Archive != "" {
		source = map[string]any{"container": map[string]string{"archive": c.Archive}}
	}
	member := map[string]any{
		"source": source, "workload": "service", "required": true,
		"endpoint":  map[string]any{"exposure": "loopback", "port": c.Port},
		"resources": map[string]string{"cpu": c.CPU, "memory": c.Memory},
		"env":       map[string]string{"SANDBOX_MODE": kind, "SANDBOX_PORT": strconv.Itoa(c.Port)},
		"secrets":   map[string]string{"SANDBOX_TOKEN": c.TokenSecret},
		"health":    map[string]any{"exec": []string{"node", "/opt/hostlier-sandbox/health.mjs"}, "timeoutSeconds": 120},
		"shutdown":  map[string]int{"graceSeconds": 20},
		"security":  security(),
		"configs":   map[string]any{"sandbox-seccomp": map[string]any{"content": profile, "mount": "/opt/hostlier-sandbox/seccomp.json"}},
	}
	return Contribution{Version: 1, Sidecars: map[string]map[string]any{c.Name: member},
		Bindings: map[string]string{c.URLEnv: fmt.Sprintf("sidecars.%s.url", c.Name)},
		Secrets:  map[string]string{c.TokenEnv: c.TokenSecret}}, nil
}

// security keeps the browser's own sandbox enabled inside a separately confined nonroot container.
func security() map[string]any {
	return map[string]any{
		"runAsUser": 1000, "runAsGroup": 1000, "readOnlyRootFilesystem": true,
		"noNewPrivileges": true, "noSwap": true, "dropCapabilities": []string{"ALL"},
		"pidsLimit": 512, "seccompConfig": "sandbox-seccomp",
		"tmpfs": map[string]any{"/tmp": map[string]any{"size": "1Gi", "mode": 01777}},
	}
}
