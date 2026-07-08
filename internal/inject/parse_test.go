package inject

import "testing"

func TestParseDirectives(t *testing.T) {
	prompt := "Do the thing.\n" +
		"@inject-file:docs/guide.md\n" +
		"  @inject-cmd:`git show --stat HEAD`\n" +
		"@inject-cmd:wc -l go.mod\n" +
		"see @inject-file:not/anchored.md mid-sentence\n" +
		"@inject-file:   \n"
	got := ParseDirectives(prompt)
	want := []Directive{
		{Kind: "file", Arg: "docs/guide.md", Source: "file:docs/guide.md"},
		{Kind: "cmd", Arg: "git show --stat HEAD", Source: "cmd:git show --stat HEAD"},
		{Kind: "cmd", Arg: "wc -l go.mod", Source: "cmd:wc -l go.mod"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d directives %+v, want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("directive %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestParseDirectivesNone(t *testing.T) {
	if got := ParseDirectives("plain prompt, no tags"); len(got) != 0 {
		t.Fatalf("want none, got %+v", got)
	}
}
