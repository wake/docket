package snapshot

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/wake/docket/internal/gitx"
)

func c(sha, time string, parents []string, trailers ...string) gitx.Commit {
	x := gitx.Commit{SHA: sha, Parents: parents, Time: time, Subject: "s " + sha}
	for i := 0; i+1 < len(trailers); i += 2 {
		x.Trailers = append(x.Trailers, gitx.Trailer{Key: trailers[i], Value: trailers[i+1]})
	}
	return x
}

var addX = Work{"Change", "add-x"}

func sideGraph() *Graph {
	return NewGraph([]gitx.Commit{
		c("m3", "2026-11-05T00:00:00Z", []string{"m2"}),
		c("m2", "2026-11-03T00:00:00Z", []string{"m1", "s2"}),
		c("s2", "2026-11-02T00:00:00Z", []string{"s1"}, "Change", "add-x"),
		c("s1", "2026-11-01T00:00:00Z", []string{"m1"}, "Change", "add-x/1"),
		c("m1", "2026-10-30T00:00:00Z", nil),
	})
}

func TestMerge(t *testing.T) {
	r := sideGraph().Merge("m3", addX)
	if r == nil || r.Commit != "m2" || r.Time != "2026-11-03T00:00:00Z" || r.Subject != "s m2" {
		t.Fatalf("got %+v", r)
	}
}

func TestMergeRebased(t *testing.T) {
	g := NewGraph([]gitx.Commit{
		c("c3", "3", []string{"c2"}),
		c("c2", "2", []string{"c1"}, "Record", "2026-11-06-fix-x"),
		c("c1", "1", nil),
	})
	r := g.Merge("c3", Work{"Record", "2026-11-06-fix-x"})
	if r == nil || r.Commit != "c2" {
		t.Fatalf("got %+v", r)
	}
}

func TestMergeNewest(t *testing.T) {
	g := NewGraph([]gitx.Commit{
		c("m4", "4", []string{"m3", "b2"}),
		c("b2", "3", []string{"m1"}, "Change", "add-x/2"),
		c("m3", "3", []string{"m2", "b1"}),
		c("b1", "2", []string{"m1"}, "Change", "add-x/1"),
		c("m2", "2", []string{"m1"}),
		c("m1", "1", nil),
	})
	if r := g.Merge("m4", addX); r == nil || r.Commit != "m4" {
		t.Fatalf("got %+v", r)
	}
}

func TestMergeNone(t *testing.T) {
	if r := sideGraph().Merge("m3", Work{"Change", "other"}); r != nil {
		t.Fatalf("got %+v", r)
	}
	if r := sideGraph().Merge("zz", addX); r != nil {
		t.Fatalf("got %+v", r)
	}
}

func TestNamesPrefixSlug(t *testing.T) {
	if n, _ := Names(c("a", "", nil, "Change", "add-x-2/1"), addX); n {
		t.Fatal("add-x-2 named add-x")
	}
	n, tasks := Names(c("a", "", nil, "Change", "add-x/1,3"), addX)
	if !n || !reflect.DeepEqual(tasks, []int{1, 3}) {
		t.Fatalf("got %v %v", n, tasks)
	}
	n, tasks = Names(c("a", "", nil, "Change", "add-x"), addX)
	if !n || len(tasks) != 0 {
		t.Fatalf("got %v %v", n, tasks)
	}
	if n, _ := Names(c("a", "", nil, "Record", "add-x"), addX); n {
		t.Fatal("Record named a Change")
	}
}

func TestRelease(t *testing.T) {
	g := NewGraph([]gitx.Commit{
		c("r3", "5", []string{"r2"}),
		c("r2", "4", []string{"m2"}),
		c("m2", "3", []string{"m1"}),
		c("m1", "1", nil),
	})
	tags := []gitx.Tag{{Name: "v0.1.0", Commit: "m1"}, {Name: "v0.2.0", Commit: "r2"}, {Name: "v0.3.0", Commit: "r3"}}
	if r := g.Release(tags, "m2"); r == nil || r.Name != "v0.2.0" {
		t.Fatalf("got %+v", r)
	}
	if r := g.Release(tags[:1], "m2"); r != nil {
		t.Fatalf("got %+v", r)
	}
}

