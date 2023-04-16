package ziface

/*
	定义一个客户端接口
*/
type IClient interface {
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

	//调用OnConnStart hook函数方法
	CallOnConnStart(connection IConnection)

	//调用OnConnStop hook函数方法
	CallOnConnStop(connection IConnection)
}
