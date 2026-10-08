package model

import (
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"
)

const MaxArticleTagsJSONBytes = 64 * 1024

func EncodeArticleTags(tags []string) (string, error) {
	if tags == nil {
		tags = []string{}
	}
	raw, e := json.Marshal(tags)
	if e != nil {
		return "", e
	}
	if len(raw) > MaxArticleTagsJSONBytes {
		return "", errors.New("标签JSON超过64KiB")
	}
	return string(raw), nil
}

// ArticleTags reads JSON when present. A corrupt JSON column never falls back to stale CSV.
func ArticleTags(a *Article) ([]string, error) {
	if a.TagsJSON != nil {
		var tags []string
		if len(*a.TagsJSON) > MaxArticleTagsJSONBytes {
			return nil, errors.New("标签JSON超过64KiB")
		}
		if e := json.Unmarshal([]byte(*a.TagsJSON), &tags); e != nil {
			return nil, e
		}
		if tags == nil {
			return nil, errors.New("标签JSON必须是数组")
		}
		return tags, nil
	}
	if a.Tags == "" {
		return []string{}, nil
	}
	return strings.Split(a.Tags, ","), nil
}
func SetArticleTags(a *Article, tags []string) error {
	raw, e := EncodeArticleTags(tags)
	if e != nil {
		return e
	}
	a.TagsJSON = &raw
	csv := strings.Join(tags, ",")
	representable := utf8.RuneCountInString(csv) <= 255
	for _, t := range tags {
		if strings.Contains(t, ",") {
			representable = false
		}
	}
	roundtrip := []string{}
	if csv != "" {
		roundtrip = strings.Split(csv, ",")
	}
	if len(roundtrip) != len(tags) {
		representable = false
	} else {
		for i := range roundtrip {
			if roundtrip[i] != tags[i] {
				representable = false
			}
		}
	}
	if representable {
		a.Tags = csv
	}
	return nil
}
func ArticleTagsEqual(a *Article, tags []string) (bool, error) {
	old, e := ArticleTags(a)
	if e != nil {
		return false, e
	}
	if len(old) != len(tags) {
		return false, nil
	}
	for i := range old {
		if old[i] != tags[i] {
			return false, nil
		}
	}
	return true, nil
}
