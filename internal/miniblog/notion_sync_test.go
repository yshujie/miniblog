package miniblog

import (
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestNotionSyncRuntimeDefaultsAndInvalidConfiguration(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	opts, err := notionSyncOptions()
	if err != nil || opts.Enabled || opts.Interval != 5*time.Minute || opts.LeaseDuration != 60*time.Second || opts.HeartbeatInterval != 15*time.Second {
		t.Fatalf("defaults: %+v %v", opts, err)
	}
	for _, item := range []struct{ key, value string }{
		{"notion.sync.enabled", "anything"}, {"notion.sync.interval", "0"}, {"notion.sync.interval", "-1m"}, {"notion.sync.interval", "10s"}, {"notion.sync.author", strings.Repeat("中", 129)},
	} {
		viper.Reset()
		viper.Set(item.key, item.value)
		if _, err = notionSyncOptions(); err == nil {
			t.Fatalf("invalid %s accepted", item.key)
		}
	}
}
func TestNotionSyncRuntimeEnvironment(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	loadConfigFromEnv()
	t.Setenv("MINIBLOG_NOTION_SYNC_ENABLED", "true")
	t.Setenv("MINIBLOG_NOTION_SYNC_INTERVAL", "8m")
	t.Setenv("MINIBLOG_NOTION_SYNC_AUTHOR", "舒杰")
	opts, err := notionSyncOptions()
	if err != nil || !opts.Enabled || opts.Interval != 8*time.Minute || opts.Author != "舒杰" {
		t.Fatalf("configuration: %+v %v", opts, err)
	}
}
