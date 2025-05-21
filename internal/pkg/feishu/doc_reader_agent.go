package feishu

import (
	"context"
	"fmt"
	"strings"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkdocs "github.com/larksuite/oapi-sdk-go/v3/service/docs/v1"
	"github.com/yshujie/miniblog/internal/pkg/log"
)

// Agent 飞书 Agent
type DocReaderAgent interface {
	ReadContent(docToken string, docType string, resultType string) (string, error)
}

// docReaderAgent 文档阅读器 Agent
type docReaderAgent struct {
	client      *lark.Client
	tokenLoader TokenLoader
}

// NewDocReaderAgent 创建文档阅读器 Agent
func NewDocReaderAgent(appID, appSecret string) (*docReaderAgent, error) {
	log.Infow("start to create doc reader agent", "app_id", appID)

	// 创建飞书客户端
	client := lark.NewClient(appID, appSecret)
	if client == nil {
		log.Errorw("failed to create lark client")
		return nil, fmt.Errorf("failed to create lark client")
	}

	// 创建 token loader
	tokenLoader := NewTokenLoader(appID, appSecret)
	if tokenLoader == nil {
		log.Errorw("failed to create token loader")
		return nil, fmt.Errorf("failed to create token loader")
	}

	log.Infow("successfully created doc reader agent")
	return &docReaderAgent{
		client:      client,
		tokenLoader: tokenLoader,
	}, nil
}

// 解析 DocToken
func (d *docReaderAgent) ParseDocToken(docUrl string) (string, error) {
	// 判断 docUrl 是否为空
	if docUrl == "" {
		return "", fmt.Errorf("docUrl is empty")
	}
	// 判断 docUrl 是否为空
	if !strings.HasPrefix(docUrl, "https://") {
		return "", fmt.Errorf("docUrl is not a valid url")
	}
	// 拆解 url 获取 docToken
	docToken := strings.Split(docUrl, "/")
	if len(docToken) < 2 {
		return "", fmt.Errorf("docUrl is not a valid url")
	}

	// 最后一位是 docToken
	return docToken[len(docToken)-1], nil
}

// ReadContent 读取文档内容
func (d *docReaderAgent) ReadContent(docToken string, docType string, resultType string) (string, error) {
	// 创建请求对象
	req := larkdocs.NewGetContentReqBuilder().
		DocToken(docToken).
		DocType(docType).
		ContentType(resultType).
		Lang(`zh`).
		Build()

	// 获取 tenantAccessToken
	tenantAccessToken, err := d.tokenLoader.LoadToken(context.Background())
	if err != nil {
		log.Errorw("failed to load tenant access token", "error", err)
		return "", fmt.Errorf("failed to load tenant access token: %v", err)
	}
	log.Infow("successfully loaded tenant access token", "token", tenantAccessToken)

	// 发起请求
	resp, err := d.client.Docs.V1.Content.Get(
		context.Background(),
		req,
		larkcore.WithTenantAccessToken(tenantAccessToken),
	)
	if err != nil {
		log.Errorw("failed to read doc", "error", err)
		return "", fmt.Errorf("failed to read doc: %v", err)
	}

	// 服务端错误处理
	if !resp.Success() {
		log.Errorw("failed to read doc", "error", larkcore.Prettify(resp.CodeError))
		return "", fmt.Errorf("failed to read doc: %s", larkcore.Prettify(resp.CodeError))
	}

	// 返回内容
	content := larkcore.Prettify(resp.Data.Content)
	log.Infow("successfully read doc content", "content_length", len(content))
	return content, nil
}
