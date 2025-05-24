package v1

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username     string `json:"username" valid:"alphanum,required,stringlength(1|255)"`
	Password     string `json:"password" valid:"required,stringlength(6|18)"`
	Nickname     string `json:"nickname" valid:"required,stringlength(1|255)"`
	Avatar       string `json:"avatar" valid:"required,stringlength(1|255)"`
	Email        string `json:"email" valid:"required,email"`
	Phone        string `json:"phone" valid:"required,stringlength(11|11)"`
	Introduction string `json:"introduction" valid:"required,stringlength(1|1024)"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" valid:"required,stringlength(6|18)"`
	NewPassword string `json:"newPassword" valid:"required,stringlength(6|18)"`
}

// GetUserResponse 指定了 `GET /v1/users/{name}` 接口的返回参数.
type GetUserResponse struct {
	User UserInfo `json:"user"`
}

// UserInfo 指定了用户的详细信息.
type UserInfo struct {
	Username     string   `json:"username"`
	Nickname     string   `json:"nickname"`
	Avatar       string   `json:"avatar"`
	Introduction string   `json:"introduction"`
	Email        string   `json:"email"`
	Phone        string   `json:"phone"`
	Roles        []string `json:"roles"`
	PostCount    int64    `json:"postCount"`
	CreatedAt    string   `json:"createdAt"`
	UpdatedAt    string   `json:"updatedAt"`
}