func TestLast(t *testing.T) {
	g := NewGraph([]gitx.Commit{
		c("tip", "3", []string{"old"}),
		c("old", "2", nil, "Change", "add-x/1"),
		c("br", "4", []string{"old"}, "Change", "add-x"),
	})
	if r := g.Last([]string{"tip", "br"}, addX); r == nil || r.Commit != "br" {
		t.Fatalf("got %+v", r)
	}
	if r := g.Last([]string{"tip"}, addX); r == nil || r.Commit != "old" {
		t.Fatalf("got %+v", r)
	}
	if r := g.Last(nil, addX); r != nil {
		t.Fatalf("got %+v", r)
	}
}

func TestTaskCommits(t *testing.T) {
	g := NewGraph([]gitx.Commit{
		c("a3", "3", []string{"a2"}, "Change", "add-x/1,2"),
		c("a2", "2", []string{"a1"}, "Change", "add-x/1"),
		c("a1", "1", nil, "Change", "add-x"),
		c("other", "4", []string{"a1"}, "Change", "add-x/2"),
	})
	got := g.TaskCommits("a3", addX)
	want := map[int][]string{1: {"a2", "a3"}, 2: {"a3"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
}

func TestNamesMalformed(t *testing.T) {
	for _, v := range []string{"add-x/a", "add-x/1,a"} {
		n, tasks := Names(c("a", "", nil, "Change", v), addX)
		if !n || tasks != nil {
			t.Fatalf("%s: got %v %v", v, n, tasks)
		}
	}
}

func TestMergeLongHistory(t *testing.T) {
	var cs []gitx.Commit
	var tags []gitx.Tag
	prev := ""
	for i := 0; i < 20000; i++ {
		sha := fmt.Sprintf("c%05d", i)
		var parents []string
		if prev != "" {
			parents = []string{prev}
		}
		var tr []string
		if i%4 == 3 { // a merge of a side commit naming a work
			side := fmt.Sprintf("s%05d", i)
			cs = append(cs, c(side, "0", []string{prev}, "Change", fmt.Sprintf("w%d/1", i%200)))
			parents = append(parents, side)
		}
		cs = append(cs, c(sha, "0", parents, tr...))
		if i%400 == 399 {
			tags = append(tags, gitx.Tag{Name: fmt.Sprint("v", i), Commit: sha})
		}
		prev = sha
	}
	g := NewGraph(cs)
	start := time.Now()
	for w := 0; w < 200; w++ {
		g.Merge(prev, Work{"Change", fmt.Sprintf("w%d", w)})
		g.Release(tags, fmt.Sprintf("c%05d", w*100))
	}
	if r := g.Merge(prev, Work{"Change", "w3"}); r == nil || r.Commit != "c19803" {
		t.Fatalf("merge: %+v", r)
	}
	if r := g.Release(tags, "c00100"); r == nil || r.Name != "v399" {
		t.Fatalf("release: %+v", r)
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("took %v", d)
	}
}

func TestMergeOlderAttribution(t *testing.T) {
	g := NewGraph([]gitx.Commit{
		c("m4", "4", []string{"m3", "d"}),
		c("d", "4", []string{"b"}),
		c("m3", "3", []string{"m2"}),
		c("m2", "2", []string{"m1", "b"}),
		c("b", "2", []string{"m1"}, "Change", "add-x/1"),
		c("m1", "1", nil),
	})
	if r := g.Merge("m4", addX); r == nil || r.Commit != "m2" {
		t.Fatalf("got %+v", r)
	}
}
