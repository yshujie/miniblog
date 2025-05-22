package doc

import (
	"context"
	"fmt"
	"strings"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkdocs "github.com/larksuite/oapi-sdk-go/v3/service/docs/v1"
	"github.com/yshujie/miniblog/internal/pkg/log"
)

// DocReader 文档阅读器
type DocReader struct {
	larkClient *lark.Client
}

// NewDocReader 创建文档阅读器
func NewDocReader(larkClient *lark.Client) *DocReader {
	return &DocReader{
		larkClient: larkClient,
	}
}

// ReadContent 读取文档内容
func (d *DocReader) ReadContent(docUrl string, docType string, resultType string) (string, error) {
	// 解析 docToken
	docToken, err := d.parseDocToken(docUrl)
	if err != nil {
		return "", fmt.Errorf("failed to parse doc token: %v", err)
	}

	// 创建请求对象
	req := larkdocs.NewGetContentReqBuilder().
		DocToken(docToken).
		DocType(docType).
		ContentType(resultType).
		Lang(`zh`).
		Build()

	// 发起请求
	resp, err := d.larkClient.Docs.V1.Content.Get(
		context.Background(),
		req,
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

	log.Infow("ReadContent resp", "resp", resp)

	// 返回内容
	content := larkcore.Prettify(resp.Data.Content)
	log.Infow("successfully read doc content", "content_length", len(content))
	log.Infow("content", "content", content)
	return content, nil
}

// 解析 DocToken
func (d *DocReader) parseDocToken(docUrl string) (string, error) {
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
