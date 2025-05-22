package feishu

import (
	"context"
	"sync"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	"github.com/yshujie/miniblog/internal/pkg/feishu/auth"
	"github.com/yshujie/miniblog/internal/pkg/feishu/doc"
)

type Client struct {
	appID        string
	appSecret    string
	tokenManager *auth.TokenManager
	larkClient   *lark.Client
	DocReader    *doc.DocReader
}

var (
	client     *Client
	clientOnce sync.Once
)

// GetClient 获取飞书客户端单例
func GetClient(appID, appSecret string, ctx context.Context) *Client {
	// 单例模式，初始化 client
	clientOnce.Do(func() {

		// 创建飞书客户端
		larkClient := lark.NewClient(appID, appSecret)

		client = &Client{
			appID:        appID,
			appSecret:    appSecret,
			larkClient:   larkClient,
			tokenManager: auth.NewTokenManager(larkClient),
			DocReader:    doc.NewDocReader(larkClient),
		}
	})

	// 重新加载新的 token
	if err := client.tokenManager.RefreshToken(ctx, "tenant"); err != nil {
		return nil
	}

	return client
}
