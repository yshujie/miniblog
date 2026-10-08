// notion-sync provides explicit, local-operator sync and historical takeover tools.
// It never migrates schemas automatically; bootstrap writes require two opt-ins.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/yshujie/miniblog/internal/miniblog/notionsync"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/pkg/db"
	"github.com/yshujie/miniblog/scripts/internal/mysqlconfig"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }
func run(args []string, out, errout io.Writer) int {
	fs := flag.NewFlagSet("notion-sync", flag.ContinueOnError)
	fs.SetOutput(errout)
	config := mysqlconfig.Bind(fs)
	mode := fs.String("mode", "dry_run", "dry_run|sync|status|bootstrap_preview|bootstrap_apply")
	manualMatches := fs.String("manual-matches", "", "bootstrap_preview 手工关联 BootstrapReviewInput JSON 文件")
	confirmations := fs.String("confirmations", "", "已核对的 BootstrapInput JSON 文件（bootstrap_apply 必填）")
	allowWrite := fs.Bool("allow-notion-write", false, "显式允许一次性接管状态回填；普通同步不会写 Notion")
	enableSync := fs.Bool("enable-sync", false, "显式启用本次本地同步；默认只读预览")
	author := fs.String("author", "", "新文章默认作者，已有文章作者保持不变")
	report := fs.String("report", "-", "JSON 结果文件路径；- 为标准输出")
	timeout := fs.Duration("timeout", 10*time.Minute, "本次任务最大时间")
	if e := fs.Parse(args); e != nil {
		return 2
	}
	if e := validateMode(*mode, *allowWrite, *confirmations); e != nil {
		fmt.Fprintln(errout, e)
		return 2
	}
	var review notionsync.BootstrapReviewInput
	if *manualMatches != "" {
		if *mode != "bootstrap_preview" {
			fmt.Fprintln(errout, "--manual-matches 仅适用于 bootstrap_preview")
			return 2
		}
		if e := readStrictJSON(*manualMatches, &review); e != nil {
			fmt.Fprintln(errout, "手工关联清单无效:", e)
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
			fmt.Fprintln(errout, "确认清单无效:", e)
			return 2
		}
	}
	gdb, e := db.NewMySQL(config.DBOptions(1))
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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()
	service := notionsync.New(store.NewStore(gdb), notionsync.Options{Enabled: *enableSync, Author: *author, Client: source.NewNotionSyncClient(os.Getenv("MINIBLOG_NOTION_TOKEN"))})
	var result interface{}
	switch *mode {
	case "status":
		result, e = service.Status(ctx)
	case "bootstrap_preview":
		result, e = service.BootstrapPreview(ctx, review)
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
	if result != nil {
		encoded, encodeErr := json.MarshalIndent(result, "", "  ")
		if encodeErr != nil {
			fmt.Fprintln(errout, "结果序列化失败")
			return 1
		}
		encoded = append(encoded, '\n')
		if *report == "-" {
			_, encodeErr = out.Write(encoded)
		} else {
			encodeErr = os.WriteFile(*report, encoded, 0600)
		}
		if encodeErr != nil {
			fmt.Fprintln(errout, "结果文件写入失败")
			return 1
		}
	}
	if e != nil {
		fmt.Fprintln(errout, e)
		return 1
	}
	return 0
}
func validateMode(mode string, write bool, path string) error {
	switch mode {
	case "dry_run", "sync", "status", "bootstrap_preview":
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
	decoder := json.NewDecoder(io.LimitReader(f, 2<<20))
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
