package notionsync

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"gorm.io/gorm"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

var allowedSources = map[string]string{
	"2bf330bd-ddf1-80a6-aa49-000bbd1e154b": "Go",
	"d579f619-4f1c-4e7e-9593-ab6529fc63d0": "DDD",
	"cfa962f3-12be-40aa-95eb-9122186bd4ba": "数据库",
	"03d24108-5172-4e8f-8cfd-e3bd52e437af": "数据结构&算法",
	"d13097e3-b78a-4b6e-8848-979749d4b2f3": "项目开发",
}

type Service struct {
	ds        store.IStore
	opts      Options
	client    source.SyncNotionClient
	mu        sync.Mutex
	stop      context.CancelFunc
	runCancel context.CancelFunc
	ctx       context.Context
	wg        sync.WaitGroup
	stopping  bool
	active    map[string]context.CancelFunc
}

func New(ds store.IStore, opts Options) *Service {
	if opts.Interval <= 0 {
		opts.Interval = 5 * time.Minute
	}
	if opts.LeaseDuration <= 0 {
		opts.LeaseDuration = 60 * time.Second
	}
	if opts.HeartbeatInterval <= 0 {
		opts.HeartbeatInterval = 15 * time.Second
	}
	if opts.OwnerID == "" {
		opts.OwnerID = newID()
	}
	if opts.Client == nil {
		opts.Client = source.NewNotionSyncClient(os.Getenv("MINIBLOG_NOTION_TOKEN"))
	}
	s := &Service{ds: ds, opts: opts, client: opts.Client, ctx: context.Background(), active: map[string]context.CancelFunc{}}
	if client, ok := s.client.(*source.HTTPNotionSyncClient); ok {
		client.SetCooldownObserver(s.observeCooldown)
	}
	return s
}
func (s *Service) db(ctx context.Context) (*gorm.DB, error) {
	if s.ds == nil || s.ds.DB() == nil {
		return nil, unavailable("同步数据库未配置")
	}
	db := s.ds.DB().WithContext(ctx)
	for _, m := range []interface{}{&model.NotionSyncControl{}, &model.NotionSyncSource{}, &model.NotionCatalogBinding{}, &model.NotionPageBinding{}, &model.NotionSyncRun{}, &model.NotionSyncRunItem{}} {
		if !db.Migrator().HasTable(m) {
			return nil, unavailable("请先完成 Notion 同步数据库迁移")
		}
	}
	return db, nil
}
func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	v := hex.EncodeToString(b[:])
	return v[:8] + "-" + v[8:12] + "-" + v[12:16] + "-" + v[16:20] + "-" + v[20:]
}
func normalizeID(id string) string {
	v := strings.ToLower(strings.ReplaceAll(id, "-", ""))
	if len(v) != 32 {
		return id
	}
	return v[:8] + "-" + v[8:12] + "-" + v[12:16] + "-" + v[16:20] + "-" + v[20:]
}
func normalizePageID(id string) string { return strings.ToLower(strings.ReplaceAll(id, "-", "")) }
func jsonText(v interface{}) string    { b, _ := json.Marshal(v); return string(b) }
func stateName(v int) string {
	switch v {
	case 1:
		return "draft"
	case 2:
		return "published"
	case 3:
		return "unpublished"
	case 4:
		return "archived"
	}
	return ""
}
func stateValue(v string) int {
	switch v {
	case "draft":
		return 1
	case "published":
		return 2
	case "unpublished":
		return 3
	case "archived":
		return 4
	}
	return 0
}

// AllowedSources returns the fixed data-source inventory, without credentials.
func AllowedSources() []string {
	result := make([]string, 0, len(allowedSources))
	for id := range allowedSources {
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}
