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
	client *lark.Client
}

func NewTokenLoader(client *lark.Client) TokenLoader {
	return &tokenLoader{client: client}
}

func (t *tokenLoader) LoadToken(ctx context.Context) (string, error) {
	// 创建请求对象
	req := larkauth.NewInternalTenantAccessTokenReqBuilder().
		Body(larkauth.NewInternalTenantAccessTokenReqBodyBuilder().
			AppId(`cli_slkdjalasdkjasd`).
			AppSecret(`dskLLdkasdjlasdKK`).
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

	return larkcore.Prettify(resp), nil
	// return resp.Data.TenantAccessToken, nil
}
