package v1

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username string `json:"username" valid:"alphanum,required,stringlength(1|255)"`
	Password string `json:"password" valid:"required,stringlength(6|18)"`
	Nickname string `json:"nickname" valid:"required,stringlength(1|255)"`
	Email    string `json:"email" valid:"required,email"`
	Phone    string `json:"phone" valid:"required,stringlength(11|11)"`
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

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" valid:"required,stringlength(6|18)"`
	NewPassword string `json:"newPassword" valid:"required,stringlength(6|18)"`
}
