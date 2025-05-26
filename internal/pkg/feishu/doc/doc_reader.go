package doc

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkdocs "github.com/larksuite/oapi-sdk-go/v3/service/docs/v1"
	"github.com/yshujie/miniblog/internal/pkg/log"
	"golang.org/x/net/html"
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

	// table 转 markdown
	result := d.tableToMarkdown(content)

	log.Infow("successfully parsed content",
		"original", content,
		"parsed", result)
	return result, nil
}

// tableToMarkdown converts an HTML <table> into Markdown format,
// supporting irregular row lengths and escaping Markdown special characters.
func (d *DocReader) tableToMarkdown(content string) string {
	// Parse the HTML content
	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return ""
	}

	// Locate the first <table> node
	var table *html.Node
	var findTable func(*html.Node)
	findTable = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "table" {
			table = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findTable(c)
			if table != nil {
				return
			}
		}
	}
	findTable(doc)
	if table == nil {
		return ""
	}

	// Extract rows (cells without colspan/rowspan handling)
	var rows [][]string
	var parseRows func(*html.Node)
	parseRows = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			var row []string
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && (c.Data == "td" || c.Data == "th") {
					text := extractText(c)
					row = append(row, strings.TrimSpace(text))
				}
			}
			if len(row) > 0 {
				rows = append(rows, row)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			parseRows(c)
		}
	}
	parseRows(table)
	if len(rows) == 0 {
		return ""
	}

	// Determine maximum number of columns
	maxCols := 0
	for _, row := range rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}

	// Pad rows to have equal columns
	for i, row := range rows {
		if len(row) < maxCols {
			pad := make([]string, maxCols-len(row))
			rows[i] = append(row, pad...)
		}
	}

	// Escape Markdown characters in cells
	escapeCell := func(cell string) string {
		// Escape pipe '|' and replace newlines with <br>
		cell = strings.ReplaceAll(cell, "|", "\\|")
		cell = strings.ReplaceAll(cell, "\n", "<br>")
		return cell
	}
	for i := range rows {
		for j := range rows[i] {
			rows[i][j] = escapeCell(rows[i][j])
		}
	}

	// Build Markdown
	var buf bytes.Buffer

	// Header row
	buf.WriteString("|")
	for _, cell := range rows[0] {
		buf.WriteString(" " + cell + " |")
	}

	// Separator row
	buf.WriteString("\n|")
	for range rows[0] {
		buf.WriteString(" --- |")
	}

	// Data rows
	for _, row := range rows[1:] {
		buf.WriteString("\n|")
		for _, cell := range row {
			buf.WriteString(" " + cell + " |")
		}
	}

	return buf.String()
}

// extractText retrieves all text within an HTML node.
func extractText(n *html.Node) string {
	var buf bytes.Buffer
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			buf.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return buf.String()
}
