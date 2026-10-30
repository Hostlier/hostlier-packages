package packageexec

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/Usefused/hostlier-packages/sdk/workflow"
	"io"
	"os/exec"
	"time"
)

// Describe discovers public metadata through the same verified snapshot as lifecycle execution.
func (r Runner) Describe(ctx context.Context, name string) (workflow.Capability, error) {
	path, _, cleanup, err := r.snapshot(name, "")
	// Unsafe installations never participate in discovery.
	if err != nil {
		return workflow.Capability{}, err
	}
	defer cleanup()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, path, "describe")
	command.Env = []string{"PATH=/usr/local/bin:/usr/bin:/bin", "LANG=C", "HOME=" + r.Directory}
	command.Stderr = io.Discard
	command.WaitDelay = time.Second
	output := &limitedOutput{limit: 1048576}
	command.Stdout = output
	// Ignore private child diagnostics and bound metadata before decoding.
	if command.Run() != nil || output.overflow {
		return workflow.Capability{}, errors.New("package discovery failed")
	}
	return decodeCapability(output.Bytes(), name)
}

// decodeCapability accepts exactly one compatible public contract for the installed identity.
func decodeCapability(raw []byte, name string) (workflow.Capability, error) {
	var capability workflow.Capability
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	// Missing schemas, identity mismatches and trailing documents invalidate discovery.
	if decoder.Decode(&capability) != nil || decoder.Decode(new(any)) != io.EOF || capability.Version != 1 || capability.Name != name || capability.ConfigSchema == nil || capability.OutputSchema == nil {
		return workflow.Capability{}, errors.New("invalid package capability")
	}
	return capability, nil
}
