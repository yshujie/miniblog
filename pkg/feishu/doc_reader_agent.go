package feishu

import (
	"context"
	"fmt"
	"strings"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkdocs "github.com/larksuite/oapi-sdk-go/v3/service/docs/v1"
)

// Agent 飞书 Agent
type DocReaderAgent interface {
	ReadContent(docToken string, docType string, resultType string) (string, error)
}

// docReaderAgent 文档阅读器 Agent
type docReaderAgent struct {
	client *lark.Client
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

	// 发起请求
	resp, err := d.client.Docs.V1.Content.Get(
		context.Background(),
		req,
		larkcore.WithUserAccessToken("u-dZhzSKHDt4A8Iud4hRDWFX1km2h4kgyrU80051s00aab"),
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
