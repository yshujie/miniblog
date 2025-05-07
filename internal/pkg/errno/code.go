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
)
