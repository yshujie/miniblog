package token

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

// Config 包含 token 包的配置选项
type Config struct {
	key         string
	identityKey string
}

// ErrMissingHeader 表示`Authorization`头为空
var ErrMissingHeader = errors.New("the length of the `Authorization` header is zero")

var (
	config = Config{"Rtg8BPKNEf2mB4mgvKONGPZZQSaJWNLijxR42qRgq0iBb5", "identityKey"}
	once   sync.Once
)

// Init 设置包级别的配置 config，config 会用于本包后面的 token 签发和解析
func Init(key string, identityKey string) {
	once.Do(func() {
		if key != "" {
			config.key = key
		}
		if identityKey != "" {
			config.identityKey = identityKey
		}
	})
}

// Sign 签发，使用 jwtSecret 签发 token，token 的 claims 中存放传入的 subject
func Sign(identityKey string) (tokenString string, err error) {
	// Token 的内容
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		config.identityKey: identityKey,                               // 用户身份标识
		"nbf":              time.Now().Unix(),                         // 生效时间
		"iat":              time.Now().Unix(),                         // 签发时间
		"exp":              time.Now().Add(100000 * time.Hour).Unix(), // 过期时间
	})

	// 签发 token
	tokenString, err = token.SignedString([]byte(config.key))

	return
}

// Parse 解析 token
func Parse(tokenString string, key string) (string, error) {
	// 解析 token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// 验证 token 的签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}

		return []byte(config.key), nil
	})
	// 解析失败
	if err != nil {
		return "", err
	}

	// 解析成功，获取 claims
	var identityKey string
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		identityKey = claims[config.identityKey].(string)
	}

	return identityKey, nil
}

// ParseRequest 从请求头中获取令牌，并将其传递给 Parse 函数
func ParseRequest(c *gin.Context) (string, error) {
	header := c.Request.Header.Get("Authorization")
	if len(header) == 0 {
		return "", ErrMissingHeader
	}

	var token string
	fmt.Sscanf(header, "Bearer %s", &token)
	if token == "" {
		return "", ErrMissingHeader
	}

	return Parse(token, config.key)
}
