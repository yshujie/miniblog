package model

import "time"

const (
	NotionManagementBaselinePending = "baseline_pending"
	NotionManagementManaged         = "managed"
	NotionManagementDetached        = "detached"
	NotionCatalogBound              = "bound"
	NotionCatalogBlocked            = "blocked"
)

// NotionSyncControl is the singleton lease and maintenance gate. Times use the DB clock.
type NotionSyncControl struct {
	ID                 uint64     `gorm:"primaryKey;autoIncrement:false" json:"-"`
	Paused             bool       `json:"paused"`
	SourceWritesPaused bool       `json:"source_writes_paused"`
	BaselineFrozen     bool       `json:"baseline_frozen"`
	BaselineAt         *time.Time `json:"baseline_at,omitempty"`
	LeaseOwner         string     `gorm:"type:varchar(64)" json:"-"`
	LeaseEpoch         uint64     `json:"-"`
	LeaseUntil         *time.Time `json:"-"`
	CurrentRunID       string     `gorm:"type:varchar(64)" json:"current_run_id,omitempty"`
	NextRunAt          *time.Time `json:"next_run_at,omitempty"`
	CooldownUntil      *time.Time `json:"cooldown_until,omitempty"`
	LastCompleteScanAt *time.Time `json:"last_complete_scan_at,omitempty"`
	LastSuccessAt      *time.Time `json:"last_success_at,omitempty"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

func (NotionSyncControl) TableName() string { return "notion_sync_control" }

type NotionSyncSource struct {
	ID                  string     `gorm:"column:source_id;primaryKey;type:varchar(64)" json:"source_id"`
	Label               string     `gorm:"type:varchar(255)" json:"label"`
	DataSourceID        string     `gorm:"type:varchar(36);uniqueIndex:uq_notion_source_data_source" json:"data_source_id"`
	ModuleCode          string     `gorm:"type:varchar(128);index" json:"module_code"`
	PropertyMappingJSON string     `gorm:"type:longtext" json:"-"`
	StatusMappingJSON   string     `gorm:"type:longtext" json:"-"`
	ConfigRevision      uint64     `json:"config_revision"`
	Enabled             bool       `json:"enabled"`
	Health              string     `gorm:"type:varchar(32)" json:"health"`
	LastAttemptAt       *time.Time `json:"last_attempt_at,omitempty"`
	LastCompleteScanAt  *time.Time `json:"last_complete_scan_at,omitempty"`
	LastSuccessAt       *time.Time `json:"last_success_at,omitempty"`
	LastError           string     `gorm:"type:text" json:"last_error,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

func (NotionSyncSource) TableName() string { return "notion_sync_sources" }

type NotionCatalogBinding struct {
	ID              uint64    `gorm:"primaryKey" json:"id,string"`
	SourceID        string    `gorm:"type:varchar(64);index" json:"source_id"`
	DataSourceID    string    `gorm:"type:varchar(36);uniqueIndex:uq_notion_catalog_identity,priority:1" json:"data_source_id"`
	ThemePropertyID string    `gorm:"type:varchar(128);uniqueIndex:uq_notion_catalog_identity,priority:2" json:"theme_property_id"`
	OptionID        string    `gorm:"type:varchar(128);uniqueIndex:uq_notion_catalog_identity,priority:3" json:"option_id"`
	OptionName      string    `gorm:"type:varchar(255)" json:"option_name"`
	SectionCode     *string   `gorm:"type:varchar(128);index" json:"section_code,omitempty"`
	Status          string    `gorm:"type:varchar(32)" json:"status"`
	Reason          string    `gorm:"type:text" json:"reason,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (NotionCatalogBinding) TableName() string { return "notion_catalog_bindings" }

type NotionPageBinding struct {
	PageID                       string     `gorm:"primaryKey;type:varchar(36)" json:"page_id"`
	ArticleID                    *uint64    `gorm:"uniqueIndex:uq_notion_page_article" json:"article_id,omitempty,string"`
	LegacySourceKey              *string    `gorm:"type:char(64);uniqueIndex:uq_notion_page_legacy_source" json:"-"`
	SourceID                     string     `gorm:"type:varchar(64);index" json:"source_id"`
	ManagementState              string     `gorm:"type:varchar(32)" json:"management_state"`
	Revision                     uint64     `json:"revision"`
	DesiredState                 int        `json:"desired_state"`
	SnapshotJSON                 string     `gorm:"type:longtext" json:"-"`
	MetadataHash                 string     `gorm:"type:char(64)" json:"metadata_hash"`
	AppliedHash                  string     `gorm:"type:char(64)" json:"applied_hash"`
	PageURL                      string     `gorm:"type:text" json:"page_url"`
	PublicURL                    *string    `gorm:"type:text" json:"public_url,omitempty"`
	PublishBlockReason           string     `gorm:"type:varchar(128)" json:"publish_block_reason,omitempty"`
	PublicationHeld              bool       `json:"-"`
	PublicationHoldReason        string     `gorm:"type:varchar(255)" json:"-"`
	NeedsRevalidation            bool       `json:"needs_revalidation"`
	NativeArchived               bool       `json:"native_archived"`
	InTrash                      bool       `json:"in_trash"`
	NotionLastEditedAt           *time.Time `json:"notion_last_edited_at,omitempty"`
	LastSeenRunID                string     `gorm:"type:varchar(64)" json:"last_seen_run_id,omitempty"`
	LastApplyRunID               string     `gorm:"type:varchar(64)" json:"last_apply_run_id,omitempty"`
	LastSuccessAt                *time.Time `json:"last_success_at,omitempty"`
	LastError                    string     `gorm:"type:text" json:"last_error,omitempty"`
	BootstrapState               string     `gorm:"type:varchar(32)" json:"bootstrap_state,omitempty"`
	BootstrapExpectedState       *int       `json:"bootstrap_expected_state,omitempty"`
	BootstrapExpectedFingerprint string     `gorm:"type:char(64)" json:"-"`
	ConfirmedBy                  string     `gorm:"type:varchar(128)" json:"confirmed_by,omitempty"`
	ConfirmedAt                  *time.Time `json:"confirmed_at,omitempty"`
	LocalBeforeJSON              string     `gorm:"type:longtext" json:"-"`
	CreatedAt                    time.Time  `json:"created_at"`
	UpdatedAt                    time.Time  `json:"updated_at"`
}

func (NotionPageBinding) TableName() string { return "notion_page_bindings" }

type NotionSyncRun struct {
	ID         string     `gorm:"column:run_id;primaryKey;type:varchar(64)" json:"run_id"`
	Mode       string     `gorm:"type:varchar(32)" json:"mode"`
	Phase      string     `gorm:"type:varchar(32)" json:"phase"`
	Status     string     `gorm:"type:varchar(32)" json:"status"`
	CountsJSON string     `gorm:"type:longtext" json:"counts"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	Error      string     `gorm:"type:text" json:"error,omitempty"`
	LeaseEpoch uint64     `json:"-"`
}

func (NotionSyncRun) TableName() string { return "notion_sync_runs" }

type NotionSyncRunItem struct {
	RunID                string    `gorm:"primaryKey;type:varchar(64)" json:"run_id"`
	ItemID               string    `gorm:"primaryKey;type:varchar(128)" json:"item_id"`
	PageID               string    `gorm:"type:varchar(36);index" json:"page_id"`
	ArticleID            *uint64   `json:"article_id,omitempty,string"`
	Phase                string    `gorm:"type:varchar(32)" json:"phase"`
	Outcome              string    `gorm:"type:varchar(32)" json:"outcome"`
	Reason               string    `gorm:"type:varchar(255)" json:"reason,omitempty"`
	BeforeJSON           string    `gorm:"type:longtext" json:"before,omitempty"`
	AfterJSON            string    `gorm:"type:longtext" json:"after,omitempty"`
	Error                string    `gorm:"type:text" json:"error,omitempty"`
	BootstrapJournalJSON string    `gorm:"type:longtext" json:"-"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

func (NotionSyncRunItem) TableName() string { return "notion_sync_run_items" }
