package feishu

import (
	"context"
	"fmt"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkauth "github.com/larksuite/oapi-sdk-go/v3/service/auth/v3"
	"github.com/yshujie/miniblog/internal/pkg/log"
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
	log.Infow("start to load tenant access token", "app_id", t.appID)

	// 创建请求对象
	req := larkauth.NewInternalTenantAccessTokenReqBuilder().
		Body(larkauth.NewInternalTenantAccessTokenReqBodyBuilder().
			AppId(t.appID).
			AppSecret(t.appSecret).
			Build()).
		Build()

	// 发起请求
	resp, err := t.client.Auth.V3.TenantAccessToken.Internal(context.Background(), req)
	if err != nil {
		log.Errorw("failed to get tenant access token", "error", err)
		return "", fmt.Errorf("failed to get tenant access token: %v", err)
	}

	// 服务端错误处理
	if !resp.Success() {
		log.Errorw("failed to get tenant access token", "error", larkcore.Prettify(resp.CodeError))
		return "", fmt.Errorf("failed to get tenant access token: %s", larkcore.Prettify(resp.CodeError))
	}

	log.Infow("successfully got tenant access token", "resp: ", resp)
	// // 解析响应
	// var data struct {
	// 	TenantAccessToken string `json:"tenant_access_token"`
	// 	Expire            int    `json:"expire"`
	// }
	// if err := resp.JSONUnmarshalBody(&data, nil); err != nil {
	// 	log.Errorw("failed to parse tenant access token response", "error", err)
	// 	return "", fmt.Errorf("failed to parse tenant access token response: %v", err)
	// }

	// log.Infow("successfully loaded tenant access token", "expire", data.Expire)
	// return data.TenantAccessToken, nil

	return "123", nil
}
