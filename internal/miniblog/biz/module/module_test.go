package module

import (
	"context"
	"testing"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
	"gorm.io/gorm"
)

type fakeModuleStore struct {
	modules map[string]*model.Module
	nextID  uint64
}

// Note: DeleteByCode implemented below for test use.

func newFakeModuleStore() *fakeModuleStore {
	return &fakeModuleStore{modules: make(map[string]*model.Module), nextID: 1}
}

func (f *fakeModuleStore) Create(module *model.Module) error {
	copy := *module
	if copy.ID == 0 {
		copy.ID = f.nextID
		f.nextID++
	}
	f.modules[copy.Code] = &copy
	return nil
}

func (f *fakeModuleStore) GetByCode(code string) (*model.Module, error) {
	if module, ok := f.modules[code]; ok {
		copy := *module
		return &copy, nil
	}
	return nil, nil
}

func (f *fakeModuleStore) GetAll() ([]*model.Module, error) {
	result := make([]*model.Module, 0, len(f.modules))
	for _, module := range f.modules {
		copy := *module
		result = append(result, &copy)
	}
	return result, nil
}

func (f *fakeModuleStore) GetNormalModules() ([]*model.Module, error) {
	var result []*model.Module
	for _, module := range f.modules {
		if module.Status == model.ModuleStatusNormal {
			copy := *module
			result = append(result, &copy)
		}
	}
	return result, nil
}

func (f *fakeModuleStore) Update(module *model.Module) error {
	if _, ok := f.modules[module.Code]; !ok {
		return nil
	}
	copy := *module
	f.modules[module.Code] = &copy
	return nil
}

type fakeModuleBizStore struct {
	modules *fakeModuleStore
}

func (f *fakeModuleBizStore) DB() *gorm.DB                 { return nil }
func (f *fakeModuleBizStore) Users() store.UserStore       { return nil }
func (f *fakeModuleBizStore) Modules() store.ModuleStore   { return f.modules }
func (f *fakeModuleBizStore) Sections() store.SectionStore { return nil }
func (f *fakeModuleBizStore) Articles() store.ArticleStore { return nil }

func TestModuleBizCreateAndGet(t *testing.T) {
	fakeStore := &fakeModuleBizStore{modules: newFakeModuleStore()}
	biz := moduleBiz{ds: fakeStore}

	resp, err := biz.Create(context.Background(), &v1.CreateModuleRequest{Code: "m1", Title: "Module 1"})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if resp.Module == nil || resp.Module.Code != "m1" {
		t.Fatalf("unexpected response: %+v", resp)
	}

	getResp, err := biz.GetOne(context.Background(), "m1")
	if err != nil {
		t.Fatalf("GetOne returned error: %v", err)
	}
	if getResp.Module == nil || getResp.Module.Title != "Module 1" {
		t.Fatalf("unexpected module info: %+v", getResp)
	}
}

// Implement DeleteByCode so biz.Delete can call it in tests
func (f *fakeModuleStore) DeleteByCode(code string) error {
	if _, ok := f.modules[code]; !ok {
		return nil
	}
	delete(f.modules, code)
	return nil
}

// emptyArticleStore returns no articles (used to simulate no direct article dependency)
type emptyArticleStore struct{}

func (e *emptyArticleStore) Create(article *model.Article) error      { return nil }
func (e *emptyArticleStore) GetOne(id uint64) (*model.Article, error) { return nil, nil }
func (e *emptyArticleStore) GetList(filter interface{}, page int, limit int) ([]*model.Article, error) {
	return []*model.Article{}, nil
}
func (e *emptyArticleStore) Update(article *model.Article) error { return nil }

// fake section store implementations for tests
type fakeSectionStoreWithDep struct{}

