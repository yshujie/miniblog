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
	Data    interface{} `json:"data"`
}

// WriteResponse 写入响应
func WriteResponse(c *gin.Context, err error, data interface{}) {
	if err != nil {
		httpStatus, errCode, errMsg := errno.Decode(err)
		c.JSON(httpStatus, Response{
			Code:    errCode,
			Message: errMsg,
			Data:    data,
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    "ok",
		Message: "",
		Data:    data,
	})
}
