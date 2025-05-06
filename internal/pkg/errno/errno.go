package errno

import "fmt"

// Errno 定义错误类型
type Errno struct {
	HTTP    int
	Code    string
	Message string
}

// Error 实现 error 接口
func (e *Errno) Error() string {
	return e.Message
}

// SetMessage 设置错误信息
func (e *Errno) SetMessage(format string, args ...interface{}) *Errno {
	e.Message = fmt.Sprintf(format, args...)
	return e
}

// Decode 解码错误
func Decode(err error) (int, string, string) {
	if err == nil {
		return OK.HTTP, OK.Code, OK.Message
	}

	// 根据错误类型解码
	switch typed := err.(type) {
	case *Errno:
		return typed.HTTP, typed.Code, typed.Message
	default:
	}

	// 默认返回未知错误，HTTP 状态码为 500
	return InternalServerError.HTTP, InternalServerError.Code, err.Error()
}