func (s *fakeSectionStoreWithDep) Create(section *model.Section) error           { return nil }
func (s *fakeSectionStoreWithDep) GetByCode(code string) (*model.Section, error) { return nil, nil }
func (s *fakeSectionStoreWithDep) GetSections(moduleCode string) ([]*model.Section, error) {
	if moduleCode == "m_del" {
		return []*model.Section{{Code: "s1", ModuleCode: "m_del"}}, nil
	}
	return []*model.Section{}, nil
}
func (s *fakeSectionStoreWithDep) GetNormalSections(moduleCode string) ([]*model.Section, error) {
	return s.GetSections(moduleCode)
}
func (s *fakeSectionStoreWithDep) Update(section *model.Section) error { return nil }
func (s *fakeSectionStoreWithDep) DeleteByCode(code string) error      { return nil }

type fakeSectionStoreNoDep struct{}

func (s *fakeSectionStoreNoDep) Create(section *model.Section) error           { return nil }
func (s *fakeSectionStoreNoDep) GetByCode(code string) (*model.Section, error) { return nil, nil }
func (s *fakeSectionStoreNoDep) GetSections(moduleCode string) ([]*model.Section, error) {
	return []*model.Section{}, nil
}
func (s *fakeSectionStoreNoDep) GetNormalSections(moduleCode string) ([]*model.Section, error) {
	return []*model.Section{}, nil
}
func (s *fakeSectionStoreNoDep) Update(section *model.Section) error { return nil }
func (s *fakeSectionStoreNoDep) DeleteByCode(code string) error      { return nil }

// composed store implements IStore for tests
type composedStore struct {
	mod store.ModuleStore
	sec store.SectionStore
	art store.ArticleStore
}

func (c *composedStore) DB() *gorm.DB                 { return nil }
func (c *composedStore) Users() store.UserStore       { return nil }
func (c *composedStore) Modules() store.ModuleStore   { return c.mod }
func (c *composedStore) Sections() store.SectionStore { return c.sec }
func (c *composedStore) Articles() store.ArticleStore { return c.art }

// Test: module delete fails when there are dependent sections or articles
func TestModuleDelete_WhenHasDependents_ShouldFail(t *testing.T) {
	fakeModules := newFakeModuleStore()
	fakeModules.modules["m_del"] = &model.Module{ID: 10, Code: "m_del", Title: "to delete"}

	fakeArticles := &emptyArticleStore{}

	ms := &composedStore{mod: fakeModules, sec: &fakeSectionStoreWithDep{}, art: fakeArticles}
	biz := moduleBiz{ds: ms}

	if err := biz.Delete(context.Background(), "m_del"); err == nil {
		t.Fatalf("expected delete to fail due to dependents, but it succeeded")
	}
}

// Test: module delete succeeds when no dependents
func TestModuleDelete_WhenNoDependents_ShouldSucceed(t *testing.T) {
	fakeModules := newFakeModuleStore()
	fakeModules.modules["m_ok"] = &model.Module{ID: 11, Code: "m_ok", Title: "ok"}

	fakeArticles := &emptyArticleStore{}

	ms := &composedStore{mod: fakeModules, sec: &fakeSectionStoreNoDep{}, art: fakeArticles}
	biz := moduleBiz{ds: ms}

	if err := biz.Delete(context.Background(), "m_ok"); err != nil {
		t.Fatalf("expected delete to succeed, got error: %v", err)
	}
}

func TestModuleBizPublishAndUnpublish(t *testing.T) {
	fakeModules := newFakeModuleStore()
	fakeModules.modules["m2"] = &model.Module{ID: 2, Code: "m2", Title: "Module 2", Status: model.ModuleStatusDeleted}
	biz := moduleBiz{ds: &fakeModuleBizStore{modules: fakeModules}}

	publishResp, err := biz.Publish(context.Background(), "m2")
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	if publishResp.Module.Status != model.ModuleStatusNormal {
		t.Fatalf("expected status normal after publish, got %d", publishResp.Module.Status)
	}

	unpublishResp, err := biz.Unpublish(context.Background(), "m2")
	if err != nil {
		t.Fatalf("Unpublish returned error: %v", err)
	}
	if unpublishResp.Module.Status != model.ModuleStatusDeleted {
		t.Fatalf("expected status deleted after unpublish, got %d", unpublishResp.Module.Status)
	}
}
