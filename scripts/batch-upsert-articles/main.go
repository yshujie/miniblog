// 批量创建/更新 article（按 JSON 文件 upsert）。
//
// 用法:
//
//	go run ./scripts/batch-upsert-articles -file ./scripts/batch-upsert-articles/articles.example.json
//	./scripts/batch-upsert-articles.sh -file articles.json -dry-run
//
// 匹配规则（决定更新还是新建）:
//  1. 指定 id 且库中已存在 → 更新
//  2. 指定 section_code + external_link（非空）→ 更新
//  3. 指定 section_code + title → 更新
//  4. 否则 → 新建
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/pkg/db"
	"gorm.io/gorm"
)

type inputFile struct {
	Articles []articleInput `json:"articles"`
}

type articleInput struct {
	ID             uint64   `json:"id"`
	Title          string   `json:"title"`
	SectionCode    string   `json:"section_code"`
	SubsectionCode string   `json:"subsection_code"`
	Author         string   `json:"author"`
	Tags           []string `json:"tags"`
	ExternalLink   string   `json:"external_link"`
	Content        string   `json:"content"`
	ContentFile    string   `json:"content_file"`
	Pos            *int     `json:"pos"`
	Status         string   `json:"status"`
}

func main() {
	os.Exit(run())
}

func run() int {
	var (
		filePath = flag.String("file", "", "文章 JSON 文件路径（必填）")
		dryRun   = flag.Bool("dry-run", false, "仅预览，不写入数据库")
		host     = flag.String("host", envOr("MYSQL_HOST", "localhost"), "MySQL 主机")
		port     = flag.String("port", envOr("MYSQL_PORT", "3306"), "MySQL 端口")
		username = flag.String("user", envOr("MYSQL_USERNAME", "miniblog"), "MySQL 用户名")
		dbPass   = flag.String("db-password", envOr("MYSQL_PASSWORD", "miniblog123"), "MySQL 密码")
		database = flag.String("database", envOr("MYSQL_DATABASE", "miniblog"), "MySQL 数据库名")
	)
	flag.Parse()

	if *filePath == "" {
		fmt.Fprintln(os.Stderr, "error: 必须指定 -file")
		flag.Usage()
		return 1
	}

	items, baseDir, err := loadArticles(*filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: 读取 JSON 失败: %v\n", err)
		return 1
	}
	if len(items) == 0 {
		fmt.Fprintln(os.Stderr, "error: 文章列表为空")
		return 1
	}

	gdb, err := db.NewMySQL(&db.MySQLOptions{
		Host:     *host,
		Port:     *port,
		Username: *username,
		Password: *dbPass,
		Database: *database,
		LogLevel: 1,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: 连接数据库失败: %v\n", err)
		return 1
	}

	created, updated, failed := 0, 0, 0
	for i, item := range items {
		label := fmt.Sprintf("[%d/%d]", i+1, len(items))
		action, articleID, err := upsertArticle(gdb, baseDir, item, *dryRun)
		if err != nil {
			failed++
			fmt.Fprintf(os.Stderr, "%s FAIL title=%q: %v\n", label, item.Title, err)
			continue
		}
		switch action {
		case "create":
			created++
			fmt.Printf("%s CREATE id=%d title=%q section=%s\n", label, articleID, item.Title, item.SectionCode)
		case "update":
			updated++
			fmt.Printf("%s UPDATE id=%d title=%q section=%s\n", label, articleID, item.Title, item.SectionCode)
		case "dry-run-create":
			created++
			fmt.Printf("%s DRY-RUN create title=%q section=%s\n", label, item.Title, item.SectionCode)
		case "dry-run-update":
			updated++
			fmt.Printf("%s DRY-RUN update title=%q section=%s\n", label, item.Title, item.SectionCode)
		}
	}

	fmt.Printf("\n完成: created=%d updated=%d failed=%d dry_run=%v\n", created, updated, failed, *dryRun)
	if failed > 0 {
		return 1
	}
	return 0
}

func loadArticles(filePath string) ([]articleInput, string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "", err
	}

	var wrapped inputFile
	if err := json.Unmarshal(data, &wrapped); err == nil && len(wrapped.Articles) > 0 {
		return wrapped.Articles, filepath.Dir(filePath), nil
	}

	var items []articleInput
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, "", err
	}
	return items, filepath.Dir(filePath), nil
}

