package article

import (
	"context"
	"github.com/yshujie/miniblog/internal/miniblog/biz/catalog"
	"github.com/yshujie/miniblog/internal/miniblog/store"
)

// EnsureSyncedTopics retains the article facade while catalog owns its transaction.
func (b *articleBiz) EnsureSyncedTopics(ctx context.Context, token store.LeaseToken, sourceID string, expectedRevision uint64, propertyID string, options []TopicOption) ([]TopicResult, error) {
	return catalog.New(b.ds).EnsureSyncedTopics(ctx, token, sourceID, expectedRevision, propertyID, options)
}

// PrepareSyncedTopics retains the maintenance contract for existing callers.
func (b *articleBiz) PrepareSyncedTopics(ctx context.Context, token store.LeaseToken, sourceID string, expectedRevision uint64, propertyID string, options []TopicOption) ([]TopicResult, uint64, error) {
	return catalog.New(b.ds).PrepareSyncedTopics(ctx, token, sourceID, expectedRevision, propertyID, options)
}
