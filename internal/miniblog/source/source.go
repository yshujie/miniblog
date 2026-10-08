// Package source identifies external articles without fetching their body.
package source

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

type Identity struct {
	Provider     string
	CanonicalURL string
	SourceKey    string
	PageID       string
}

var notionID = regexp.MustCompile(`(?i)(?:^|-)([0-9a-f]{32}|[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$`)
var exactNotionID = regexp.MustCompile(`(?i)^([0-9a-f]{32}|[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$`)

func Parse(rawURL string) (Identity, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return Identity{}, fmt.Errorf("外链必须是有效的 http(s) 地址")
	}
	u.Scheme = strings.ToLower(u.Scheme)
	if u.Hostname() == "" || u.User != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return Identity{}, fmt.Errorf("外链必须是有效的 http(s) 地址")
	}
	if len(rawURL) > 8192 {
		return Identity{}, fmt.Errorf("外链过长")
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	if (u.Scheme == "https" && u.Port() == "443") || (u.Scheme == "http" && u.Port() == "80") {
		u.Host = u.Hostname()
		if strings.Contains(u.Host, ":") {
			u.Host = "[" + u.Host + "]"
		}
	}
	identity := Identity{Provider: "other", CanonicalURL: u.String()}
	host := strings.ToLower(u.Hostname())
	notionHost := host == "notion.so" || strings.HasSuffix(host, ".notion.so") || host == "notion.site" || strings.HasSuffix(host, ".notion.site")
	if notionHost {
		identity.Provider = "notion"
		// Database/view URLs can point to a different page through p/v. Preserve
		// their complete URL identity rather than merging on the database path ID.
		path := strings.TrimSuffix(u.Path, "/")
		segment := path[strings.LastIndex(path, "/")+1:]
		if id := notionID.FindStringSubmatch(segment); len(id) > 1 && !u.Query().Has("p") && !u.Query().Has("v") {
			identity.PageID = normalizePageID(id[1])
			identity.CanonicalURL = "https://www.notion.so/" + identity.PageID
		}
	} else if host == "feishu.cn" || strings.HasSuffix(host, ".feishu.cn") || host == "larksuite.com" || strings.HasSuffix(host, ".larksuite.com") {
		identity.Provider = "feishu"
	}
	key := "url:" + identity.CanonicalURL
	if identity.PageID != "" {
		key = "notion:" + identity.PageID
	}
	digest := sha256.Sum256([]byte(key))
	identity.SourceKey = hex.EncodeToString(digest[:])
	return identity, nil
}

func normalizePageID(id string) string { return strings.ToLower(strings.ReplaceAll(id, "-", "")) }
