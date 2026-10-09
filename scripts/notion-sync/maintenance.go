package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"unicode/utf8"

	"github.com/yshujie/miniblog/internal/miniblog/biz/catalog"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/notionsync"
	"github.com/yshujie/miniblog/internal/miniblog/store"
)

type maintenanceCommand struct {
	SourceID    string
	Revision    uint64
	Control     notionsync.ControlInput
	Source      notionsync.SourceInput
	Binding     notionsync.BindingInput
	CatalogPlan notionsync.CatalogPrepareInput
	Activation  catalog.CatalogActivationInput
}

type bindingCommand struct {
	BindingID   string `json:"binding_id"`
	OptionID    string `json:"option_id"`
	SectionCode string `json:"section_code"`
}

func allowedSourceID(value string) bool {
	normal := strings.ToLower(strings.ReplaceAll(value, "-", ""))
	for _, id := range notionsync.AllowedSources() {
		if normal == strings.ReplaceAll(id, "-", "") {
			return true
		}
	}
	return false
}

func validateMaintenanceArguments(mode, sourceID string, revision uint64, revisionSet bool, input, plan string) error {
	if mode == "bootstrap_apply" && (sourceID != "" || revisionSet) {
		if !allowedSourceID(sourceID) || !revisionSet || revision == 0 || input != "" || plan != "" {
			return fmt.Errorf("接管范围需白名单来源和明确版本")
		}
		return nil
	}
	scoped := mode == "source_update" || mode == "catalog_bind" || mode == "catalog_activate"
	if scoped {
		if !allowedSourceID(sourceID) || !revisionSet || revision == 0 || input == "" || plan != "" {
			return fmt.Errorf("维护命令需白名单来源、明确版本及输入文件")
		}
		return nil
	}
	if mode == "control_update" {
		if input == "" || sourceID != "" || revisionSet || plan != "" {
			return fmt.Errorf("控制命令仅接受明确输入文件")
		}
		return nil
	}
	if input != "" || (plan != "" && mode != "catalog_prepare") {
		return fmt.Errorf("此模式不接受维护输入文件")
	}
	return validateCatalogArguments(mode, sourceID, revision, revisionSet)
}

func readMaintenanceCommand(mode, sourceID string, revision uint64, input, plan string) (maintenanceCommand, error) {
	command := maintenanceCommand{SourceID: sourceID, Revision: revision, CatalogPlan: notionsync.CatalogPrepareInput{SourceID: sourceID, ExpectedConfigRevision: revision}}
	switch mode {
	case "control_update":
		if err := readStrictJSON(input, &command.Control); err != nil {
			return command, err
		}
		if command.Control.Enabled != nil || (command.Control.Paused == nil && command.Control.SourceWritesPaused == nil) {
			return command, fmt.Errorf("控制输入只接受暂停字段")
		}
	case "source_update":
		if err := readStrictJSON(input, &command.Source); err != nil {
			return command, err
		}
		if command.Source.ExpectedConfigRevision != nil && *command.Source.ExpectedConfigRevision != revision {
			return command, fmt.Errorf("来源版本冲突")
		}
		if command.Source.Label == "" && command.Source.ModuleCode == "" && command.Source.Enabled == nil && command.Source.Config == nil {
			return command, fmt.Errorf("来源修改为空")
		}
		command.Source.ExpectedConfigRevision = &command.Revision
	case "catalog_bind":
		var value bindingCommand
		if err := readStrictJSON(input, &value); err != nil {
			return command, err
		}
		id, idErr := strconv.ParseUint(value.BindingID, 10, 63)
		if idErr != nil || id == 0 || value.SectionCode == "" {
			return command, fmt.Errorf("绑定输入缺少ID或目录")
		}
		command.Binding = notionsync.BindingInput{BindingID: value.BindingID, SourceID: sourceID, OptionID: value.OptionID, SectionCode: value.SectionCode, ExpectedConfigRevision: &command.Revision}
	case "catalog_activate":
		if err := readStrictJSON(input, &command.Activation); err != nil {
			return command, err
		}
	case "catalog_prepare":
		if plan != "" {
			var reviewed struct {
				ReuseMap []catalog.TopicReuseInput `json:"reuse_map"`
			}
			if err := readStrictJSON(plan, &reviewed); err != nil {
				return command, err
			}
			if len(reviewed.ReuseMap) == 0 {
				return command, fmt.Errorf("审核复用映射为空")
			}
			command.CatalogPlan.ReuseMap = reviewed.ReuseMap
		}
	}
	return command, nil
}

