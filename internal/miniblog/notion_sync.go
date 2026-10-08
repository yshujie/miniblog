package miniblog

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/spf13/viper"
	"github.com/yshujie/miniblog/internal/miniblog/notionsync"
)

func notionSyncOptions() (notionsync.Options, error) {
	opts := notionsync.Options{Interval: 5 * time.Minute, LeaseDuration: 60 * time.Second, HeartbeatInterval: 15 * time.Second}
	if raw := strings.TrimSpace(viper.GetString("notion.sync.enabled")); raw != "" {
		enabled, err := strconv.ParseBool(raw)
		if err != nil {
			return opts, fmt.Errorf("notion.sync.enabled must be a boolean")
		}
		opts.Enabled = enabled
	}
	if raw := strings.TrimSpace(viper.GetString("notion.sync.interval")); raw != "" {
		interval, err := time.ParseDuration(raw)
		if err != nil || interval < time.Minute {
			return opts, fmt.Errorf("notion.sync.interval must be at least one minute")
		}
		opts.Interval = interval
	}
	opts.Author = viper.GetString("notion.sync.author")
	if utf8.RuneCountInString(opts.Author) > 128 {
		return opts, fmt.Errorf("notion.sync.author exceeds 128 characters")
	}
	return opts, nil
}
