package article

import (
	"github.com/yshujie/miniblog/internal/miniblog/biz/catalog"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"time"
)

// SyncInput contains trusted, validated page metadata, never body, author or order.
type SyncInput struct {
	Lease                                                                                  store.LeaseToken
	RunID, SourceID, DataSourceID, PageID, ThemePropertyID, ThemeOptionID, ThemeOptionName string
	Title, PageURL, MetadataHash, SnapshotJSON, MetadataError                              string
	Tags                                                                                   []string
	PublicURLObserved                                                                      bool
	PublicURL                                                                              *string
	DesiredState                                                                           int
	NativeArchived, InTrash, MetadataComplete                                              bool
	NotionLastEditedAt                                                                     *time.Time
	ExpectedConfigRevision, ExpectedBindingRevision                                        uint64
}
type SyncResult struct {
	ArticleID               uint64
	PageID, Outcome, Reason string
	AppliedState            int
	BindingRevision         uint64
}
type LocalPatchInput struct{ Author string }
type HoldInput struct {
	Held   bool
	Reason string
}
type AdoptInput struct {
	Lease                                      store.LeaseToken
	PageID, SourceID, ExpectedFingerprint      string
	ExpectedConfigRevision                     uint64
	ExpectedArticleID, ExpectedBindingRevision uint64
}

type TopicOption = catalog.TopicOption
type TopicResult = catalog.TopicResult
