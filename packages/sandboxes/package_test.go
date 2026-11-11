package sandboxes

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// TestProtocolContributions exercises both binaries' contract without contacting providers or building images.
func TestProtocolContributions(t *testing.T) {
	for _, kind := range []string{"browser", "computer"} {
		var out bytes.Buffer
		request := `{"version":1,"architecture":"arm64","config":{"image":"example/sandbox:1","tokenSecret":"agent-token"}}`
		if err := Serve(kind, []string{"project-contribute"}, strings.NewReader(request), &out); err != nil {
			t.Fatal(err)
		}
		var c Contribution
		if err := json.Unmarshal(out.Bytes(), &c); err != nil || len(c.Sidecars) != 1 {
			t.Fatal(out.String(), err)
		}
		member := c.Sidecars[kind]
		if member["endpoint"].(map[string]any)["exposure"] != "loopback" || member["required"] != true {
			t.Fatal("public or optional sandbox", member)
		}
		security := member["security"].(map[string]any)
		if security["runAsUser"] != float64(1000) || security["readOnlyRootFilesystem"] != true {
			t.Fatal("missing confinement", security)
		}
		if c.Secrets[Defaults(kind).TokenEnv] != "agent-token" || c.Bindings[Defaults(kind).URLEnv] != "sidecars."+kind+".url" {
			t.Fatal("missing parent connection", c)
		}
	}
}

// TestInvalidRequestsFailClosed covers direct hook calls, safe diagnostics and mutually exclusive sources.
func TestInvalidRequestsFailClosed(t *testing.T) {
	requests := []string{
		`{"version":2,"architecture":"arm64","config":{}}`,
		`{"version":1,"architecture":"mips","config":{}}`,
		`{"version":1,"architecture":"amd64","config":null}`,
		`{"version":1,"architecture":"amd64","config":{"image":"x","tokenSecret":"DO_NOT_ECHO/private"}}`,
		`{"version":1,"architecture":"amd64","config":{"image":"x","archive":"x.tar","tokenSecret":"token"}}`,
		`{"version":1,"architecture":"amd64","config":{"archive":"../x.tar","tokenSecret":"token"}}`,
		`{"version":1,"architecture":"amd64","config":{"image":"x","tokenSecret":"token","unknown":"DO_NOT_ECHO"}}`,
		`{"version":1,"architecture":"amd64","config":{"image":"x","tokenSecret":"token","port":80}}`,
		`{} {}`,
	}
	for _, request := range requests {
		var out bytes.Buffer
		if err := Serve("browser", []string{"project-contribute"}, strings.NewReader(request), &out); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), `"error"`) || strings.Contains(out.String(), "DO_NOT_ECHO") {
			t.Fatal(out.String())
		}
	}
}

// TestArchiveAndCustomBindings preserves artifact upload and configurable names without provisioning.
func TestArchiveAndCustomBindings(t *testing.T) {
	request := `{"version":1,"architecture":"amd64","config":{"archive":"images/computer.tar","tokenSecret":"computer-key","name":"desktop","port":9334,"urlEnv":"DESKTOP_URL","tokenEnv":"DESKTOP_TOKEN"}}`
	var out bytes.Buffer
	if err := Serve("computer", []string{"project-contribute"}, strings.NewReader(request), &out); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"images/computer.tar", "sidecars.desktop.url", "DESKTOP_TOKEN", "9334"} {
		if !strings.Contains(out.String(), expected) {
			t.Fatal("missing", expected)
		}
	}
}

// TestOCIProfiles preserves effective policy while removing Docker-only metadata.
func TestOCIProfiles(t *testing.T) {
	for _, arch := range []string{"amd64", "arm64"} {
		for _, kind := range []string{"browser", "computer"} {
			t.Run(kind+"/"+arch, func(t *testing.T) {
				var out bytes.Buffer
				request := `{"version":1,"architecture":"` + arch + `","config":{"image":"example/sandbox:1","tokenSecret":"key"}}`
				if err := Serve(kind, []string{"project-contribute"}, strings.NewReader(request), &out); err != nil {
					t.Fatal(err)
				}
				var c Contribution
				if err := json.Unmarshal(out.Bytes(), &c); err != nil {
					t.Fatal(err)
				}
				profile := c.Sidecars[kind]["configs"].(map[string]any)["sandbox-seccomp"].(map[string]any)["content"].(string)
				assertOCIProfile(t, profile, arch)
			})
		}
	}
}

// assertOCIProfile checks safety invariants independently of the profile generator.
func assertOCIProfile(t *testing.T, raw, arch string) {
	t.Helper()
	var p struct {
		DefaultAction string
		Architectures []string
		Syscalls      []struct {
			Names  []string
			Action string
			Args   []any
		}
	}
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{`"archMap"`, `"includes"`, `"excludes"`} {
		if strings.Contains(raw, forbidden) {
			t.Fatal("Docker metadata retained", forbidden)
		}
	}
	want := map[string][]string{"amd64": {"SCMP_ARCH_X86_64", "SCMP_ARCH_X86", "SCMP_ARCH_X32"}, "arm64": {"SCMP_ARCH_AARCH64", "SCMP_ARCH_ARM"}}[arch]
	if p.DefaultAction != "SCMP_ACT_ERRNO" || strings.Join(p.Architectures, ",") != strings.Join(want, ",") {
		t.Fatal(p)
	}
	allowed := map[string]bool{}
	for _, r := range p.Syscalls {
		if r.Action == "SCMP_ACT_ALLOW" && len(r.Args) == 0 {
			for _, name := range r.Names {
				allowed[name] = true
			}
		}
	}
	for _, name := range []string{"chroot", "clone", "setns", "unshare"} {
		if !allowed[name] {
			t.Fatal("Chromium namespace syscall blocked", name)
		}
	}
	for _, r := range p.Syscalls {
		for _, name := range r.Names {
			if r.Action == "SCMP_ACT_ALLOW" && strings.Contains("|mount|umount|umount2|reboot|bpf|perf_event_open|init_module|finit_module|delete_module|open_by_handle_at|", "|"+name+"|") {
				t.Fatal("capability-gated syscall admitted", name)
			}
		}
	}
	if allowed["arch_prctl"] != (arch == "amd64") || allowed["arm_fadvise64_64"] != (arch == "arm64") {
		t.Fatal("wrong architecture filters")
	}
}
