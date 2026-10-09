// Package notionsync owns Notion scanning and takeover. It never reads article bodies.
package notionsync

import (
	"context"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"time"
)

type Options struct {
	Enabled           bool
	Interval          time.Duration
	Author            string
	Client            source.SyncNotionClient
	OwnerID           string
	LeaseDuration     time.Duration
	HeartbeatInterval time.Duration
}
type ListQuery struct {
	Page  int `json:"page" form:"page"`
	Limit int `json:"limit" form:"limit"`
}
type PageQuery struct {
	ListQuery
	SourceID        string `json:"source_id" form:"source_id"`
	ManagementState string `json:"management_state" form:"management_state"`
	Status          string `json:"status" form:"status"`
	Title           string `json:"title" form:"title"`
}
type PageResult[T any] struct {
	Items []T   `json:"items"`
	Total int64 `json:"total"`
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
}
type SourceConfig struct {
	TitlePropertyID       string            `json:"title_property_id"`
	StatePropertyID       string            `json:"state_property_id"`
	TopicPropertyID       string            `json:"topic_property_id"`
	TagsPropertyID        string            `json:"tags_property_id"`
	StateOptionIDs        map[string]string `json:"state_option_ids"`
	DescriptionPropertyID string            `json:"description_property_id,omitempty"`
	DifficultyPropertyID  string            `json:"difficulty_property_id,omitempty"`
	DatePropertyID        string            `json:"date_property_id,omitempty"`
}
type SourceInput struct {
	Label                  string        `json:"label"`
	ModuleCode             string        `json:"module_code"`
	Enabled                *bool         `json:"enabled,omitempty"`
	Config                 *SourceConfig `json:"config,omitempty"`
	ExpectedConfigRevision *uint64       `json:"expected_config_revision,omitempty"`
}
type ControlInput struct {
	Enabled            *bool `json:"enabled,omitempty"`
	Paused             *bool `json:"paused,omitempty"`
	SourceWritesPaused *bool `json:"source_writes_paused,omitempty"`
}
type BindingInput struct {
	BindingID              string  `json:"-"`
	SourceID               string  `json:"source_id"`
	OptionID               string  `json:"option_id"`
	SectionCode            string  `json:"section_code"`
	ExpectedConfigRevision *uint64 `json:"expected_config_revision,omitempty"`
}
type TriggerInput struct {
	Mode string `json:"mode"`
}
type TriggerResult struct {
	RunID string `json:"run_id"`
}
type TopicBindingDTO struct {
	ID          string `json:"id"`
	SourceID    string `json:"source_id"`
	OptionID    string `json:"option_id"`
	OptionName  string `json:"option_name"`
	SectionCode string `json:"section_code,omitempty"`
	Status      string `json:"status"`
	Reason      string `json:"reason,omitempty"`
}
type SourceDTO struct {
	SourceID           string            `json:"source_id"`
	Label              string            `json:"label"`
	ModuleCode         string            `json:"module_code"`
	Enabled            bool              `json:"enabled"`
	ConfigRevision     uint64            `json:"config_revision"`
	Config             SourceConfig      `json:"config"`
	Health             string            `json:"health"`
	LastAttemptAt      *time.Time        `json:"last_attempt_at,omitempty"`
	LastCompleteScanAt *time.Time        `json:"last_complete_scan_at,omitempty"`
	LastSuccessAt      *time.Time        `json:"last_success_at,omitempty"`
	LastError          string            `json:"last_error,omitempty"`
	CatalogBindings    []TopicBindingDTO `json:"catalog_bindings"`
}
type PageDTO struct {
	LocalState            *string `json:"local_state"`
	NeedsRevalidation     bool    `json:"needs_revalidation"`
	EffectiveVisibility   bool    `json:"effective_visibility"`
	VisibilityReason      string  `json:"visibility_reason"`
	PageID                string  `json:"page_id"`
	ArticleID             *string `json:"article_id,omitempty"`
	SourceID              string  `json:"source_id"`
	Title                 string  `json:"title"`
	DesiredState          string  `json:"desired_state"`
	Status                string  `json:"status"`
	ManagementState       string  `json:"management_state"`
	PublishBlockReason    string  `json:"publish_block_reason,omitempty"`
	LastError             string  `json:"last_error,omitempty"`
	PageURL               string  `json:"page_url"`
	PublicURL             *string `json:"public_url"`
	PublicationHold       bool    `json:"publication_hold"`
	PublicationHoldReason string  `json:"publication_hold_reason,omitempty"`
	BootstrapState        string  `json:"bootstrap_state,omitempty"`
	Revision              uint64  `json:"revision"`
}
type RunCounts struct {
	Pending   int `json:"pending"`
	Frozen    int `json:"frozen"`
	Seen      int `json:"seen"`
	Created   int `json:"created"`
	Updated   int `json:"updated"`
	Unchanged int `json:"unchanged"`
	Blocked   int `json:"blocked"`
	Failed    int `json:"failed"`
}
type RunDTO struct {
	RunID      string     `json:"run_id"`
	Mode       string     `json:"mode"`
	Status     string     `json:"status"`
	Phase      string     `json:"phase"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	Counts     RunCounts  `json:"counts"`
	Error      string     `json:"error,omitempty"`
}
type ItemDTO struct {
	ItemID    string      `json:"item_id,omitempty"`
	PageID    string      `json:"page_id"`
	ArticleID *string     `json:"article_id,omitempty"`
	Outcome   string      `json:"outcome"`
	Reason    string      `json:"reason,omitempty"`
	Before    interface{} `json:"before,omitempty"`
	After     interface{} `json:"after,omitempty"`
	Error     string      `json:"error,omitempty"`
}
type StatusDTO struct {
	LastAttemptAt      *time.Time  `json:"last_attempt_at,omitempty"`
	Enabled            bool        `json:"enabled"`
	Paused             bool        `json:"paused"`
	SourceWritesPaused bool        `json:"source_writes_paused"`
	BaselineFrozen     bool        `json:"baseline_frozen"`
	CurrentRunID       *string     `json:"current_run_id,omitempty"`
	LastCompleteScanAt *time.Time  `json:"last_complete_scan_at,omitempty"`
	LastSuccessAt      *time.Time  `json:"last_success_at,omitempty"`
	Health             string      `json:"health"`
	PendingCount       int64       `json:"pending_count"`
	ErrorCount         int64       `json:"error_count"`
	BlockedCount       int64       `json:"blocked_count"`
	Sources            []SourceDTO `json:"sources"`
}
type BootstrapConfirm struct {
	NewPage             bool   `json:"new_page,omitempty"`
	AllowLegacyAlias    bool   `json:"allow_legacy_alias,omitempty"`
	ExpectedFingerprint string `json:"expected_fingerprint"`
	PageID              string `json:"page_id"`
	ArticleID           string `json:"article_id"`
	ExpectedState       string `json:"expected_state"`
	ConfirmedBy         string `json:"confirmed_by"`
}
type BootstrapInput struct {
	SourceID               string             `json:"source_id,omitempty"`
	ExpectedConfigRevision uint64             `json:"expected_config_revision,omitempty"`
	Confirmations          []BootstrapConfirm `json:"confirmations"`
}

type Error struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
}

func (e *Error) Error() string   { return e.Message }
func (e *Error) StatusCode() int { return e.HTTPStatus }
func invalid(message string) error {
	return &Error{Code: "invalid_request", Message: message, HTTPStatus: 400}
}
func conflict(message string) error {
	return &Error{Code: "conflict", Message: message, HTTPStatus: 409}
}
func unavailable(message string) error {
	return &Error{Code: "not_configured", Message: message, HTTPStatus: 503}
}

// API is exported for HTTP adapters and fixture tests; bootstrap is deliberately separate.
type API interface {
	Status(context.Context) (*StatusDTO, error)
	Sources(context.Context, ListQuery) (*PageResult[SourceDTO], error)
	Pages(context.Context, PageQuery) (*PageResult[PageDTO], error)
	Runs(context.Context, ListQuery) (*PageResult[RunDTO], error)
	Run(context.Context, string) (*RunDTO, error)
	Items(context.Context, string, ListQuery) (*PageResult[ItemDTO], error)
	UpdateControl(context.Context, ControlInput) (*StatusDTO, error)
	UpdateSource(context.Context, string, SourceInput) (*SourceDTO, error)
	BindCatalog(context.Context, BindingInput) (*TopicBindingDTO, error)
	Trigger(context.Context, TriggerInput) (*TriggerResult, error)
}

type BootstrapCandidate struct {
	RequiresLegacyAlias  bool     `json:"requires_legacy_alias"`
	MatchMethod          string   `json:"match_method"`
	NewPage              bool     `json:"new_page"`
	PublicCondition      string   `json:"public_condition"`
	ProposedSectionCode  string   `json:"proposed_section_code,omitempty"`
	ProposedSectionTitle string   `json:"proposed_section_title"`
	PublishBlockReason   string   `json:"publish_block_reason,omitempty"`
	PlacementChange      string   `json:"placement_change"`
	TitleHintArticleIDs  []string `json:"title_hint_article_ids"`
	PageID               string   `json:"page_id"`
	SourceID             string   `json:"source_id"`
	Title                string   `json:"title"`
	Topic                string   `json:"topic"`
	NotionState          string   `json:"notion_state"`
	CandidateArticleIDs  []string `json:"candidate_article_ids"`
	ArticleID            string   `json:"article_id,omitempty"`
	LocalState           string   `json:"local_state,omitempty"`
	LocalTitle           string   `json:"local_title,omitempty"`
	LocalSectionCode     string   `json:"local_section_code,omitempty"`
	LocalSubsectionCode  string   `json:"local_subsection_code,omitempty"`
	ExpectedFingerprint  string   `json:"expected_fingerprint,omitempty"`
	Reason               string   `json:"reason,omitempty"`
}
type BootstrapPreviewResult struct {
	RunID string               `json:"run_id"`
	Items []BootstrapCandidate `json:"items"`
}
type BootstrapApplyResult struct {
	RunID string    `json:"run_id"`
	Items []ItemDTO `json:"items"`
}

type BootstrapManualMatch struct {
	PageID    string `json:"page_id"`
	ArticleID string `json:"article_id"`
}
type BootstrapReviewInput struct {
	ManualMatches []BootstrapManualMatch `json:"manual_matches"`
}
