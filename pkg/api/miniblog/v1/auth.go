package v1

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" valid:"alphanum,required,stringlength(1|255)"`
	Password string `json:"password" valid:"required,stringlength(6|18)"`
	Nickname string `json:"nickname" valid:"required,stringlength(1|255)"`
	Email    string `json:"email" valid:"required,email"`
	Phone    string `json:"phone" valid:"required,stringlength(11|11)"`
}

// RegisterResponse 注册响应
type RegisterResponse struct {
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" valid:"required,stringlength(1|255)"`
	Password string `json:"password" valid:"required,stringlength(6|18)"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token string `json:"token"`
}

// LogoutRequest 登出请求
type LogoutRequest struct {
	Token string `json:"token"`
}

// LogoutResponse 登出响应
type LogoutResponse struct {
	Message string `json:"message"`
}
