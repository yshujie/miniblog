package doc

import (
	"context"
	"fmt"
	"strconv"
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
	log.Infow("read doc", "docUrl", docUrl, "docType", docType, "resultType", resultType)
	resp, err := d.larkClient.Docs.V1.Content.Get(
		context.Background(),
		req,
	)
	log.Infow("read doc", "resp", resp)
	if err != nil {
		return "", fmt.Errorf("failed to read doc: %v", err)
	}

	// 服务端错误处理
	if !resp.Success() {
		return "", fmt.Errorf("failed to read doc: %s", larkcore.Prettify(resp.CodeError))
	}

	// 返回内容
	content := *resp.Data.Content
	log.Infow("read doc", "content", content)

	// 解析 content 中的 ASCII 码
	parsedContent, err := d.parseContent(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse content: %v", err)
	}

	return parsedContent, nil
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

// 解析 content 中的 ASCII 码
func (d *DocReader) parseContent(content string) (string, error) {
	// 记录原始内容，用于调试
	log.Infow("parsing content", "original", content)

	// 如果内容为空，直接返回
	if content == "" {
		return "", nil
	}

	// 使用 strconv.Unquote 处理转义符、ASCII 码
	unquoted, err := strconv.Unquote(`"` + content + `"`)
	if err != nil {
		log.Errorw("failed to unquote content",
			"error", err,
			"content", content,
			"content_length", len(content))
		// 如果解析失败，返回原始内容
		return content, nil
	}

	log.Infow("successfully parsed content",
		"original", content,
		"parsed", unquoted)
	return unquoted, nil
}
