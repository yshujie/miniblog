package errno

var (
	// OK 成功
	OK = &Errno{HTTP: 200, Code: "OK", Message: "OK"}

	// InternalServerError 服务器内部错误
	InternalServerError = &Errno{HTTP: 500, Code: "InternalServerError", Message: "服务器内部错误"}

	// ErrPageNotFound 页面未找到
	ErrPageNotFound = &Errno{HTTP: 404, Code: "PageNotFound", Message: "页面未找到"}

	// ErrUserAlreadyExists 用户已存在
	ErrUserAlreadyExists = &Errno{HTTP: 400, Code: "UserAlreadyExists", Message: "用户已存在"}

	// ErrBind 请求参数错误
	ErrBind = &Errno{HTTP: 400, Code: "ErrBind", Message: "请求参数错误"}

	// ErrInvalidParameter 请求参数错误
	ErrInvalidParameter = &Errno{HTTP: 400, Code: "ErrInvalidParameter", Message: "请求参数错误"}

	// ErrInvalidToken 无效的 token
	ErrInvalidToken = &Errno{HTTP: 401, Code: "ErrInvalidToken", Message: "无效的 token"}

	// ErrTokenSign 签发 token 失败
	ErrTokenSign = &Errno{HTTP: 500, Code: "ErrTokenSign", Message: "签发 token 失败"}

	// ErrPasswordIncorrect 密码错误
	ErrPasswordIncorrect = &Errno{HTTP: 401, Code: "ErrPasswordIncorrect", Message: "密码错误"}

	// ErrUserNotFound 用户不存在
	ErrUserNotFound = &Errno{HTTP: 404, Code: "ErrUserNotFound", Message: "用户不存在"}

	// ErrUnauthorized 未授权
	ErrUnauthorized = &Errno{HTTP: 401, Code: "ErrUnauthorized", Message: "未授权"}

	// ErrModuleAlreadyExists 模块已存在
	ErrModuleAlreadyExists = &Errno{HTTP: 400, Code: "ErrModuleAlreadyExists", Message: "模块已存在"}

	// ErrSectionAlreadyExists section 已存在
	ErrSectionAlreadyExists = &Errno{HTTP: 400, Code: "ErrSectionAlreadyExists", Message: "section 已存在"}

	// ErrSectionNotFound section 不存在
	ErrSectionNotFound = &Errno{HTTP: 404, Code: "ErrSectionNotFound", Message: "section 不存在"}

	// ErrModuleNotFound module 不存在
	ErrModuleNotFound = &Errno{HTTP: 404, Code: "ErrModuleNotFound", Message: "module 不存在"}

	// ErrReadDocFailed 读取文档失败
	ErrReadDocFailed = &Errno{HTTP: 500, Code: "ErrReadDocFailed", Message: "读取文档失败"}

	// ErrFeishu 飞书错误
	ErrFeishuTokenRefreshFailed = &Errno{HTTP: 500, Code: "ErrFeishuTokenRefreshFailed", Message: "飞书 token 刷新失败"}

	// ErrArticleNotFound 文章不存在
	ErrArticleNotFound = &Errno{HTTP: 404, Code: "ErrArticleNotFound", Message: "文章不存在"}

	// ErrUpdateArticleFailed 更新文章失败
	ErrUpdateArticleFailed = &Errno{HTTP: 500, Code: "ErrUpdateArticleFailed", Message: "更新文章失败"}
)