func upsertArticle(gdb *gorm.DB, baseDir string, item articleInput, dryRun bool) (string, uint64, error) {
	if err := validateInput(item); err != nil {
		return "", 0, err
	}
	if err := validateRelations(gdb, item); err != nil {
		return "", 0, err
	}

	content, err := resolveContent(baseDir, item)
	if err != nil {
		return "", 0, err
	}

	status, err := parseStatus(item.Status)
	if err != nil {
		return "", 0, err
	}

	existing, matchBy, err := findExisting(gdb, item)
	if err != nil {
		return "", 0, err
	}

	if dryRun {
		if existing != nil {
			return "dry-run-update", existing.ID, nil
		}
		return "dry-run-create", item.ID, nil
	}

	now := time.Now()
	tags := strings.Join(item.Tags, ",")

	if existing != nil {
		existing.Title = item.Title
		existing.Author = item.Author
		existing.Tags = tags
		existing.ExternalLink = item.ExternalLink
		existing.SectionCode = item.SectionCode
		existing.SubsectionCode = item.SubsectionCode
		existing.Content = content
		existing.Status = status
		if item.Pos != nil {
			existing.Pos = *item.Pos
		}
		existing.UpdatedAt = now

		if err := gdb.Save(existing).Error; err != nil {
			return "", 0, fmt.Errorf("更新失败（match=%s）: %w", matchBy, err)
		}
		return "update", existing.ID, nil
	}

	pos := 0
	if item.Pos != nil {
		pos = *item.Pos
	} else {
		pos, err = nextPos(gdb, item.SectionCode, item.SubsectionCode)
		if err != nil {
			return "", 0, err
		}
	}

	article := &model.Article{
		Title:          item.Title,
		Content:        content,
		ExternalLink:   item.ExternalLink,
		SectionCode:    item.SectionCode,
		SubsectionCode: item.SubsectionCode,
		Author:         item.Author,
		Tags:           tags,
		Pos:            pos,
		Status:         status,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if item.ID > 0 {
		article.ID = item.ID
		if err := gdb.Session(&gorm.Session{SkipHooks: true}).Create(article).Error; err != nil {
			return "", 0, fmt.Errorf("按指定 id 创建失败: %w", err)
		}
		return "create", article.ID, nil
	}

	if err := gdb.Create(article).Error; err != nil {
		return "", 0, fmt.Errorf("创建失败: %w", err)
	}
	return "create", article.ID, nil
}

func validateInput(item articleInput) error {
	if strings.TrimSpace(item.Title) == "" {
		return errors.New("title 不能为空")
	}
	if strings.TrimSpace(item.SectionCode) == "" {
		return errors.New("section_code 不能为空")
	}
	if strings.TrimSpace(item.Author) == "" {
		return errors.New("author 不能为空")
	}
	if strings.TrimSpace(item.ExternalLink) == "" {
		return errors.New("external_link 不能为空")
	}
	if len(item.Tags) == 0 {
		return errors.New("tags 不能为空")
	}
	return nil
}

func validateRelations(gdb *gorm.DB, item articleInput) error {
	var section model.Section
	if err := gdb.Where("code = ?", item.SectionCode).First(&section).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("章节不存在: section_code=%s", item.SectionCode)
		}
		return err
	}

	if item.SubsectionCode == "" {
		return nil
	}

	var subsection model.Subsection
	if err := gdb.Where("code = ?", item.SubsectionCode).First(&subsection).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("子章节不存在: subsection_code=%s", item.SubsectionCode)
		}
		return err
	}
	if subsection.SectionCode != item.SectionCode {
		return fmt.Errorf("子章节 %s 不属于章节 %s", item.SubsectionCode, item.SectionCode)
	}
	return nil
}

func resolveContent(baseDir string, item articleInput) (string, error) {
	if strings.TrimSpace(item.Content) != "" {
		return item.Content, nil
	}
	if strings.TrimSpace(item.ContentFile) == "" {
		if strings.TrimSpace(item.ExternalLink) != "" {
			return "content from local", nil
		}
		return "", errors.New("content、content_file、external_link 至少提供一个有效内容来源")
	}

	path := item.ContentFile
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取 content_file 失败: %w", err)
	}
	return string(data), nil
}

func findExisting(gdb *gorm.DB, item articleInput) (*model.Article, string, error) {
	if item.ID > 0 {
		var article model.Article
		err := gdb.Where("id = ?", item.ID).First(&article).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", nil
		}
		if err != nil {
			return nil, "", err
		}
		return &article, "id", nil
	}

	if item.ExternalLink != "" {
		var article model.Article
		err := gdb.Where("section_code = ? AND external_link = ?", item.SectionCode, item.ExternalLink).First(&article).Error
		if err == nil {
			return &article, "section_code+external_link", nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", err
		}
	}

	var article model.Article
	err := gdb.Where("section_code = ? AND title = ?", item.SectionCode, item.Title).First(&article).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	return &article, "section_code+title", nil
}

func nextPos(gdb *gorm.DB, sectionCode, subsectionCode string) (int, error) {
	query := gdb.Model(&model.Article{}).Where("section_code = ?", sectionCode)
	if subsectionCode != "" {
		query = query.Where("subsection_code = ?", subsectionCode)
	} else {
		query = query.Where("subsection_code = '' OR subsection_code IS NULL")
	}

	var maxPos *int
	if err := query.Select("MAX(pos)").Scan(&maxPos).Error; err != nil {
		return 0, err
	}
	if maxPos == nil {
		return 1, nil
	}
	return *maxPos + 1, nil
}

func parseStatus(raw string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "draft":
		return model.ArticleStatusDraft, nil
	case "published", "publish":
		return model.ArticleStatusPublished, nil
	case "unpublished", "unpublish":
		return model.ArticleStatusUnpublished, nil
	default:
		n, err := strconv.Atoi(raw)
		if err != nil {
			return 0, fmt.Errorf("无效 status: %q（支持 draft/published/unpublished 或 1/2/3）", raw)
		}
		switch n {
		case model.ArticleStatusDraft, model.ArticleStatusPublished, model.ArticleStatusUnpublished:
			return n, nil
		default:
			return 0, fmt.Errorf("无效 status 数值: %d", n)
		}
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
