package main

import (
	"context"
	"fmt"
	"github.com/yshujie/miniblog/internal/miniblog/notionsync"
	"os"
	"path/filepath"
	"testing"
)

func TestBootstrapWriteRequiresBothOptIns(t *testing.T) {
	for _, c := range []struct {
		mode  string
		write bool
		path  string
	}{{"bootstrap_apply", false, "review.json"}, {"bootstrap_apply", true, ""}, {"dry_run", true, "review.json"}, {"invalid", false, ""}} {
		if validateMode(c.mode, c.write, c.path) == nil {
			t.Fatalf("accepted unsafe mode %+v", c)
		}
	}
	if e := validateMode("bootstrap_apply", true, "review.json"); e != nil {
		t.Fatal(e)
	}
}
func TestConfirmationFileStrictJSON(t *testing.T) {
	for _, content := range []string{`{"confirmations":[]}`, `{"confirmations":[],"unexpected":true}`, `{"confirmations":[]} {}`} {
		path := filepath.Join(t.TempDir(), "review.json")
		if e := os.WriteFile(path, []byte(content), 0600); e != nil {
			t.Fatal(e)
		}
		if _, e := readConfirmations(path); e == nil {
			t.Fatalf("accepted %s", content)
		}
	}
}

type fixtureItemsReader struct {
	total int
	empty bool
	calls int
}

func (f *fixtureItemsReader) Items(_ context.Context, _ string, q notionsync.ListQuery) (*notionsync.PageResult[notionsync.ItemDTO], error) {
	f.calls++
	result := &notionsync.PageResult[notionsync.ItemDTO]{Total: int64(f.total), Items: []notionsync.ItemDTO{}}
	if f.empty {
		return result, nil
	}
	start := (q.Page - 1) * q.Limit
	end := start + q.Limit
	if end > f.total {
		end = f.total
	}
	for i := start; i < end; i++ {
		result.Items = append(result.Items, notionsync.ItemDTO{ItemID: fmt.Sprint(i)})
	}
	return result, nil
}
func TestAuditReportIncludesEveryPage(t *testing.T) {
	f := &fixtureItemsReader{total: 247}
	items, e := readAllItems(context.Background(), f, "fixture")
	if e != nil || len(items) != 247 || f.calls != 3 || items[246].ItemID != "246" {
		t.Fatalf("report truncated count=%d calls=%d %v", len(items), f.calls, e)
	}
	if _, e := readAllItems(context.Background(), &fixtureItemsReader{total: 4, empty: true}, "fixture"); e == nil {
		t.Fatal("accepted incomplete audit pagination")
	}
}
