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
)
