package packagekit

import (
	"context"

	"github.com/Usefused/hostlier-packages/sdk/workflow"
	"testing"
)

// testConfig is deliberately strict so schema drift is observable.
type testConfig struct {
	Name string `json:"name"`
}

// TestDefinition validates typed dispatch, strict config, secret aliases and secret output mapping.
func TestDefinition(t *testing.T) {
	// Exercise each controlled fixture path without real provider mutations.
	d, err := Define(Definition[testConfig, string, string, string]{Name: "fixture", Config: Codec[testConfig]{Schema: map[string]any{"type": "object"}}, Outputs: OutputCodec[string]{Codec: Codec[string]{Schema: map[string]any{"type": "string"}}, Names: []string{"credential", "status"}, Encode: func(v string) (map[string]string, map[string]string, error) {
		return map[string]string{"status": "ready"}, map[string]string{"credential": v}, nil
		// Exercise each controlled fixture path without real provider mutations.
	}}, Plan: func(context.Context, Call[testConfig, string]) (Plan[string], error) {
		return Plan[string]{Prepared: "saved", Summary: []string{"one change"}}, nil
		// Exercise each controlled fixture path without real provider mutations.
	}, Apply: func(context.Context, Call[testConfig, string], string) (Result[string, string], error) {
		return Result[string, string]{State: "receipt", Outputs: "secret-sentinel"}, nil
	}})
	// Verify the expected preparation and approval boundary.
	if err != nil {
		t.Fatal(err)
	}
	r := workflow.Request{Version: 1, Key: "operation", Action: "plan", Step: workflow.Step{Config: map[string]any{"name": "fixture"}}}
	response, err := d.Handle(context.Background(), r)
	// Verify the expected preparation and approval boundary.
	if err != nil || response.Prepared == nil {
		t.Fatal(err)
	}
	r.Action = "apply"
	r.Prepared = response.Prepared.Payload
	response, err = d.Handle(context.Background(), r)
	// Verify the expected preparation and approval boundary.
	if err != nil {
		t.Fatal(err)
	}
	// Verify the expected preparation and approval boundary.
	if response.Result.Outputs["credential"] != "" || response.Result.PrivateOutputs["credential"] != "secret-sentinel" {
		t.Fatal("private output was published")
	}
	r.Secrets = map[string]string{"unknown": "sentinel"}
	// Verify the expected preparation and approval boundary.
	if _, err = d.Handle(context.Background(), r); err == nil {
		t.Fatal("undeclared secret accepted")
	}
	r.Secrets = nil
	r.Step.Config["unknown"] = true
	// Verify the expected preparation and approval boundary.
	if _, err = d.Handle(context.Background(), r); err == nil {
		t.Fatal("unknown config accepted")
	}
}

// TestContractNames rejects duplicate and nonbinding capability aliases.
func TestContractNames(t *testing.T) {
	// Exercise each controlled fixture path without real provider mutations.
	for _, names := range [][]string{{"same", "same"}, {"BAD=VALUE"}, {"_leading"}} {
		// Verify the expected preparation and approval boundary.
		if validNames(names) {
			t.Fatal("invalid names accepted", names)
		}
	}
	// Verify the expected preparation and approval boundary.
	if !validNames([]string{"DB_URL", "endpoint"}) {
		t.Fatal("valid aliases rejected")
	}
}
