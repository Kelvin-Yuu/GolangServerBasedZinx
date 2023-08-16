package c_router

import (
	"zinx_server/zinx/ziface"
	"zinx_server/zinx/zlog"
	"zinx_server/zinx/znet"
)

// Ping test custom routing.
type PingRouter struct {
	znet.BaseRouter
}

// Ping Handle
func (this *PingRouter) Handle(request ziface.IRequest) {
	zlog.Debug("Call PingRouter Handle")
	zlog.Debug("recv from server : msgId=", request.GetMsgID(), ", data=", string(request.GetData()))

	err := request.GetConnection().SendBuffMsg(100, []byte("ping-client"))
	if err != nil {
		zlog.Error(err)
	}
}
