package main

import (
	"zinx_server/example/websocket/s_router"
	"zinx_server/zinx/zconf"
	"zinx_server/zinx/znet"
)

func main() {
	zconf.GlobalObject.Mode = ""
	zconf.GlobalObject.LogFile = ""

	s := znet.NewServer()

	s.AddRouter(100, &s_router.PingRouter{})
	s.AddRouter(1, &s_router.HelloZinxRouter{})

	s.Serve()
}
