package ziface

import "time"

//定义一个服务器接口
type IServer interface {
	//启动服务器
	Start()
	//停止服务器
	Stop()
	//运行服务器
	Server()

	//路由功能：给当前的服务注册一个路由方法，供客户端的链接处理使用
	AddRouter(msgID uint32, router IRouter)

	//获取当前server的连接管理器
	GetConnMgr() IConnManager

	//设置该Server的连接创建时Hook函数
	SetOnConnStart(func(connection IConnection))

	//设置该Server的连接断开时的Hook函数
	SetOnConnStop(func(connection IConnection))

	//得到该Server的连接创建时Hook函数
	GetOnConnStart() func(IConnection)

	//得到该Server的连接断开时的Hook函数
	GetOnConnStop() func(IConnection)

	//启动心跳检测
	StartHeartBeat(time.Duration)
	//启动心跳检测(自定义回调)
	StartHeartBeatWithOption(time.Duration, *HeartBeatOption)
}
