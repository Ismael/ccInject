package inject

import (
	"fmt"
	"strings"
)

// Unquoted, these signal the author expected a shell. ccinject never invokes
// a shell, so passing them through as data would silently mislead — reject.
const metachars = "|><;&`"

// SplitCommand tokenizes a command line with POSIX-style quoting and no
// expansion of any kind: single quotes are literal, double quotes are literal
// except \" and \\, a backslash escapes the next byte outside quotes.
func SplitCommand(line string) ([]string, error) {
	var args []string
	var cur strings.Builder
	inToken := false
	flush := func() {
		if inToken {
			args = append(args, cur.String())
			cur.Reset()
			inToken = false
		}
	}
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch c {
		case ' ', '\t':
			flush()
		case '\'':
			end := strings.IndexByte(line[i+1:], '\'')
			if end < 0 {
				return nil, fmt.Errorf("unbalanced single quote")
			}
			cur.WriteString(line[i+1 : i+1+end])
			inToken = true
			i += end + 1
		case '"':
			i++
			closed := false
			for i < len(line) {
				if line[i] == '\\' && i+1 < len(line) && (line[i+1] == '"' || line[i+1] == '\\') {
					cur.WriteByte(line[i+1])
					i += 2
					continue
				}
				if line[i] == '"' {
					closed = true
					break
				}
				cur.WriteByte(line[i])
				i++
			}
			if !closed {
				return nil, fmt.Errorf("unbalanced double quote")
			}
			inToken = true
		case '\\':
			if i+1 >= len(line) {
				return nil, fmt.Errorf("trailing backslash")
			}
			cur.WriteByte(line[i+1])
			inToken = true
			i++
		default:
			if strings.IndexByte(metachars, c) >= 0 {
				return nil, fmt.Errorf("unquoted %q — ccinject runs commands without a shell (no pipes/redirects)", string(c))
			}
			if c == '$' && i+1 < len(line) && line[i+1] == '(' {
				return nil, fmt.Errorf("unquoted $( — ccinject runs commands without a shell")
			}
			cur.WriteByte(c)
			inToken = true
		}
	}
	flush()
	if len(args) == 0 {
		return nil, fmt.Errorf("empty command")
	}
	return args, nil
}
