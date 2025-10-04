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

func (f *fakeSectionStore) DeleteByCode(code string) error {
	if _, ok := f.sections[code]; !ok {
		return errors.New("section not found")
	}
	delete(f.sections, code)
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
func (f *fakeModuleStore) DeleteByCode(code string) error {
	if f.modules == nil {
		return errors.New("modules map is nil")
	}
	if _, ok := f.modules[code]; !ok {
		return errors.New("module not found")
	}
	delete(f.modules, code)
	return nil
}

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

// Tests for Delete behavior
type fakeArticleStoreForSectionTest struct {
	articles map[uint64]*model.Article
}

func (f *fakeArticleStoreForSectionTest) Create(article *model.Article) error      { return nil }
func (f *fakeArticleStoreForSectionTest) GetOne(id uint64) (*model.Article, error) { return nil, nil }
func (f *fakeArticleStoreForSectionTest) GetList(filter interface{}, page int, limit int) ([]*model.Article, error) {
	m := filter.(map[string]interface{})
	if code, ok := m["section_code"]; ok {
		for _, a := range f.articles {
			if a.SectionCode == code.(string) {
				return []*model.Article{a}, nil
			}
		}
	}
	return []*model.Article{}, nil
}
func (f *fakeArticleStoreForSectionTest) Update(article *model.Article) error { return nil }

// Test: delete fails when articles exist under section
func TestSectionDelete_WhenHasArticles_ShouldFail(t *testing.T) {
	sections := newFakeSectionStore()
	sections.sections["s_del"] = &model.Section{Code: "s_del", ModuleCode: "m1", Status: model.SectionStatusNormal}

	fakeArticles := &fakeArticleStoreForSectionTest{articles: map[uint64]*model.Article{1: {ID: 1, SectionCode: "s_del"}}}

	// composed store defined at file scope (see composedSectionStore)
	ms2 := &composedSectionStore{sec: sections, mod: &fakeModuleStore{modules: map[string]*model.Module{"m1": {Code: "m1"}}}, art: fakeArticles}
	biz := sectionBiz{ds: ms2}

	if err := biz.Delete(context.Background(), "s_del"); err == nil {
		t.Fatalf("expected delete to fail due to articles, but it succeeded")
	}
}

// Test: delete succeeds when no articles
func TestSectionDelete_WhenNoArticles_ShouldSucceed(t *testing.T) {
	sections := newFakeSectionStore()
	sections.sections["s_ok"] = &model.Section{Code: "s_ok", ModuleCode: "m1", Status: model.SectionStatusNormal}

	fakeArticles := &fakeArticleStoreForSectionTest{articles: map[uint64]*model.Article{}}

	ms2 := &composedSectionStore{sec: sections, mod: &fakeModuleStore{modules: map[string]*model.Module{"m1": {Code: "m1"}}}, art: fakeArticles}
	biz := sectionBiz{ds: ms2}

	if err := biz.Delete(context.Background(), "s_ok"); err != nil {
		t.Fatalf("expected delete to succeed, got error: %v", err)
	}
}

// composedSectionStore implements IStore for section tests
type composedSectionStore struct {
	sec store.SectionStore
	mod store.ModuleStore
	art store.ArticleStore
}

func (c *composedSectionStore) DB() *gorm.DB                 { return nil }
func (c *composedSectionStore) Users() store.UserStore       { return nil }
func (c *composedSectionStore) Modules() store.ModuleStore   { return c.mod }
func (c *composedSectionStore) Sections() store.SectionStore { return c.sec }
func (c *composedSectionStore) Articles() store.ArticleStore { return c.art }
