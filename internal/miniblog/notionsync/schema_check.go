package notionsync

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/yshujie/miniblog/internal/miniblog/source"
)

type SchemaPropertyDTO struct {
	ID      string                `json:"id"`
	Name    string                `json:"name"`
	Type    string                `json:"type"`
	Options []source.NotionOption `json:"options,omitempty"`
}

type SchemaSourceDTO struct {
	SourceID           string              `json:"source_id"`
	ActualDataSourceID string              `json:"actual_data_source_id,omitempty"`
	Label              string              `json:"label"`
	Status             string              `json:"status"`
	Config             *SourceConfig       `json:"suggested_config,omitempty"`
	Properties         []SchemaPropertyDTO `json:"properties"`
	Error              string              `json:"error,omitempty"`
}

type SchemaCheckResult struct {
	NotionVersion string            `json:"notion_version"`
	Complete      bool              `json:"complete"`
	Sources       []SchemaSourceDTO `json:"sources"`
}

// CheckSchemas is intentionally independent of Service/Store: it only retrieves
// the five allowlisted schemas. It never opens a DB, persists mappings, queries
// rows/pages, or installs the leased DB cooldown callback used by runtime sync.
func CheckSchemas(ctx context.Context, client source.SyncNotionClient) (*SchemaCheckResult, error) {
	result := &SchemaCheckResult{NotionVersion: source.NotionSyncVersion, Complete: true, Sources: []SchemaSourceDTO{}}
	for _, id := range AllowedSources() {
		item := SchemaSourceDTO{SourceID: id, Label: allowedSources[id], Status: "not_checked", Properties: []SchemaPropertyDTO{}}
		if ctx.Err() != nil || client == nil {
			item.Error = "schema read unavailable"
			result.Complete = false
			result.Sources = append(result.Sources, item)
			continue
		}
		schema, err := client.RetrieveDataSource(ctx, id)
		// Preserve the returned identity even when the client rejects a mismatch.
		// Only a UUID may be emitted, never an arbitrary error or response string.
		if _, identityErr := source.NotionIdentity(schema.ID); identityErr == nil {
			item.ActualDataSourceID = normalizeID(schema.ID)
		}
		if err != nil {
			item.Status, item.Error = "unavailable", schemaReadError(err)
			result.Complete = false
		} else {
			for name, property := range schema.Properties {
				if property.Name != "" {
					name = property.Name
				}
				value := SchemaPropertyDTO{ID: property.ID, Name: name, Type: property.Type}
				if property.Type == "select" {
					value.Options = property.Select.Options
				}
				if property.Type == "multi_select" {
					value.Options = property.MultiSelect.Options
				}
				item.Properties = append(item.Properties, value)
			}
			sort.Slice(item.Properties, func(i, j int) bool { return item.Properties[i].ID < item.Properties[j].ID })
			err = validateSchemaIdentity(schema, id)
			var cfg SourceConfig
			if err == nil {
				cfg, err = discoverConfig(schema)
			}
			if err != nil {
				item.Status, item.Error = "invalid", err.Error()
				result.Complete = false
			} else {
				item.Status, item.Config = "valid", &cfg
			}
		}
		result.Sources = append(result.Sources, item)
	}
	if !result.Complete {
		return result, invalid("五库 schema 检查未全部通过，请查看完整报告")
	}
	return result, nil
}

// Property IDs are opaque and may be percent-encoded. Check identity, presence,
// uniqueness and storage limits rather than imposing an invented ASCII format.
func validateSchemaIdentity(schema source.NotionDataSource, expectedSourceID string) error {
	if normalizeID(schema.ID) != expectedSourceID {
		return invalid("数据源 schema 身份不一致")
	}
	if schema.InTrash {
		return invalid("数据源已在垃圾箱")
	}
	seen := map[string]bool{}
	for _, property := range schema.Properties {
		id := propertyID(property.ID)
		if strings.TrimSpace(id) == "" || len(property.ID) > 128 || seen[id] {
			return invalid("schema 属性ID为空、重复或超出长度限制")
		}
		seen[id] = true
		var options []source.NotionOption
		if property.Type == "select" {
			options = property.Select.Options
		}
		if property.Type == "multi_select" {
			options = property.MultiSelect.Options
		}
		optionIDs := map[string]bool{}
		for _, option := range options {
			if option.ID == "" || len(option.ID) > 128 || optionIDs[option.ID] {
				return invalid(fmt.Sprintf("schema %s 选项ID为空、重复或超出长度限制", property.ID))
			}
			optionIDs[option.ID] = true
		}
	}
	return nil
}

// A read failure may wrap a driver/transport error containing credentials or
// environment values. Reports expose only stable categories and HTTP status.
func schemaReadError(err error) string {
	if errors.Is(err, context.Canceled) {
		return "schema read cancelled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "schema read timed out"
	}
	var api *source.NotionAPIError
	if errors.As(err, &api) {
		if api.StatusCode >= 100 && api.StatusCode <= 599 {
			return fmt.Sprintf("schema read unavailable (HTTP %d)", api.StatusCode)
		}
		if api.Code == "notion_not_configured" {
			return "schema read connection not configured"
		}
	}
	return "schema read unavailable"
}
