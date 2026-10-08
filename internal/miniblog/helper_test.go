package miniblog

import (
	"encoding/json"
	"github.com/spf13/viper"
	"strings"
	"testing"
)

func TestConfigLogSummaryUsesAllowlist(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	for _, key := range []string{"database.password", "redis.password", "jwt.secret", "feishu.docreader.appsecret", "notion.token", "future.secret"} {
		viper.Set(key, "sentinel-secret")
	}
	viper.Set("log.level", "info")
	data, err := json.Marshal(safeConfigSummary())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "sentinel-secret") || strings.Contains(string(data), "password") || strings.Contains(string(data), "token") {
		t.Fatal(string(data))
	}
	if !strings.Contains(string(data), "\"log.level\":\"info\"") {
		t.Fatal(string(data))
	}
}
