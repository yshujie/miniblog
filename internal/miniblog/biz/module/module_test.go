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
