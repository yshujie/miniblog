package auth

import (
	"context"
	"fmt"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkauth "github.com/larksuite/oapi-sdk-go/v3/service/auth/v3"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	"github.com/yshujie/miniblog/internal/pkg/log"
)

// TokenManager 飞书 token 管理器
type TokenManager struct {
	larkClient *lark.Client
}

// NewTokenManager 创建飞书 token 管理器
func NewTokenManager(larkClient *lark.Client) *TokenManager {
	return &TokenManager{
		larkClient: larkClient,
	}
}

// RefreshToken 刷新 token
func (t *TokenManager) RefreshToken(ctx context.Context, tokenType string) error {
	switch tokenType {
	case "tenant":
		err := t.refreshTenantAccessToken(ctx)
		if err != nil {
			log.Errorw("failed to refresh tenant token", "error", err)
			return errno.ErrFeishuTokenRefreshFailed
		}
	}

	return nil
}

// refreshTenantAccessToken 刷新 tenant access token
func (t *TokenManager) refreshTenantAccessToken(ctx context.Context) error {
	// 创建请求对象
	req := larkauth.NewInternalTenantAccessTokenReqBuilder().
		Body(larkauth.NewInternalTenantAccessTokenReqBodyBuilder().
			AppId(`cli_a8a6833e6859501c`).
			AppSecret(`A87ckTk0iNJRSta5zD1XNgqdnbpSoKNv`).
			Build()).
		Build()

	// 发起请求
	resp, err := t.larkClient.Auth.V3.TenantAccessToken.Internal(ctx, req)
	if err != nil {
		return err
	}

	// 服务端错误处理
	if !resp.Success() {
		return fmt.Errorf("failed to refresh tenant token: %s", larkcore.Prettify(resp.CodeError))
	}

	log.Infow("successfully refreshed tenant token", "resp", resp)
	return nil
}
