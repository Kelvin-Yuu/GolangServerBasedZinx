package main

import "zinx_server/zinx/znet"

func main() {
	//创建server句柄
	s := znet.NewServer("MMO Game Server")

	//连接创建和销毁Hook函数

	//注册路由

	//启动服务
	s.Start()
}
