// 批量创建/更新 article（按 JSON 文件 upsert）。
//
// 用法:
//
//	MYSQL_DSN='user:password@tcp(host:3306)/miniblog?parseTime=true' \
//	  ./scripts/batch-upsert-articles.sh -file articles.json -dry-run
//	go run ./scripts/batch-upsert-articles -file ./scripts/batch-upsert-articles/articles.example.json
//	./scripts/batch-upsert-articles.sh -file articles.json -dry-run
//
// 指定 id 时按 ID 更新/创建；其余外链按来源身份匹配。
// 标题不作为身份。目录、状态、顺序和事务共用后台文章用例。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	articlebiz "github.com/yshujie/miniblog/internal/miniblog/biz/article"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/pkg/db"
	"github.com/yshujie/miniblog/scripts/internal/mysqlconfig"
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
	Content        *string  `json:"content"`
	ContentFile    string   `json:"content_file"`
	Pos            *int     `json:"pos"`
	Status         string   `json:"status"`
}

func (item *articleInput) UnmarshalJSON(data []byte) error {
	type alias articleInput
	raw := struct {
		ID json.RawMessage `json:"id"`
		*alias
	}{alias: (*alias)(item)}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if len(raw.ID) == 0 || string(raw.ID) == "null" {
		return nil
	}
	text := strings.TrimSpace(string(raw.ID))
	if strings.HasPrefix(text, "\"") {
		if err := json.Unmarshal(raw.ID, &text); err != nil {
			return err
		}
	}
	id, err := strconv.ParseUint(text, 10, 64)
	if err != nil || id > math.MaxInt64 {
		return errors.New("id 必须是 signed BIGINT 范围内的十进制整数")
	}
	item.ID = id
	return nil
}

func main() {
	os.Exit(run())
}

func run() int {
	mysqlConfig := mysqlconfig.Bind(flag.CommandLine)
	var (
		filePath = flag.String("file", "", "文章 JSON 文件路径（必填）")
		dryRun   = flag.Bool("dry-run", false, "仅预览，不写入数据库")
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

	gdb, err := db.NewMySQL(mysqlConfig.DBOptions(1))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: 连接数据库失败: %v\n", err)
		return 1
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer sqlDB.Close()

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
		case "already_registered":
			fmt.Printf("%s EXISTS id=%d title=%q\n", label, articleID, item.Title)
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
	content, err := resolveContent(baseDir, item)
	if err != nil {
		return "", 0, err
	}
	status, err := parseStatus(item.Status)
	if err != nil {
		return "", 0, err
	}
	var explicitStatus *int
	if strings.TrimSpace(item.Status) != "" {
		explicitStatus = &status
	}
	result, err := articlebiz.New(store.NewStore(gdb)).Import(context.Background(), articlebiz.ImportRequest{
		ID: item.ID, Title: item.Title, ExternalLink: item.ExternalLink, Content: content,
		SectionCode: item.SectionCode, SubsectionCode: item.SubsectionCode,
		Author: item.Author, Tags: item.Tags, Pos: item.Pos, Status: explicitStatus,
	}, dryRun)
	if err != nil {
		return "", 0, err
	}
	action := result.Outcome
	if action == "created" {
		action = "create"
	}
	if action == "updated" {
		action = "update"
	}
	if dryRun && action != "already_registered" {
		action = "dry-run-" + action
	}
	return action, result.ID, nil
}

func validateInput(item articleInput) error {
	if strings.TrimSpace(item.Title) == "" {
		return errors.New("title 不能为空")
	}
	if strings.TrimSpace(item.SectionCode) == "" {
		return errors.New("section_code 不能为空")
	}
	if item.ID > math.MaxInt64 {
		return errors.New("id 超出 signed BIGINT 范围")
	}
	return nil
}

func resolveContent(baseDir string, item articleInput) (*string, error) {
	if item.Content != nil {
		return item.Content, nil
	}
	if strings.TrimSpace(item.ContentFile) == "" {
		if strings.TrimSpace(item.ExternalLink) != "" {
			return nil, nil
		}
		return nil, errors.New("content、content_file、external_link 至少提供一个有效内容来源")
	}

	path := item.ContentFile
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取 content_file 失败: %w", err)
	}
	content := string(data)
	return &content, nil
}

func parseStatus(raw string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "draft":
		return model.ArticleStatusDraft, nil
	case "published", "publish":
		return model.ArticleStatusPublished, nil
	case "unpublished", "unpublish":
		return model.ArticleStatusUnpublished, nil
	case "archived", "archive", "deleted":
		return model.ArticleStatusDeleted, nil
	default:
		n, err := strconv.Atoi(raw)
		if err != nil {
			return 0, fmt.Errorf("无效 status: %q（支持 draft/published/unpublished/archived 或 1/2/3/4）", raw)
		}
		switch n {
		case model.ArticleStatusDraft, model.ArticleStatusPublished, model.ArticleStatusUnpublished, model.ArticleStatusDeleted:
			return n, nil
		default:
			return 0, fmt.Errorf("无效 status 数值: %d", n)
		}
	}
}
