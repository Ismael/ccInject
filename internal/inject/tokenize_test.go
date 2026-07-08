package inject

import (
	"reflect"
	"testing"
)

func TestSplitCommandQuoting(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{`jq '.tasks[] | select(.id == "3")' plan.md.tasks.json`,
			[]string{"jq", `.tasks[] | select(.id == "3")`, "plan.md.tasks.json"}},
		{`git show --stat HEAD~1..HEAD`, []string{"git", "show", "--stat", "HEAD~1..HEAD"}},
		{`grep "a b" file\ name`, []string{"grep", "a b", "file name"}},
		{`sed -n '10,20p' notes.txt`, []string{"sed", "-n", "10,20p", "notes.txt"}},
	}
	for _, c := range cases {
		got, err := SplitCommand(c.in)
		if err != nil {
			t.Errorf("%q: unexpected error %v", c.in, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%q: got %#v, want %#v", c.in, got, c.want)
		}
	}
}

func TestSplitCommandRejects(t *testing.T) {
	for _, bad := range []string{
		"cat a | wc -l", "cat a > b", "git log; id", "cat a & cat b",
		"echo `id`", "cat $(pwd)/f", "cat 'unbalanced", `cat "unbalanced`, "   ",
	} {
		if _, err := SplitCommand(bad); err == nil {
			t.Errorf("%q: want error, got nil", bad)
		}
	}
}
