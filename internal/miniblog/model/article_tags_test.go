package model

import (
	"strings"
	"testing"
)

func TestArticleTagsPreserveCommaAndLongLabels(t *testing.T) {
	a := &Article{Tags: "legacy"}
	tags := []string{"A,B", strings.Repeat("中", 300)}
	if e := SetArticleTags(a, tags); e != nil {
		t.Fatal(e)
	}
	got, e := ArticleTags(a)
	if e != nil || len(got) != 2 || got[0] != "A,B" || got[1] != tags[1] {
		t.Fatalf("%v %v", got, e)
	}
	if a.Tags != "legacy" {
		t.Fatal("unrepresentable CSV shadow was overwritten")
	}
	if e = SetArticleTags(a, []string{}); e != nil {
		t.Fatal(e)
	}
	got, e = ArticleTags(a)
	if e != nil || got == nil || len(got) != 0 || a.Tags != "" {
		t.Fatalf("empty array: %v %v", got, e)
	}
}
func TestArticleTagsLegacyAndCorruption(t *testing.T) {
	a := &Article{Tags: "one,two"}
	got, e := ArticleTags(a)
	if e != nil || len(got) != 2 {
		t.Fatal(got, e)
	}
	bad := "not json"
	a.TagsJSON = &bad
	if _, e = ArticleTags(a); e == nil {
		t.Fatal("corruption masked by CSV")
	}
	null := "null"
	a.TagsJSON = &null
	if _, e = ArticleTags(a); e == nil {
		t.Fatal("null JSON accepted")
	}
	if _, e = EncodeArticleTags([]string{strings.Repeat("x", MaxArticleTagsJSONBytes)}); e == nil {
		t.Fatal("oversize accepted")
	}
}

func TestArticleTagsSingleEmptyLabelKeepsCSVShadow(t *testing.T) {
	a := &Article{Tags: "legacy"}
	if e := SetArticleTags(a, []string{""}); e != nil {
		t.Fatal(e)
	}
	got, e := ArticleTags(a)
	if e != nil || len(got) != 1 || got[0] != "" || a.Tags != "legacy" {
		t.Fatal(got, a.Tags, e)
	}
	if e = SetArticleTags(a, []string{"", "x", ""}); e != nil {
		t.Fatal(e)
	}
	if a.Tags != ",x," {
		t.Fatal(a.Tags)
	}
}
