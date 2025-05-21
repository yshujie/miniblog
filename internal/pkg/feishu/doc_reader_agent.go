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
	client := lark.NewClient(appID, appSecret)
	return &docReaderAgent{client: client}, nil
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
	log.Infow("tenantAccessToken", "tenantAccessToken", tenantAccessToken)
	if err != nil {
		return "", fmt.Errorf("failed to load tenantAccessToken: %s", err)
	}

	// 发起请求
	resp, err := d.client.Docs.V1.Content.Get(
		context.Background(),
		req,
		larkcore.WithTenantAccessToken(tenantAccessToken),
	)
	if err != nil {
		return "", fmt.Errorf("failed to read doc: %s", err)
	}

	// 服务端错误处理
	if !resp.Success() {
		return "", fmt.Errorf("failed to read doc: %s", larkcore.Prettify(resp.CodeError))
	}

	// 返回内容
	return larkcore.Prettify(resp.Data.Content), nil
}
