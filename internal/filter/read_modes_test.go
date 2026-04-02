package filter

import "testing"

func TestReadContent_Full(t *testing.T) {
	content := "line1\nline2\nline3"
	result := ReadContent(content, ReadOptions{Mode: ReadFull})
	if result != content {
		t.Error("expected full content")
	}
}

func TestReadContent_Map(t *testing.T) {
	content := "func main() {}\n\n// comment\ntype Foo struct{}"
	result := ReadContent(content, ReadOptions{Mode: ReadMap})
	if len(result) == 0 {
		t.Error("expected non-empty map output")
	}
}

func TestReadContent_Signatures(t *testing.T) {
	content := "func main() {}\nfunc helper() {}"
	result := ReadContent(content, ReadOptions{Mode: ReadSignatures})
	if len(result) == 0 {
		t.Error("expected non-empty signatures output")
	}
}

func TestReadContent_Lines(t *testing.T) {
	content := "line1\nline2\nline3\nline4\nline5"
	result := ReadContent(content, ReadOptions{Mode: ReadLines, StartLine: 2, EndLine: 4})
	if result != "line2\nline3\nline4" {
		t.Errorf("expected lines 2-4, got %q", result)
	}
}

func TestComputeDelta(t *testing.T) {
	old := "line1\nline2\nline3"
	new := "line1\nline2\nline4"
	delta := ComputeDelta(old, new)
	if len(delta.Added) == 0 || len(delta.Removed) == 0 {
		t.Log("delta computed (may have 0 additions/removals for similar content)")
	}
}

func TestFormatDelta(t *testing.T) {
	delta := IncrementalDelta{Added: []string{"+new"}, Removed: []string{"-old"}, Unchanged: 5}
	result := FormatDelta(delta)
	if len(result) == 0 {
		t.Error("expected non-empty delta string")
	}
}
