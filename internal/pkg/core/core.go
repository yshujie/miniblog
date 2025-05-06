package core

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yshujie/miniblog/internal/pkg/errno"
)

type ErrResponse struct {
	// Code 错误码，指定业务错误码
	Code string `json:"code"`

	// Message 错误信息
	Message string `json:"message"`
}

// WriteResponse 写入响应
func WriteResponse(c *gin.Context, err error, data interface{}) {
	if err != nil {
		httpStatus, errCode, errMsg := errno.Decode(err)
		c.JSON(httpStatus, ErrResponse{
			Code:    errCode,
			Message: errMsg,
		})
		return
	}

	c.JSON(http.StatusOK, data)
}
