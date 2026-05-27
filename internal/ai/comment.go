package ai

import "strings"

// CommentBlock is a contiguous run of "--" comment lines containing a cursor.
// StartLine and EndLine are 0-indexed and inclusive, suitable for slicing the
// editor buffer when replacing the block on accept.
type CommentBlock struct {
	Text      string
	StartLine int
	EndLine   int
}

// ExtractCommentBlock returns the comment block at the given line, if any.
// Lines are considered comments when their first non-whitespace characters
// are "--". The returned Text is the joined comment payload with the "--"
// markers and one optional leading space stripped from each line.
func ExtractCommentBlock(buf string, line int) (CommentBlock, bool) {
	lines := strings.Split(buf, "\n")
	if line < 0 || line >= len(lines) {
		return CommentBlock{}, false
	}
	if !isCommentLine(lines[line]) {
		return CommentBlock{}, false
	}

	start := line
	for start > 0 && isCommentLine(lines[start-1]) {
		start--
	}
	end := line
	for end < len(lines)-1 && isCommentLine(lines[end+1]) {
		end++
	}

	parts := make([]string, 0, end-start+1)
	for i := start; i <= end; i++ {
		parts = append(parts, stripCommentMarker(lines[i]))
	}
	text := strings.TrimSpace(strings.Join(parts, "\n"))
	if text == "" {
		return CommentBlock{}, false
	}
	return CommentBlock{Text: text, StartLine: start, EndLine: end}, true
}

func isCommentLine(s string) bool {
	return strings.HasPrefix(strings.TrimLeft(s, " \t"), "--")
}

func stripCommentMarker(s string) string {
	trimmed := strings.TrimLeft(s, " \t")
	rest := strings.TrimPrefix(trimmed, "--")
	return strings.TrimPrefix(rest, " ")
}
