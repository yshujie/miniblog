package article

import (
	"errors"
	"strings"

	"github.com/yshujie/miniblog/internal/miniblog/biz/catalog"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	"gorm.io/gorm"
)

// ReviewedPublicationInput contains successfully read metadata plus the local hold
// and retained-subsection context. It deliberately does not require managed state
// or cleared revalidation: adoption has not yet granted management authority.
type ReviewedPublicationInput struct {
	Title               string   `json:"title"`
	Tags                []string `json:"tags"`
	TopicOptionID       string   `json:"topic_option_id"`
	TopicOptionName     string   `json:"topic_option_name"`
	TopicPropertyID     string   `json:"-"`
	PublicURL           *string  `json:"public_url"`
	NativeArchived      bool     `json:"native_archived"`
	InTrash             bool     `json:"in_trash"`
	PublicationHeld     bool     `json:"-"`
	LocalSectionCode    string   `json:"-"`
	LocalSubsectionCode string   `json:"-"`
}

// ReviewedPublicationReason borrows the caller's store and returns a stable public
// gate reason. Database failures remain errors instead of being treated as empty data.
func ReviewedPublicationReason(ds store.IStore, src *model.NotionSyncSource, r ReviewedPublicationInput) (string, error) {
	if !src.Enabled {
		return "source_disabled", nil
	}
	if src.ModuleCode == "" {
		return "source_module_unconfigured", nil
	}
	if r.NativeArchived {
		return "native_archived", nil
	}
	if r.InTrash {
		return "in_trash", nil
	}
	if r.PublicationHeld {
		return "publication_hold", nil
	}
	if r.PublicURL == nil || strings.TrimSpace(*r.PublicURL) == "" {
		return "public_url_missing", nil
	}
	if _, err := source.Parse(*r.PublicURL); err != nil {
		return "public_url_invalid", nil
	}
	if err := validateFields(r.Title, "", r.Tags); err != nil {
		return "metadata_invalid", nil
	}
	if r.TopicPropertyID == "" {
		return "theme_binding_invalid", nil
	}
	var binding model.NotionCatalogBinding
	err := ds.DB().Where("data_source_id = ? AND theme_property_id = ? AND option_id = ?", src.DataSourceID, r.TopicPropertyID, r.TopicOptionID).First(&binding).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "theme_binding_invalid", nil
	}
	if err != nil {
		return "", err
	}
	if binding.SourceID != src.ID || binding.Status != model.NotionCatalogBound || binding.SectionCode == nil || binding.OptionName != r.TopicOptionName {
		return "theme_binding_invalid", nil
	}
	placement, err := catalog.Resolve(ds, *binding.SectionCode, "")
	if err != nil {
		var missing *errno.Errno
		if errors.As(err, &missing) {
			return "catalog_invalid", nil
		}
		return "", err
	}
	module, err := ds.Modules().GetByCode(src.ModuleCode)
	if err != nil {
		return "", err
	}
	if module == nil || placement.Module.ID != module.ID {
		return "catalog_invalid", nil
	}
	if err = catalog.CheckSyncedThemeSectionAvailable(ds, placement.Section.Code, binding.ID); err != nil {
		var conflict *errno.Errno
		if errors.As(err, &conflict) {
			return "theme_binding_conflict", nil
		}
		return "", err
	}
	if r.LocalSubsectionCode != "" && r.LocalSectionCode != "" {
		local, err := ds.Sections().GetByCode(r.LocalSectionCode)
		if err != nil {
			return "", err
		}
		if local != nil && local.ID == placement.Section.ID {
			placement, err = catalog.Resolve(ds, placement.Section.Code, r.LocalSubsectionCode)
			if err != nil {
				var missing *errno.Errno
				if errors.As(err, &missing) {
					return "catalog_invalid", nil
				}
				return "", err
			}
		}
	}
	if err = catalog.RequireVisible(placement); err != nil {
		return "catalog_disabled", nil
	}
	return "", nil
}
func ValidateReviewedPublication(ds store.IStore, src *model.NotionSyncSource, r ReviewedPublicationInput) error {
	reason, err := ReviewedPublicationReason(ds, src, r)
	if err != nil {
		return err
	}
	if reason != "" {
		return syncConflict("已发布接管条件未满足: " + reason)
	}
	return nil
}
