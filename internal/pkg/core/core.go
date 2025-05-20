package core

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yshujie/miniblog/internal/pkg/errno"
)

// Response 响应结构体
type Response struct {
	Code    string      `json:"code"`
	Message string      `json:"msg"`
	Payload interface{} `json:"payload"`
}

// WriteResponse 写入响应
func WriteResponse(c *gin.Context, err error, payload interface{}) {
	if err != nil {
		httpStatus, errCode, errMsg := errno.Decode(err)
		c.JSON(httpStatus, Response{
			Code:    errCode,
			Message: errMsg,
			Payload: payload,
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    "ok",
		Message: "",
		Payload: payload,
	})
}
