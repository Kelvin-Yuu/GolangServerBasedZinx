package ziface

import "time"

/*
定义一个客户端接口
*/
type IClient interface {
	// 重启客户端
	Restart()

	//启动客户端
	Start()

	//停止客户端
	Stop()

	//路由功能：给当前的服务注册一个路由方法，供客户端的链接处理使用
	AddRouter(msgID uint32, router IRouter)

	//获取当前Client的连接
	Conn() IConnection

	//注册OnConnStart hook函数方法
	SetOnConnStart(func(connection IConnection))

	//注册OnConnStop hook函数方法
	SetOnConnStop(func(connection IConnection))

	//获取该Client的连接创建时Hook函数
	GetOnConnStart() func(IConnection)

	//设置该Client的连接断开时的Hook函数
	GetOnConnStop() func(IConnection)

	//获取Client绑定的数据协议封包方法
	GetPacket() IDataPack

	//设置Client绑定的数据协议封包方法
	SetPacket(pack IDataPack)

	//获取Client绑定的消息处理模块
	GetMsgHandler() IMsgHandle

	//启动心跳检测
	StartHeartBeat(duration time.Duration)

	//启动心跳检测（自定义回调）
	StartHeartBeatWithOption(duration time.Duration, option *HeartBeatOption)

	//获取客户端的长度字段
	GetLengthField() *LengthField

	//设置解码器
	SetDecoder(decoder IDecoder)

	//添加拦截器
	AddInterceptor(interceptor IInterceptor)

	//获取客户端错误管道
	GetErrChan() chan error

	//设置客户端Client名称
	SetName(string)

	//获取客户端Client名称
	GetName() string
}
