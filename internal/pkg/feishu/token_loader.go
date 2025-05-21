package feishu

import (
	"context"
	"fmt"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkauth "github.com/larksuite/oapi-sdk-go/v3/service/auth/v3"
)

type TokenLoader interface {
	LoadToken(ctx context.Context) (string, error)
}

type tokenLoader struct {
	client    *lark.Client
	appID     string
	appSecret string
}

func NewTokenLoader(appID, appSecret string) TokenLoader {
	return &tokenLoader{
		client:    lark.NewClient(appID, appSecret),
		appID:     appID,
		appSecret: appSecret,
	}
}

func (t *tokenLoader) LoadToken(ctx context.Context) (string, error) {
	// 创建请求对象
	req := larkauth.NewInternalTenantAccessTokenReqBuilder().
		Body(larkauth.NewInternalTenantAccessTokenReqBodyBuilder().
			AppId(t.appID).
			AppSecret(t.appSecret).
			Build()).
		Build()

	// 发起请求
	resp, err := t.client.Auth.V3.TenantAccessToken.Internal(context.Background(), req)

	// 处理错误
	if err != nil {
		return "", err
	}

	// 服务端错误处理
	if !resp.Success() {
		return "", fmt.Errorf("logId: %s, error response: \n%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
	}

	// 解析响应
	var data struct {
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int    `json:"expire"`
	}
	if err := resp.JSONUnmarshalBody(&data, nil); err != nil {
		return "", fmt.Errorf("failed to parse tenant access token response: %v", err)
	}

	return data.TenantAccessToken, nil
}
