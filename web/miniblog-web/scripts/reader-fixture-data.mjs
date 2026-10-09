// Public reader fixtures only. No admin endpoint, persistence or remote request.
export function createReaderFixture(origin = 'http://127.0.0.1:8771') {
  const requests = [];
  const state = { list: 'ok', detailError: false, articleError: false, held: false, waiting: false };
  const article = (id, title, module = 'go', section = 'base', sub = '') => ({
    id, title, module_code: module, section_code: section, subsection_code: sub,
    reading_url: `${origin}/frame/${id}`, external_link: `${origin}/original/${id}`,
    author: 'Shujie', tags: module === 'go' ? ['Go'] : [],
  });
  const articles = [
    article('9007199254740993', '从 Hello World 到程序的执行过程', 'go', 'base', 'start'),
    article('9007199254740994', 'Go 的类型、变量与作用域', 'go', 'base', 'start'),
    article('9007199254740995', '理解函数、闭包与 defer'),
    article('9007199254740996', '为什么 Go 程序需要垃圾回收？', 'go', 'lifecycle'),
    article('9007199254740997', '一个很长的文章标题：从运行时、调度器和并发原语理解 Go 程序的生命周期与工程实践', 'go', 'lifecycle'),
    article('9007199254740998', '从业务问题走向领域模型', 'ddd', 'analysis'),
    article('9007199254740999', '聚合、实体与值对象', 'ddd', 'architecture'),
    article('9007199254741000', '问卷与量表系统的设计', 'project', 'questionnaire'),
  ];
  const section = (id, code, title, module, direct, subsections = []) => ({ id, code, title, module_code: module, articles: direct, subsections });
  const modules = [
    { id: '1', code: 'go', title: 'Golang', sections: [
      section('11', 'base', 'Go 语言基础', 'go', [articles[2]], [{ id: '21', code: 'start', title: '入门与类型', section_code: 'base', articles: articles.slice(0, 2) }]),
      section('12', 'lifecycle', 'Go 语言的生命周期', 'go', articles.slice(3, 5)),
      section('13', 'interesting', '有意思系列', 'go', []),
    ] },
    { id: '2', code: 'ddd', title: '领域驱动设计', sections: [section('14', 'analysis', '需求分析', 'ddd', [articles[5]]), section('15', 'architecture', '架构设计', 'ddd', [articles[6]])] },
    { id: '3', code: 'project', title: '项目', sections: [section('16', 'questionnaire', '问卷 & 量表系统', 'project', [articles[7]]), section('17', 'consultation', '在线问诊', 'project', [])] },
  ];
  const json = (payload, status = 200) => ({ status, contentType: 'application/json; charset=utf-8', body: JSON.stringify(status === 200 ? { code: 'ok', payload } : { code: 'fixture_error', message: '本机验收替身响应', payload: {} }) });
  function handle(rawURL, method = 'GET') {
    const url = new URL(rawURL, origin);
    if (url.pathname.startsWith('/frame/') || url.pathname.startsWith('/original/')) {
      if (state.waiting) return { pending: true };
      return { status: 200, contentType: 'text/html; charset=utf-8', body: '<!doctype html><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><style>body{font:16px/1.9 system-ui,sans-serif;color:#20262e;max-width:720px;margin:40px auto;padding:0 24px}small{color:#59636e}h1{font-size:28px}p{margin:24px 0}hr{border:0;border-top:1px solid #e3e8ed}</style><small>本机验收替身 · 不连接 Notion</small><h1>外部文章展示区域</h1><p>此文档用于验证博客外框、独立滚动和 iframe 稳定性。真实文章仍通过已有 Notion 外部链接展示。</p><hr><h2>阅读与思考</h2>' + '<p>知识需要反复阅读，也需要在实践中验证。这个段落只属于本机浏览器验收资料。</p>'.repeat(14) };
    }
    const pathname = url.pathname.replace(/^\/api/, '');
    if (!pathname.startsWith('/v1/')) return null;
    requests.push({ method, pathname, query: Object.fromEntries(url.searchParams) });
    if (method !== 'GET') return json({}, 405);
    if (pathname === '/v1/blog/modules') return state.list === 'error' ? json({}, 503) : json({ modules: state.list === 'empty' ? [] : modules.map(({ id, code, title }) => ({ id, code, title })) });
    if (pathname === '/v1/blog/moduleDetail') {
      if (state.detailError) return json({}, 503);
      const code = url.searchParams.get('module_code');
      if (code === 'empty') return json({ module_detail: { id: '4', code, title: '空主题', sections: [] } });
      if (code === 'unknown') return json({ module_detail: { id: '5', code, title: '一个来自接口的新主题名称', sections: [] } });
      const module = modules.find(item => item.code === code);
      return module ? json({ module_detail: module }) : json({}, 404);
    }
    if (pathname === '/v1/blog/articleDetail') {
      if (state.articleError) return json({}, 503);
      const selected = articles.find(item => item.id === url.searchParams.get('article_id'));
      return selected && !state.held ? json({ article_detail: selected }) : json({}, 404);
    }
    return json({}, 404);
  }
  return { handle, requests, state, articles, modules };
}
