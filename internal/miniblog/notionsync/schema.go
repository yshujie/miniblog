package notionsync

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

type Snapshot struct {
	PageID          string    `json:"page_id"`
	SourceID        string    `json:"source_id"`
	Title           string    `json:"title"`
	Tags            []string  `json:"tags"`
	TopicOptionID   string    `json:"topic_option_id"`
	TopicOptionName string    `json:"topic_option_name"`
	DesiredState    int       `json:"desired_state"`
	PageURL         string    `json:"page_url"`
	PublicURL       *string   `json:"public_url"`
	NativeArchived  bool      `json:"native_archived"`
	InTrash         bool      `json:"in_trash"`
	LastEditedAt    time.Time `json:"last_edited_at"`
	StateOptionID   string    `json:"state_option_id"`
	Description     string    `json:"description,omitempty"`
	Difficulty      string    `json:"difficulty,omitempty"`
	CreatedDate     string    `json:"created_date,omitempty"`
	Reason          string    `json:"reason,omitempty"`
}

func configComplete(c SourceConfig) bool { return validateConfigShape(c) == nil }
func validateConfigShape(c SourceConfig) error {
	ids := []string{c.TitlePropertyID, c.StatePropertyID, c.TopicPropertyID, c.TagsPropertyID}
	seen := map[string]bool{}
	for _, id := range ids {
		if id == "" || len(id) > 128 {
			return invalid("字段映射不完整或ID过长")
		}
		if seen[propertyID(id)] {
			return invalid("字段映射不能重复")
		}
		seen[propertyID(id)] = true
	}
	for _, id := range []string{c.DescriptionPropertyID, c.DifficultyPropertyID, c.DatePropertyID} {
		if id == "" {
			continue
		}
		if len(id) > 128 || seen[propertyID(id)] {
			return invalid("可选字段ID过长或重复")
		}
		seen[propertyID(id)] = true
	}
	if len(c.StateOptionIDs) != 4 {
		return invalid("博客状态映射须恰好包含四态")
	}
	seen = map[string]bool{}
	for _, state := range []string{"draft", "published", "unpublished", "archived"} {
		id := c.StateOptionIDs[state]
		if id == "" || len(id) > 128 || seen[id] {
			return invalid("四态选项映射不完整或重复")
		}
		seen[id] = true
	}
	return nil
}
func propertyID(id string) string {
	v, e := url.PathUnescape(id)
	if e != nil {
		return id
	}
	return v
}
func schemaProperty(schema source.NotionDataSource, id string) (source.NotionSchemaProperty, bool) {
	for name, p := range schema.Properties {
		if propertyID(p.ID) == propertyID(id) {
			if p.Name == "" {
				p.Name = name
			}
			return p, true
		}
	}
	return source.NotionSchemaProperty{}, false
}
func discoverConfig(schema source.NotionDataSource) (SourceConfig, error) {
	var c SourceConfig
	find := func(name, kind string) (string, error) {
		var ids []string
		for n, p := range schema.Properties {
			if (n == name || p.Name == name) && p.Type == kind {
				ids = append(ids, p.ID)
			}
		}
		if len(ids) != 1 {
			return "", fmt.Errorf("schema field %s must be a unique %s", name, kind)
		}
		return ids[0], nil
	}
	var e error
	if c.TitlePropertyID, e = find("标题", "title"); e != nil {
		return c, e
	}
	if c.StatePropertyID, e = find("博客状态", "select"); e != nil {
		return c, e
	}
	if c.TopicPropertyID, e = find("主题", "select"); e != nil {
		return c, e
	}
	if c.TagsPropertyID, e = find("知识点", "multi_select"); e != nil {
		return c, e
	}
	states := map[string]string{"草稿": "draft", "已发布": "published", "已下架": "unpublished", "归档": "archived"}
	c.StateOptionIDs = map[string]string{}
	p, _ := schemaProperty(schema, c.StatePropertyID)
	for _, option := range p.Select.Options {
		if state, ok := states[option.Name]; ok {
			if c.StateOptionIDs[state] != "" {
				return c, errors.New("duplicate blog state option")
			}
			c.StateOptionIDs[state] = option.ID
		}
	}
	c.DescriptionPropertyID, _ = find("描述", "rich_text")
	c.DifficultyPropertyID, _ = find("难度", "select")
	c.DatePropertyID, _ = find("创建日期", "date")
	return c, validateConfig(schema, c)
}
func validateConfig(schema source.NotionDataSource, c SourceConfig) error {
	if e := validateConfigShape(c); e != nil {
		return e
	}
	if schema.InTrash {
		return errors.New("data source is in trash")
	}
	required := map[string]string{c.TitlePropertyID: "title", c.StatePropertyID: "select", c.TopicPropertyID: "select", c.TagsPropertyID: "multi_select"}
	for id, kind := range required {
		p, ok := schemaProperty(schema, id)
		if !ok || p.Type != kind {
			return fmt.Errorf("configured property %s missing or type changed", id)
		}
	}
	p, _ := schemaProperty(schema, c.StatePropertyID)
	options := map[string]bool{}
	for _, o := range p.Select.Options {
		options[o.ID] = true
	}
	for _, id := range c.StateOptionIDs {
		if !options[id] {
			return errors.New("configured blog state option disappeared")
		}
	}
	optional := map[string]string{c.DescriptionPropertyID: "rich_text", c.DifficultyPropertyID: "select", c.DatePropertyID: "date"}
	for id, kind := range optional {
		if id == "" {
			continue
		}
		p, ok := schemaProperty(schema, id)
		if !ok || p.Type != kind {
			return errors.New("optional property type changed")
		}
	}
	return nil
}
func pageProperty(page source.NotionPage, id string) (source.NotionProperty, bool) {
	for _, p := range page.Properties {
		if propertyID(p.ID) == propertyID(id) {
			return p, true
		}
	}
	return source.NotionProperty{}, false
}
func richText(parts []source.NotionRichText) string {
	var text strings.Builder
	for _, part := range parts {
		if part.PlainText != "" {
			text.WriteString(part.PlainText)
		} else {
			text.WriteString(part.Text.Content)
		}
	}
	return strings.TrimSpace(text.String())
}
func snapshotOf(page source.NotionPage, srcID string, c SourceConfig) (Snapshot, error) {
	identity, e := source.NotionIdentity(page.ID)
	if e != nil {
		return Snapshot{}, e
	}
	snap := Snapshot{PageID: normalizePageID(identity.PageID), SourceID: srcID, PageURL: page.URL, PublicURL: page.PublicURL, NativeArchived: page.IsArchived, InTrash: page.InTrash, LastEditedAt: page.LastEditedTime, Tags: []string{}}
	if page.Object != "" && page.Object != "page" {
		return snap, errors.New("unexpected non-page result")
	}
	if page.Parent.Type != "data_source_id" || normalizeID(page.Parent.DataSourceID) != srcID {
		return snap, errors.New("page parent does not match scanned data source")
	}
	// Parse the authoritative state before optional metadata: a broken title may not block a confirmed withdrawal.
	state, ok := pageProperty(page, c.StatePropertyID)
	if !ok || state.Type != "select" {
		snap.Reason = "unknown_state"
		return snap, errors.New("page blog state property unavailable")
	}
	if state.Select != nil {
		snap.StateOptionID = state.Select.ID
		for _, name := range []string{"draft", "published", "unpublished", "archived"} {
			if c.StateOptionIDs[name] == snap.StateOptionID {
				snap.DesiredState = stateValue(name)
			}
		}
	}
	if snap.DesiredState == 0 {
		snap.Reason = "unknown_state"
	}
	var fieldErrors []string
	title, ok := pageProperty(page, c.TitlePropertyID)
	if !ok || title.Type != "title" {
		fieldErrors = append(fieldErrors, "title property unavailable")
	} else {
		snap.Title = richText(title.Title)
		if snap.Title == "" || utf8.RuneCountInString(snap.Title) > 255 {
			fieldErrors = append(fieldErrors, "invalid title")
		}
	}
	topic, ok := pageProperty(page, c.TopicPropertyID)
	if !ok || topic.Type != "select" {
		fieldErrors = append(fieldErrors, "topic property unavailable")
	} else if topic.Select != nil {
		snap.TopicOptionID = topic.Select.ID
		snap.TopicOptionName = topic.Select.Name
	}
	if snap.TopicOptionID == "" && ok && topic.Type == "select" {
		snap.TopicOptionName = "未分类"
	}
	tags, ok := pageProperty(page, c.TagsPropertyID)
	if !ok || tags.Type != "multi_select" {
		fieldErrors = append(fieldErrors, "tags property unavailable")
	} else {
		for _, tag := range tags.MultiSelect {
			snap.Tags = append(snap.Tags, tag.Name)
		}
		sort.Strings(snap.Tags)
		_, tagErr := model.EncodeArticleTags(snap.Tags)
		if tagErr != nil {
			fieldErrors = append(fieldErrors, "tags exceed 64 KiB")
		}
	}
	if p, ok := pageProperty(page, c.DescriptionPropertyID); ok && c.DescriptionPropertyID != "" {
		snap.Description = richText(p.RichText)
	}
	if p, ok := pageProperty(page, c.DifficultyPropertyID); ok && p.Select != nil && c.DifficultyPropertyID != "" {
		snap.Difficulty = p.Select.Name
	}
	if p, ok := pageProperty(page, c.DatePropertyID); ok && p.Date != nil && c.DatePropertyID != "" {
		snap.CreatedDate = p.Date.Start
	}
	if len(fieldErrors) > 0 {
		if snap.DesiredState != 0 {
			snap.Reason = "invalid_field"
		}
		return snap, errors.New(strings.Join(fieldErrors, "; "))
	}
	return snap, nil
}
func metadataHash(snap Snapshot) string {
	snap.LastEditedAt = time.Time{}
	sum := sha256.Sum256([]byte(jsonText(snap)))
	return hex.EncodeToString(sum[:])
}
