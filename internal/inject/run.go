package inject

import "encoding/json"

type hookInput struct {
	ToolName  string                     `json:"tool_name"`
	ToolInput map[string]json.RawMessage `json:"tool_input"`
	CWD       string                     `json:"cwd"`
}

// Run implements the whole hook: stdin bytes in, stdout bytes out. A nil
// return means "print nothing" — the dispatch proceeds untouched. tool_input
// is held as raw JSON so every field we don't touch survives unmodified;
// only "prompt" is replaced.
func Run(stdin []byte, cfg Config) []byte {
	var in hookInput
	if json.Unmarshal(stdin, &in) != nil || in.ToolName != "Agent" || in.ToolInput == nil {
		return nil
	}
	var prompt string
	if json.Unmarshal(in.ToolInput["prompt"], &prompt) != nil || prompt == "" {
		return nil
	}
	res := Process(prompt, in.CWD, cfg)
	if res.Injected+res.Skipped+res.Failed == 0 {
		return nil
	}
	out := map[string]any{"systemMessage": res.Message()}
	if res.Prompt != prompt {
		raw, err := json.Marshal(res.Prompt)
		if err != nil {
			return nil
		}
		in.ToolInput["prompt"] = raw
		out["hookSpecificOutput"] = map[string]any{
			"hookEventName": "PreToolUse",
			"updatedInput":  in.ToolInput,
		}
	}
	b, err := json.Marshal(out)
	if err != nil {
		return nil
	}
	return b
}
