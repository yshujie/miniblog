package model

import "testing"

func TestArticleBeforeCreatePreservesExplicitID(t *testing.T) {
	const explicit uint64 = 9007199254740993
	article := Article{ID: explicit}
	if err := article.BeforeCreate(nil); err != nil {
		t.Fatal(err)
	}
	if article.ID != explicit {
		t.Fatalf("explicit ID replaced: %d", article.ID)
	}
}
func TestArticleBeforeCreateGeneratesOnlyZeroID(t *testing.T) {
	article := Article{}
	if err := article.BeforeCreate(nil); err != nil {
		t.Fatal(err)
	}
	if article.ID == 0 {
		t.Fatal("zero ID was not generated")
	}
	generated := article.ID
	if err := article.BeforeCreate(nil); err != nil {
		t.Fatal(err)
	}
	if article.ID != generated {
		t.Fatal("second hook replaced generated ID")
	}
}
