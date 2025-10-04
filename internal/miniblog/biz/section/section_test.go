package section

import (
	"context"
	"errors"
	"testing"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"gorm.io/gorm"
)

type fakeSectionStore struct {
	sections map[string]*model.Section
}

func newFakeSectionStore() *fakeSectionStore {
	return &fakeSectionStore{sections: make(map[string]*model.Section)}
}

func (f *fakeSectionStore) Create(section *model.Section) error {
	if _, exists := f.sections[section.Code]; exists {
		return errors.New("duplicate section")
	}
	copy := *section
	f.sections[section.Code] = &copy
	return nil
}

func (f *fakeSectionStore) GetByCode(code string) (*model.Section, error) {
	if section, ok := f.sections[code]; ok {
		copy := *section
		return &copy, nil
	}
	return nil, nil
}

func (f *fakeSectionStore) GetSections(moduleCode string) ([]*model.Section, error) {
	var result []*model.Section
	for _, section := range f.sections {
		if section.ModuleCode == moduleCode {
			copy := *section
			result = append(result, &copy)
		}
	}
	return result, nil
}

func (f *fakeSectionStore) GetNormalSections(moduleCode string) ([]*model.Section, error) {
	var result []*model.Section
	for _, section := range f.sections {
		if section.ModuleCode == moduleCode && section.Status == model.SectionStatusNormal {
			copy := *section
			result = append(result, &copy)
		}
	}
	return result, nil
}

func (f *fakeSectionStore) Update(section *model.Section) error {
	if _, ok := f.sections[section.Code]; !ok {
		return errors.New("section not found")
	}
	copy := *section
	f.sections[section.Code] = &copy
	return nil
}

type fakeModuleStore struct {
	modules map[string]*model.Module
}

func (f *fakeModuleStore) Create(module *model.Module) error {
	if f.modules == nil {
		f.modules = make(map[string]*model.Module)
	}
	copy := *module
	f.modules[module.Code] = &copy
	return nil
}

func (f *fakeModuleStore) GetByCode(code string) (*model.Module, error) {
	if module, ok := f.modules[code]; ok {
		copy := *module
		return &copy, nil
	}
	return nil, nil
}
func (f *fakeModuleStore) GetAll() ([]*model.Module, error)           { return nil, nil }
func (f *fakeModuleStore) GetNormalModules() ([]*model.Module, error) { return nil, nil }
func (f *fakeModuleStore) Update(_ *model.Module) error               { return nil }

type fakeStore struct {
	sections *fakeSectionStore
	modules  *fakeModuleStore
}

func (f *fakeStore) DB() *gorm.DB                 { return nil }
func (f *fakeStore) Users() store.UserStore       { return nil }
func (f *fakeStore) Modules() store.ModuleStore   { return f.modules }
func (f *fakeStore) Sections() store.SectionStore { return f.sections }
func (f *fakeStore) Articles() store.ArticleStore { return nil }

func TestSectionBizPublish(t *testing.T) {
	sections := newFakeSectionStore()
	sections.sections["s1"] = &model.Section{Code: "s1", ModuleCode: "moduleA", Status: model.SectionStatusDeleted}
	biz := sectionBiz{ds: &fakeStore{sections: sections, modules: &fakeModuleStore{modules: map[string]*model.Module{"moduleA": {Code: "moduleA"}}}}}

	resp, err := biz.Publish(context.Background(), "s1")
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	if resp.Section.Status != model.SectionStatusNormal {
		t.Fatalf("expected section status to be normal, got %d", resp.Section.Status)
	}
	stored := sections.sections["s1"]
	if stored.Status != model.SectionStatusNormal {
		t.Fatalf("expected stored section status to be normal, got %d", stored.Status)
	}
}

func TestSectionBizGetListReturnsAllStatuses(t *testing.T) {
	sections := newFakeSectionStore()
	sections.sections["normal"] = &model.Section{Code: "normal", ModuleCode: "moduleA", Status: model.SectionStatusNormal}
	sections.sections["deleted"] = &model.Section{Code: "deleted", ModuleCode: "moduleA", Status: model.SectionStatusDeleted}
	biz := sectionBiz{ds: &fakeStore{sections: sections, modules: &fakeModuleStore{modules: map[string]*model.Module{"moduleA": {Code: "moduleA"}}}}}

	resp, err := biz.GetList(context.Background(), "moduleA")
	if err != nil {
		t.Fatalf("GetList returned error: %v", err)
	}
	if len(resp.Sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(resp.Sections))
	}
}
