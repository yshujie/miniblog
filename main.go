package main

import (
	"github.com/yshujie/blog/model"
	"github.com/yshujie/blog/routes"
)

func main() {
	//连接数据库
	model.InitDb()
	model.InitRedis()
	routes.InitRouter()
	//model.SendEmail()
	//fe202a84-64f6-4908-bf7f-cb06e056ac8a
	//CreateUUID()
	//utils.CreateVcode()
}
