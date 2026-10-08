package v1

import (
	"encoding/json"
	"testing"
)

func TestArticleIDContracts(t *testing.T) {
	const id uint64 = 9007199254740993
	old := &ArticleInfo{ID: id, IDText: "9007199254740993"}
	legacy, err := json.Marshal(old)
	if err != nil {
		t.Fatal(err)
	}
	var l map[string]json.RawMessage
	if json.Unmarshal(legacy, &l) != nil {
		t.Fatal("JSON")
	}
	if string(l["id"]) != "9007199254740993" || string(l["id_text"]) != "\"9007199254740993\"" {
		t.Fatal(string(legacy))
	}
	command, err := json.Marshal(ArticleDTO{ArticleInfo: old, ID: old.IDText})
	if err != nil {
		t.Fatal(err)
	}
	var c map[string]json.RawMessage
	json.Unmarshal(command, &c)
	if string(c["id"]) != "\"9007199254740993\"" {
		t.Fatal(string(command))
	}
}
func TestUpdateContentPresence(t *testing.T) {
	var omitted, explicit UpdateArticleRequest
	json.Unmarshal([]byte(`{"title":"T"}`), &omitted)
	json.Unmarshal([]byte(`{"title":"T","content":""}`), &explicit)
	if omitted.Content != nil || explicit.Content == nil || *explicit.Content != "" {
		t.Fatal("content omission contract")
	}
}
