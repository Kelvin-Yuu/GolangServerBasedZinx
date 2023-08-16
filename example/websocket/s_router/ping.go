package s_router

import (
	"zinx_server/zinx/ziface"
	"zinx_server/zinx/zlog"
	"zinx_server/zinx/znet"
)

// ping test 自定义路由
type PingRouter struct {
	znet.BaseRouter
}

// Ping Handle
func (this *PingRouter) Handle(request ziface.IRequest) {

	zlog.Ins().DebugF("Call PingRouter Handle")
	// Read the data from the client first, then send back "ping...ping...ping".
	zlog.Ins().DebugF("recv from client : msgId=%d, data=%+v, len=%d", request.GetMsgID(), string(request.GetData()), len(request.GetData()))

	err := request.GetConnection().SendBuffMsg(2, []byte("pong-server"))
	if err != nil {
		zlog.Error(err)
	}
}
