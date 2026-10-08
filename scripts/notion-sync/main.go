// notion-sync provides explicit, local-operator sync and historical takeover tools.
// It never migrates schemas automatically; bootstrap writes require two opt-ins.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/yshujie/miniblog/internal/miniblog/notionsync"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/pkg/db"
	"github.com/yshujie/miniblog/scripts/internal/mysqlconfig"
	"github.com/yshujie/miniblog/scripts/internal/safereport"
	"gorm.io/gorm"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

type dependencies struct {
	client  func(string) source.SyncNotionClient
	connect func(*db.MySQLOptions) (*gorm.DB, error)
}

func run(args []string, out, errout io.Writer) int {
	return runWithDependencies(args, out, errout, dependencies{})
}
func runWithDependencies(args []string, out, errout io.Writer, deps dependencies) int {
	if deps.client == nil {
		deps.client = func(token string) source.SyncNotionClient { return source.NewNotionSyncClient(token) }
	}
	if deps.connect == nil {
		deps.connect = db.NewMySQL
	}
	fs := flag.NewFlagSet("notion-sync", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	config := mysqlconfig.Bind(fs)
	mode := fs.String("mode", "dry_run", "schema_check|catalog_prepare|dry_run|sync|status|bootstrap_preview|bootstrap_apply")
	sourceID := fs.String("source-id", "", "catalog_prepare 的单个白名单 data source ID")
	revision := fs.Uint64("expected-config-revision", 0, "catalog_prepare 审核时的来源配置版本")
	manualMatches := fs.String("manual-matches", "", "bootstrap_preview 手工关联 BootstrapReviewInput JSON 文件")
	confirmations := fs.String("confirmations", "", "已核对的 BootstrapInput JSON 文件（bootstrap_apply 必填）")
	allowWrite := fs.Bool("allow-notion-write", false, "显式允许一次性接管状态回填；普通同步不会写 Notion")
	enableSync := fs.Bool("enable-sync", false, "显式启用本次本地同步；默认只读预览")
	author := fs.String("author", "", "新文章默认作者，已有文章作者保持不变")
	report := fs.String("report", "-", "JSON 结果文件路径；- 为标准输出")
	timeout := fs.Duration("timeout", 10*time.Minute, "本次任务最大时间")
	if e := fs.Parse(args); e != nil {
		if errors.Is(e, flag.ErrHelp) {
			fs.SetOutput(errout)
			fs.PrintDefaults()
			return 0
		}
		fmt.Fprintln(errout, "命令参数无效；使用 -h 查看帮助")
		return 2
	}
	if e := validateMode(*mode, *allowWrite, *confirmations); e != nil {
		fmt.Fprintln(errout, e)
		return 2
	}
	if fs.NArg() != 0 || *timeout <= 0 {
		fmt.Fprintln(errout, "不接受位置参数，timeout 必须为正数")
		return 2
	}
	revisionSet := false
	fs.Visit(func(value *flag.Flag) {
		if value.Name == "expected-config-revision" {
			revisionSet = true
		}
	})
	if e := validateCatalogArguments(*mode, *sourceID, *revision, revisionSet); e != nil {
		fmt.Fprintln(errout, e)
		return 2
	}
	if *manualMatches != "" && *mode != "bootstrap_preview" {
		fmt.Fprintln(errout, "--manual-matches 仅适用于 bootstrap_preview")
		return 2
	}
	if *confirmations != "" && *mode != "bootstrap_apply" {
		fmt.Fprintln(errout, "--confirmations 仅适用于 bootstrap_apply")
		return 2
	}
	if *report != "-" {
		if e := safereport.CheckDestination(*report); e != nil {
			fmt.Fprintln(errout, "结果文件目标无效")
			return 2
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()
	if *mode == "schema_check" {
		result, checkErr := notionsync.CheckSchemas(ctx, deps.client(os.Getenv("MINIBLOG_NOTION_TOKEN")))
		if e := writeResult(result, *report, out); e != nil {
			fmt.Fprintln(errout, "结果文件写入失败")
			return 1
		}
		if checkErr != nil {
			fmt.Fprintln(errout, checkErr)
			return 1
		}
		return 0
	}
	var review notionsync.BootstrapReviewInput
	if *manualMatches != "" {
		if e := readStrictJSON(*manualMatches, &review); e != nil {
			fmt.Fprintln(errout, "手工关联清单无效，请核对文件与 JSON 字段")
			return 2
		}
		if len(review.ManualMatches) == 0 {
			fmt.Fprintln(errout, "手工关联清单为空")
			return 2
		}
	}
	var input notionsync.BootstrapInput
	if *mode == "bootstrap_apply" {
		if strings.TrimSpace(os.Getenv("MINIBLOG_NOTION_BOOTSTRAP_TOKEN")) == "" {
			fmt.Fprintln(errout, "bootstrap_apply 需要独立 MINIBLOG_NOTION_BOOTSTRAP_TOKEN")
			return 2
		}
		var e error
		input, e = readConfirmations(*confirmations)
		if e != nil {
			fmt.Fprintln(errout, "确认清单无效，请核对文件与 JSON 字段")
			return 2
		}
	}
	gdb, e := deps.connect(config.DBOptions(1))
	if e != nil {
		fmt.Fprintln(errout, "连接数据库失败")
		return 1
	}
	sqlDB, e := gdb.DB()
	if e != nil {
		fmt.Fprintln(errout, "数据库句柄不可用")
		return 1
	}
	defer sqlDB.Close()
	service := notionsync.New(store.NewStore(gdb), notionsync.Options{Enabled: *enableSync, Author: *author, Client: deps.client(os.Getenv("MINIBLOG_NOTION_TOKEN"))})
	var result interface{}
	switch *mode {
	case "catalog_prepare":
		prepared, prepareErr := service.PrepareCatalog(ctx, notionsync.CatalogPrepareInput{SourceID: *sourceID, ExpectedConfigRevision: *revision})
		result, e = prepared, prepareErr
		if e == nil && prepared != nil {
			for _, item := range prepared.Items {
				if item.Status != "bound" {
					e = fmt.Errorf("部分主题目录待核对绑定，请查看完整结果")
					break
				}
			}
		}
	case "status":
		result, e = service.Status(ctx)
	case "bootstrap_preview":
		preview, previewErr := service.BootstrapPreview(ctx, review)
		result, e = preview, previewErr
		if e == nil {
			if preview == nil {
				e = validatePreviewRun(nil)
			} else {
				run, readErr := service.Run(ctx, preview.RunID)
				if readErr != nil {
					// Database/driver errors may contain connection details. Do not echo them.
					e = fmt.Errorf("无法核验历史预览运行结果，请查看运行记录")
				} else {
					e = validatePreviewRun(run)
				}
			}
		}
	case "bootstrap_apply":
		applied, applyErr := service.BootstrapApply(ctx, input, source.NewNotionSyncClient(os.Getenv("MINIBLOG_NOTION_BOOTSTRAP_TOKEN")))
		result, e = applied, applyErr
		if e == nil && applied != nil {
			for _, item := range applied.Items {
				if item.Outcome == "failed" {
					e = fmt.Errorf("部分接管项失败，请查阅完整结果并重新核验")
					break
				}
			}
		}
	default:
		var trigger *notionsync.TriggerResult
		trigger, e = service.Trigger(ctx, notionsync.TriggerInput{Mode: *mode})
		if e == nil {
			var run *notionsync.RunDTO
			run, e = waitRun(ctx, service, trigger.RunID)
			if e == nil {
				items, itemErr := readAllItems(ctx, service, trigger.RunID)
				e = itemErr
				result = struct {
					Run   *notionsync.RunDTO   `json:"run"`
					Items []notionsync.ItemDTO `json:"items"`
				}{run, items}
				if run.Status != "completed" {
					e = fmt.Errorf("同步未完成: %s", run.Status)
				}
			}
		}
	}
	stopctx, stopcancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopcancel()
	_ = service.Stop(stopctx)
	if err := writeResult(result, *report, out); err != nil {
		fmt.Fprintln(errout, "结果文件写入失败")
		return 1
	}
	if e != nil {
		fmt.Fprintln(errout, e)
		return 1
	}
	return 0
}

// Pending/frozen entries are normal review states. Candidate public conditions
// remain in the report for per-item approval; only run failures/blockers reject it.
func validatePreviewRun(run *notionsync.RunDTO) error {
	if run == nil || run.Status != "completed" || run.Counts.Failed != 0 || run.Counts.Blocked != 0 {
		return fmt.Errorf("历史预览未完整完成，请查看运行记录与逐页结果")
	}
	return nil
}

func validateMode(mode string, write bool, path string) error {
	switch mode {
	case "schema_check", "catalog_prepare", "dry_run", "sync", "status", "bootstrap_preview":
		if write {
			return fmt.Errorf("--allow-notion-write 仅适用于 bootstrap_apply")
		}
	case "bootstrap_apply":
		if !write || path == "" {
			return fmt.Errorf("bootstrap_apply 需要 --allow-notion-write 和 --confirmations")
		}
	default:
		return fmt.Errorf("未知 mode")
	}
	return nil
}
func validateCatalogArguments(mode, sourceID string, revision uint64, revisionSet bool) error {
	if mode != "catalog_prepare" {
		if sourceID != "" || revisionSet {
			return fmt.Errorf("source-id/expected-config-revision 仅适用于 catalog_prepare")
		}
		return nil
	}
	if sourceID == "" || revision == 0 || !revisionSet {
		return fmt.Errorf("catalog_prepare 需要 source-id 和有效 expected-config-revision")
	}
	normal := strings.ToLower(strings.ReplaceAll(sourceID, "-", ""))
	for _, id := range notionsync.AllowedSources() {
		if normal == strings.ReplaceAll(id, "-", "") {
			return nil
		}
	}
	return fmt.Errorf("数据源不在五库白名单")
}
func writeResult(result interface{}, report string, out io.Writer) error {
	if result == nil {
		return nil
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if report == "-" {
		_, err = out.Write(encoded)
		return err
	}
	return safereport.Write(report, encoded)
}

func readConfirmations(path string) (notionsync.BootstrapInput, error) {
	var input notionsync.BootstrapInput
	if e := readStrictJSON(path, &input); e != nil {
		return input, e
	}
	if len(input.Confirmations) == 0 {
		return input, fmt.Errorf("确认清单为空")
	}
	return input, nil
}
func readStrictJSON(path string, input interface{}) error {
	f, e := os.Open(path)
	if e != nil {
		return e
	}
	defer f.Close()
	const maxConfirmationBytes = 2 << 20
	data, e := io.ReadAll(io.LimitReader(f, maxConfirmationBytes+1))
	if e != nil {
		return e
	}
	if len(data) > maxConfirmationBytes {
		return fmt.Errorf("清单超过 2 MiB")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if e := decoder.Decode(input); e != nil {
		return e
	}
	var extra interface{}
	if decoder.Decode(&extra) != io.EOF {
		return fmt.Errorf("清单只能包含一个 JSON 对象")
	}
	return nil
}

type itemsReader interface {
	Items(context.Context, string, notionsync.ListQuery) (*notionsync.PageResult[notionsync.ItemDTO], error)
}

func readAllItems(ctx context.Context, service itemsReader, id string) ([]notionsync.ItemDTO, error) {
	items := []notionsync.ItemDTO{}
	for page := 1; ; page++ {
		batch, e := service.Items(ctx, id, notionsync.ListQuery{Page: page, Limit: 100})
		if e != nil {
			return nil, e
		}
		items = append(items, batch.Items...)
		if int64(len(items)) >= batch.Total {
			return items, nil
		}
		if len(batch.Items) == 0 {
			return nil, fmt.Errorf("运行审计分页不完整")
		}
	}
}

func waitRun(ctx context.Context, service *notionsync.Service, id string) (*notionsync.RunDTO, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		run, e := service.Run(ctx, id)
		if e != nil {
			return nil, e
		}
		if run.Status != "running" {
			return run, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}
