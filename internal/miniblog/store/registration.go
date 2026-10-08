package store

import (
	"context"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"gorm.io/gorm"
	"strings"
)

// RegistrationReady is checked at the HTTP boundary. New source writers require
// both the unique constraint and completed legacy backfill before cutover.
func RegistrationReady(ctx context.Context, db *gorm.DB) (bool, error) {
	if db == nil {
		return false, nil
	}
	indexes, err := db.WithContext(ctx).Migrator().GetIndexes(&model.Article{})
	if err != nil {
		return false, err
	}
	unique := false
	for _, index := range indexes {
		value, known := index.Unique()
		columns := index.Columns()
		if index.Name() == "uq_article_source_key" && known && value && len(columns) == 1 && strings.EqualFold(columns[0], "source_key") {
			unique = true
		}
	}
	if !unique {
		return false, nil
	}
	var missing int64
	err = db.WithContext(ctx).Model(&model.Article{}).
		Where("TRIM(COALESCE(external_link, '')) <> ''").
		Where("TRIM(COALESCE(provider, '')) = '' OR TRIM(COALESCE(canonical_url, '')) = '' OR TRIM(COALESCE(source_key, '')) = ''").
		Count(&missing).Error
	return missing == 0, err
}
