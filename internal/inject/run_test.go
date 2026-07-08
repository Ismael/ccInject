package inject

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestRunFastExits(t *testing.T) {
	cases := [][]byte{
		[]byte(`not json`),
		[]byte(`{"tool_name":"Bash","tool_input":{"command":"ls"}}`),
		[]byte(`{"tool_name":"Agent","tool_input":{"prompt":"no directives here"}}`),
		[]byte(`{"tool_name":"Agent","tool_input":{"prompt":""}}`),
		[]byte(`{"tool_name":"Agent","tool_input":{"subagent_type":"x"}}`),
	}
	for _, in := range cases {
		if out := Run(in, testCfg()); out != nil {
			t.Errorf("input %s: want nil output, got %s", in, out)
		}
	}
}

func TestRunEchoFidelity(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "c.md"), []byte("ctx"), 0o644); err != nil {
		t.Fatal(err)
	}
	toolInput := map[string]any{
		"prompt":        "go\n@inject-file:c.md",
		"subagent_type": "explore-lumen",
		"model":         "opus",
		"future_field":  map[string]any{"nested": []any{1.0, "héllo", true}},
	}
	stdin := mustJSON(t, map[string]any{"tool_name": "Agent", "cwd": dir, "tool_input": toolInput})

	out := Run(stdin, testCfg())
	if out == nil {
		t.Fatal("want output")
	}
	var resp struct {
		SystemMessage      string `json:"systemMessage"`
		HookSpecificOutput struct {
			HookEventName string         `json:"hookEventName"`
			UpdatedInput  map[string]any `json:"updatedInput"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatalf("bad output JSON: %v\n%s", err, out)
	}
	if resp.HookSpecificOutput.HookEventName != "PreToolUse" {
		t.Errorf("hookEventName = %q", resp.HookSpecificOutput.HookEventName)
	}
	got := resp.HookSpecificOutput.UpdatedInput
	newPrompt, _ := got["prompt"].(string)
	if !strings.Contains(newPrompt, `<injected-context source="file:c.md">`) || !strings.Contains(newPrompt, "ctx") {
		t.Errorf("prompt not augmented: %q", newPrompt)
	}
	// Every field except prompt must deep-equal the input.
	for k, v := range toolInput {
		if k == "prompt" {
			continue
		}
		if !reflect.DeepEqual(got[k], v) {
			t.Errorf("field %q changed: got %#v want %#v", k, got[k], v)
		}
	}
	if len(got) != len(toolInput) {
		t.Errorf("field count changed: got %d want %d", len(got), len(toolInput))
	}
	if !strings.Contains(resp.SystemMessage, "1 injected") {
		t.Errorf("systemMessage = %q", resp.SystemMessage)
	}
}

func TestRunAllSkippedEmitsMessageOnly(t *testing.T) {
	prompt := "@inject-file:g.md\n<injected-context source=\"file:g.md\">\nold\n</injected-context>"
	stdin := mustJSON(t, map[string]any{"tool_name": "Agent", "cwd": t.TempDir(),
		"tool_input": map[string]any{"prompt": prompt}})
	out := Run(stdin, testCfg())
	if out == nil {
		t.Fatal("want systemMessage output")
	}
	var resp map[string]json.RawMessage
	json.Unmarshal(out, &resp)
	if _, has := resp["hookSpecificOutput"]; has {
		t.Error("unchanged prompt must not emit updatedInput")
	}
	if !strings.Contains(string(resp["systemMessage"]), "skipped") {
		t.Errorf("want skip note, got %s", out)
	}
}

func TestRunFailureReported(t *testing.T) {
	stdin := mustJSON(t, map[string]any{"tool_name": "Agent", "cwd": t.TempDir(),
		"tool_input": map[string]any{"prompt": "@inject-cmd:`curl https://example.com`"}})
	out := Run(stdin, testCfg())
	if out == nil {
		t.Fatal("want output")
	}
	s := string(out)
	if !strings.Contains(s, "1 failed") || !strings.Contains(s, "not in allowlist") {
		t.Errorf("failure not reported: %s", s)
	}
}