func executeMaintenanceCommand(ctx context.Context, service *notionsync.Service, ds store.IStore, mode string, in maintenanceCommand) (interface{}, error) {
	switch mode {
	case "control_update":
		return service.UpdateControl(ctx, in.Control)
	case "source_update":
		return leasedMaintenance(ctx, ds, func(txStore store.IStore) (interface{}, error) {
			return notionsync.New(txStore, notionsync.Options{}).UpdateSource(ctx, in.SourceID, in.Source)
		})
	case "catalog_bind":
		return leasedMaintenance(ctx, ds, func(txStore store.IStore) (interface{}, error) {
			return notionsync.New(txStore, notionsync.Options{}).BindCatalog(ctx, in.Binding)
		})
	case "drain_status":
		control, err := store.NewNotionSyncRepository(ds.DB()).Control(ctx)
		if err != nil {
			return nil, fmt.Errorf("维护排空状态查询失败")
		}
		unresolved := []string{}
		if err := ds.DB().WithContext(ctx).Model(&model.NotionPageBinding{}).Where("management_state <> ? AND bootstrap_state IN ?", model.NotionManagementManaged, []string{"write_requested", "verified"}).Order("page_id").Pluck("page_id", &unresolved).Error; err != nil {
			return nil, fmt.Errorf("接管日志排空查询失败")
		}
		return struct {
			UnresolvedBootstrapCount   int64      `json:"unresolved_bootstrap_count"`
			UnresolvedBootstrapPageIDs []string   `json:"unresolved_bootstrap_page_ids"`
			Paused                     bool       `json:"paused"`
			SourceWritesPaused         bool       `json:"source_writes_paused"`
			CurrentRunID               string     `json:"current_run_id"`
			LeaseOwner                 string     `json:"lease_owner"`
			LeaseEpoch                 string     `json:"lease_epoch"`
			LeaseUntil                 *time.Time `json:"lease_until"`
			BaselineFrozen             bool       `json:"baseline_frozen"`
		}{int64(len(unresolved)), unresolved, control.Paused, control.SourceWritesPaused, control.CurrentRunID, control.LeaseOwner, strconv.FormatUint(control.LeaseEpoch, 10), control.LeaseUntil, control.BaselineFrozen}, nil
	case "author_resolve":
		var users []struct{ Nickname string }
		if err := ds.DB().WithContext(ctx).Model(&model.UserM{}).Select("nickname").Limit(2).Find(&users).Error; err != nil {
			return nil, fmt.Errorf("管理员昵称查询失败")
		}
		if len(users) != 1 || strings.TrimSpace(users[0].Nickname) == "" || utf8.RuneCountInString(users[0].Nickname) > 128 || strings.ContainsAny(users[0].Nickname, "\r\n\x00") {
			return nil, fmt.Errorf("需明确且唯一的有效管理员昵称")
		}
		return struct {
			Author string `json:"author"`
		}{users[0].Nickname}, nil
	case "catalog_activate":
		var random [16]byte
		if _, err := rand.Read(random[:]); err != nil {
			return nil, fmt.Errorf("维护租约初始化失败")
		}
		repo := store.NewNotionSyncRepository(ds.DB())
		token, ok, err := repo.AcquireLease(ctx, "cli-catalog-"+hex.EncodeToString(random[:]), time.Minute)
		if err != nil || !ok {
			return nil, fmt.Errorf("目录维护租约不可用")
		}
		defer func() {
			release, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = repo.ReleaseLease(release, token)
		}()
		var result interface{}
		err = store.InTransaction(ctx, ds, func(txStore store.IStore) error {
			if err := requireMaintenanceGate(txStore, token); err != nil {
				return err
			}
			var applyErr error
			result, applyErr = catalog.New(txStore).ActivateReviewedCatalog(ctx, token, in.SourceID, in.Revision, in.Activation)
			if applyErr != nil {
				return applyErr
			}
			_, applyErr = store.LockLease(txStore, token)
			return applyErr
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return nil, fmt.Errorf("未知维护命令")
}

func validateRunArguments(mode, id string) error {
	if mode == "run_status" {
		if !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`).MatchString(id) {
			return fmt.Errorf("运行ID无效")
		}
		return nil
	}
	if id != "" {
		return fmt.Errorf("此模式不接受运行ID")
	}
	return nil
}

func applyBootstrapScope(input *notionsync.BootstrapInput, sourceID string, revision uint64) error {
	normalize := func(v string) string { return strings.ToLower(strings.ReplaceAll(v, "-", "")) }
	if input.SourceID != "" && normalize(input.SourceID) != normalize(sourceID) {
		return fmt.Errorf("确认来源不一致")
	}
	if input.ExpectedConfigRevision != 0 && input.ExpectedConfigRevision != revision {
		return fmt.Errorf("确认版本不一致")
	}
	input.SourceID, input.ExpectedConfigRevision = sourceID, revision
	return nil
}

// A CLI preflight is advisory. The lease and pause checks below fence the same
// transaction as the mutation, including a final DB-time expiry check.
func requireMaintenanceGate(ds store.IStore, token store.LeaseToken) error {
	c, err := store.LockLease(ds, token)
	if err != nil {
		return err
	}
	if !c.Paused || !c.SourceWritesPaused || !c.BaselineFrozen || c.CurrentRunID != "" {
		return fmt.Errorf("维护要求双暂停、冻结基线与已排空任务")
	}
	var unresolved int64
	if err = ds.DB().Model(&model.NotionPageBinding{}).Where("management_state <> ? AND bootstrap_state IN ?", model.NotionManagementManaged, []string{"write_requested", "verified"}).Count(&unresolved).Error; err != nil {
		return err
	}
	if unresolved != 0 {
		return fmt.Errorf("未决接管日志需要先恢复或重新审核")
	}
	return nil
}

func leasedMaintenance(ctx context.Context, ds store.IStore, apply func(store.IStore) (interface{}, error)) (interface{}, error) {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return nil, fmt.Errorf("维护租约初始化失败")
	}
	repo := store.NewNotionSyncRepository(ds.DB())
	token, ok, err := repo.AcquireLease(ctx, "cli-maintenance-"+hex.EncodeToString(random[:]), time.Minute)
	if err != nil || !ok {
		return nil, fmt.Errorf("维护租约不可用")
	}
	defer func() {
		release, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = repo.ReleaseLease(release, token)
	}()
	var result interface{}
	err = store.InTransaction(ctx, ds, func(txStore store.IStore) error {
		if err := requireMaintenanceGate(txStore, token); err != nil {
			return err
		}
		var applyErr error
		result, applyErr = apply(txStore)
		if applyErr != nil {
			return applyErr
		}
		_, applyErr = store.LockLease(txStore, token)
		return applyErr
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
