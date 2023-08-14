package ziface

import (
	"net/http"
	"time"
)

// 定义一个服务器接口
type IServer interface {
	//启动服务器
	Start()
	//停止服务器
	Stop()
	//运行服务器
	Serve()

	//路由功能：给当前的服务注册一个路由方法，供客户端的链接处理使用
	AddRouter(msgID uint32, router IRouter)

	// 新版路由方式
	AddRouterSlices(msgID uint32, router ...RouterHandler) IRouterSlices

	// 路由组管理
	Group(start, end uint32, Handlers ...RouterHandler) IGroupRouterSlices

	// 公共组件管理
	Use(Handlers ...RouterHandler) IRouterSlices

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

	// 获取Server绑定的数据协议封包方式
	GetPacket() IDataPack

	// 获取Server绑定的消息处理模块
	GetMsgHandler() IMsgHandle

	// 设置Server绑定的数据协议封包方式
	SetPacket(IDataPack)

	//启动心跳检测
	StartHeartBeat(time.Duration)
	//启动心跳检测(自定义回调)
	StartHeartBeatWithOption(time.Duration, *HeartBeatOption)

	GetLengthField() *LengthField
	SetDecoder(IDecoder)
	AddInterceptor(IInterceptor)

	// 获取心跳检测器
	GetHeartBeat() IHeartbeatChecker

	// 添加websocket认证方法
	SetWebsocketAuth(func(r *http.Request) error)

	// 获取服务器名称
	ServerName() string
}
