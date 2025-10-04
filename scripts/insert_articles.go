package main

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

// 请根据实际情况修改
const (
	host     = "mysql"
	port     = 3306
	user     = "miniblog"
	password = "2gy0dCwG"
	dbname   = "miniblog"
)

var articles = []struct {
	ID           int64
	Title        string
	Content      string
	ExternalLink string
	SectionCode  string
	Author       string
	Tags         string
	Pos          int
	Status       int
	CreatedAt    string
	UpdatedAt    string
}{
	{573483961993409070, "事件风暴实践指南：以事件为中心的业务建模方法", "", "https://vdbapmvz0i.feishu.cn/docx/ZYaUdZWr1o8u27xcgp8cEZrFnYf?from=from_copylink", "analysis", "clack", "领域事件,领域命令,领域名词", 2, 2, "2025-06-18 14:43:12", "2025-08-12 14:13:12"},
	{573483962110849582, "分层架构全景解析：从单体到领域驱动的演进之路", "", "https://vdbapmvz0i.feishu.cn/docx/HYCid9j8soGR4lxrcjOcVTdlnDM", "architecture_base", "clack", "mvc,六边形架构", 1, 2, "2025-06-18 14:44:16", "2025-08-12 14:13:12"},
	{573483962228290094, "「 问卷&量表」系统软件架构设计", "", "", "ai_prompt", "clack", "架构设计", 0, 1, "2025-06-19 15:12:02", "2025-07-10 11:15:16"},
	{573941328732238382, "Go 语言的诞生：在 C 语言的基础上重塑编程的未来", "", "https://vdbapmvz0i.feishu.cn/docx/LHjidY6dBo22vGxfaFMcRQiXnd2", "go_interesting", "clack", "编译型语言", 1, 2, "2025-06-18 13:52:31", "2025-08-12 14:13:12"},
	{574073215433847342, "生命周期（上）：从构建到服务关停的全景", "", "https://vdbapmvz0i.feishu.cn/docx/KwGLdPFGIo1wbax8kJ3cmIulnXb", "go_lifecycle", "clack", "生命周期", 1, 2, "2025-06-18 13:54:12", "2025-08-12 14:13:12"},
	{574341382857044526, "用户故事映射：从用户角度的需求分析", "", "https://vdbapmvz0i.feishu.cn/docx/FyVZdEwNto2qkMxF5oBcNYtxnXd", "analysis", "clack", "用户故事,用户行为", 1, 2, "2025-06-18 14:41:51", "2025-08-12 14:13:12"},
	{574517839474471470, "静态类型 vs 强类型，到底有啥区别？", "", "https://vdbapmvz0i.feishu.cn/docx/LB0gd8bF5oqe3cxQXFZcQixvn3e", "go_interesting", "clack", "静态/动态类型语言,强类型/弱类型语言", 2, 2, "2025-06-23 13:27:59", "2025-08-12 14:13:12"},
	{574518135256789550, "项目首页", "", "https://vdbapmvz0i.feishu.cn/docx/Ef2OdJMMfoq1OGxWFtFcJMLonjh", "qs", "clack", "调查问卷,医学量表", 1, 2, "2025-06-23 14:07:51", "2025-08-12 14:29:39"},
	{574518224796791342, "事件风暴分析笔记（原始行为→领域事件映射）", "", "https://vdbapmvz0i.feishu.cn/docx/AQx6dAleioLnIYxKFescjxlknvb", "qs", "clack", "调查问卷,医学量表", 2, 2, "2025-06-23 14:13:09", "2025-08-12 14:29:39"},
	{574518281872880174, "问卷&量表领域建模（领域模型、词汇表、规则表）", "", "https://vdbapmvz0i.feishu.cn/docx/Wx2xdcjIzolpsHxGUZTc75fCnwW", "qs", "clack", "领域建模", 3, 2, "2025-06-23 19:24:49", "2025-08-12 14:29:39"},
	{574518322037535278, "系统架构总览（分层架构 + 模块职责）", "", "https://vdbapmvz0i.feishu.cn/docx/Lzx9drS5eoksnJxyyL8cChqUn1g", "qs", "clack", "架构设计", 4, 2, "2025-06-23 19:25:58", "2025-08-12 14:29:39"},
	{574518771465597486, "项目首页", "", "https://vdbapmvz0i.feishu.cn/docx/FGLSdUjHYoSUZRxtr1Bc4vpXnzf", "online_consultation", "clack", "在线问诊", 1, 2, "2025-06-23 19:32:22", "2025-08-12 14:13:38"},
	{574518790960722478, "「在线问诊」模块需求分析", "", "https://vdbapmvz0i.feishu.cn/docx/CBLEdsWEyojzSjxMkTdcJ8qZnDe", "online_consultation", "clack", "在线问诊", 2, 2, "2025-06-23 19:33:04", "2025-08-12 14:13:38"},
	{574518835739111982, "「在线问诊」模块领域建模", "", "https://vdbapmvz0i.feishu.cn/docx/HKnLdekiKoOlpuxGxGdcdjCznpg", "online_consultation", "clack", "在线问诊", 3, 2, "2025-06-23 19:33:43", "2025-08-12 14:13:38"},
	{574519017050485294, "「电子处方」模块需求分析", "", "https://vdbapmvz0i.feishu.cn/docx/E5WidJCNqoOsUdxytZEcf81vn9b", "online_consultation", "clack", "电子处方", 4, 2, "2025-06-23 19:34:24", "2025-08-12 14:13:38"},
	{574521019050485296, "「电子处方」模块领域建模", "", "https://vdbapmvz0i.feishu.cn/docx/J9Wrd8hegoBtnuxwT57cbDcJnmc", "online_consultation", "clack", "电子处方", 5, 2, "2025-06-23 19:35:02", "2025-08-12 14:13:38"},
	{575352612975555118, "设计原则 -- 单一职责原则", "", "https://vdbapmvz0i.feishu.cn/docx/DG5ndUXddo26zdxRQMXcn9wpnRb", "design_pattern", "clack", "单一职责,设计原则", 1, 2, "2025-07-14 12:14:41", "2025-08-12 14:13:12"},
	{575353036734476846, "Go 语言为什么不支持三元表达式 ？", "", "https://vdbapmvz0i.feishu.cn/docx/JwVMdiEO5odWfpxqwlvckO1Tnze", "go_interesting", "clack", "三元表达式", 3, 2, "2025-07-14 12:18:54", "2025-08-12 14:13:13"},
	{575790201524204078, "六边形架构与模块化设计", "", "https://vdbapmvz0i.feishu.cn/docx/OWd5dnf3NoHOibxkuEScSv2HnDd", "architecture_base", "clack", "架构设计,模块化", 2, 2, "2025-07-17 12:41:44", "2025-08-12 14:13:12"},
	{575806912218542638, "“值传递”与“指针传递”到底该如何选择？", "", "https://vdbapmvz0i.feishu.cn/docx/FYv7d6861okcDUxjTH5cotLUnJb", "go_interesting", "clack", "值传递与指针传递", 4, 2, "2025-07-17 15:27:45", "2025-08-12 15:02:51"},
	{576514688787952174, "设计原则 -- 开闭原则", "", "https://vdbapmvz0i.feishu.cn/docx/CTNKdrVofodeNoxU6w9c3Ekmnvc", "design_pattern", "clack", "开闭原则", 2, 2, "2025-07-22 12:38:52", "2025-08-12 14:13:12"},
	{578111354439741998, "生命周期（中）：编译器执行流程全解析", "", "https://vdbapmvz0i.feishu.cn/docx/Iw2RdMLptox4VoxfDK0c4v6hnmg", "go_lifecycle", "clack", "编译器", 3, 2, "2025-08-02 13:00:19", "2025-08-12 14:13:12"},
	{578232900051284526, "生命周期（下）：运行时调度与退出机制", "", "https://vdbapmvz0i.feishu.cn/docx/HJ2qdqC7Uoc2B2xYQ6LcJunPn3g", "go_lifecycle", "clack", "运行时", 2, 2, "2025-08-03 09:07:46", "2025-08-12 14:13:12"},
	{578714209538290222, "Go 字符串（上）：结构、编码与类型转换", "", "https://vdbapmvz0i.feishu.cn/docx/J9PgdGtHnoOaZnxEIEgcsqrZnlf", "go_base", "clack", "字符串", 1, 2, "2025-08-06 16:49:09", "2025-08-12 15:45:51"},
	{579442688638595630, "数组与切片（上）：底层原理、性能差异与扩容机制", "", "https://vdbapmvz0i.feishu.cn/docx/EziFdMSKEoOuSqxTg4TcD9LEnhh", "go_base", "clack", "数组,切片", 3, 2, "2025-08-11 17:25:56", "2025-08-12 15:00:31"},
	{579555062699799086, "内存分配器：make 和 new 的本质与区别", "", "https://vdbapmvz0i.feishu.cn/docx/CavFdrIwioCTKZxPlAucsvSgnTc", "go_interesting", "clack", "make", 4, 2, "2025-08-12 12:02:16", "2025-08-12 15:02:39"},
	{579569026913546798, "字符串（下）：字符串的常见操作", "", "https://vdbapmvz0i.feishu.cn/docx/FpRXdsLV2o6orUxpVHicAdyJnRg", "go_base", "clack", "字符串", 2, 2, "2025-08-12 14:21:00", "2025-08-12 14:22:04"},
	{579600881696125486, "Go 散列表（上）：Map 的实现原理与扩容机制", "", "https://vdbapmvz0i.feishu.cn/docx/BAuydieOYoHqWexqsBMcHw3HnFb", "go_base", "clack", "哈希表", 4, 2, "2025-08-12 19:37:27", "2025-08-12 19:37:30"},
	{579606132864070190, "Go 散列表（中）：Map 基本操作、常见陷阱", "", "https://vdbapmvz0i.feishu.cn/docx/ICt5dcXSFoKSXZxcoxOcfCUtnoh", "go_base", "clack", "哈希表", 5, 2, "2025-08-12 20:29:37", "2025-08-12 20:32:15"},
	{579706979635704366, "Go 散列表（下）：Map 与 线程安全", "", "https://vdbapmvz0i.feishu.cn/docx/Xg1udfThZoDWDMxyV3PcPrYan9c", "go_base", "clack", "哈希表", 6, 2, "2025-08-13 13:11:26", "2025-08-13 13:11:29"},
	{580902458532835886, "Go 并发（上）：进程、线程、协程", "", "https://vdbapmvz0i.feishu.cn/docx/QQXSdRaxWoM6oTxlbQlcJ86Ln5f", "go_base", "clack", "并发编程,协程,并发&并行", 7, 2, "2025-08-21 19:07:27", "2025-08-21 19:07:30"},
	{581163679936950830, "Go 并发（中）：协程间的通信机制", "", "https://vdbapmvz0i.feishu.cn/docx/Pv10dMUkrogwp6xkLa2cdS01ntg", "go_base", "clack", "CSP,协程间通信", 8, 2, "2025-08-23 14:22:27", "2025-08-23 14:22:31"},
	{582925797476545070, "Go 并发（下）：Go 调度器 与 GMP 模型", "", "https://vdbapmvz0i.feishu.cn/docx/CNWPdhv0kodsRHxkZ8HcXqSBnbf", "go_base", "clack", "GMP,调度器", 9, 2, "2025-09-04 18:07:31", "2025-09-04 18:07:34"},
}

func main() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, password, host, port, dbname)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	stmt, err := db.Prepare(`INSERT INTO article (id, title, content, external_link, section_code, author, tags, pos, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	for _, a := range articles {
		_, err := stmt.Exec(a.ID, a.Title, a.Content, a.ExternalLink, a.SectionCode, a.Author, a.Tags, a.Pos, a.Status, a.CreatedAt, a.UpdatedAt)
		if err != nil {
			fmt.Printf("插入失败: %v\n", err)
		} else {
			fmt.Printf("插入成功: %s\n", a.Title)
		}
	}
}
