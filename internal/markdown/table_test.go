package markdown

import (
	"reflect"
	"testing"
)

func find(t *testing.T, lines []string) (Table, bool) {
	t.Helper()
	return FindTable(lines, FenceMask(lines), 0, len(lines))
}

func TestFindTable(t *testing.T) {
	t.Run("header delimiter rows", func(t *testing.T) {
		lines := []string{"intro", "| A | B |", "|:--|--:|", "| 1 | 2 |", "| 3 | 4 |", "", "| x | y |"}
		tb, ok := find(t, lines)
		if !ok {
			t.Fatal("no table")
		}
		if tb.Index != 1 || !reflect.DeepEqual(tb.Header, []string{"A", "B"}) {
			t.Fatalf("table = %+v", tb)
		}
		want := []Row{{[]string{"1", "2"}, 3}, {[]string{"3", "4"}, 4}}
		if !reflect.DeepEqual(tb.Rows, want) {
			t.Fatalf("rows = %+v", tb.Rows)
		}
	})
	t.Run("escaped pipe", func(t *testing.T) {
		tb, ok := find(t, []string{"A | B", "-|-", `a\|b | c`})
		if !ok || !reflect.DeepEqual(tb.Rows[0].Cells, []string{"a|b", "c"}) {
			t.Fatalf("table = %+v ok=%v", tb, ok)
		}
	})
	t.Run("short row padded", func(t *testing.T) {
		tb, ok := find(t, []string{"| A | B | C |", "|---|---|---|", "| 1 |"})
		if !ok || !reflect.DeepEqual(tb.Rows[0].Cells, []string{"1", "", ""}) {
			t.Fatalf("table = %+v ok=%v", tb, ok)
		}
	})
	t.Run("in fence", func(t *testing.T) {
		if _, ok := find(t, []string{"```", "| A |", "|---|", "| 1 |", "```"}); ok {
			t.Fatal("found table in fence")
		}
	})
	t.Run("no delimiter", func(t *testing.T) {
		if _, ok := find(t, []string{"| A | B |", "| 1 | 2 |"}); ok {
			t.Fatal("found table without delimiter")
		}
	})
	t.Run("range", func(t *testing.T) {
		lines := []string{"| A |", "|---|", "| 1 |", "x", "| B |", "|---|", "| 2 |"}
		tb, ok := FindTable(lines, FenceMask(lines), 3, len(lines))
		if !ok || tb.Header[0] != "B" {
			t.Fatalf("table = %+v ok=%v", tb, ok)
		}
	})
}
