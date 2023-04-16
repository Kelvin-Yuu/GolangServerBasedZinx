package ziface

import "net"

//定义链接模块的抽象层
type IConnection interface {
	//启动链接 让当前的链接准备开始工作
	Start()

	//停止链接 结束当前链接的工作
	Stop()

	//获取当前链接的绑定socket conn
	GetTCPConnection() *net.TCPConn

	//获取当前链接模块的链接ID
	GetConnID() uint32

	//获取远程客户端的TCP状态（IP,Port)
	RemoteAddr() net.Addr

	//获取本地服务器的TCP状态（IP,Port）
	LocalAddr() net.Addr

	//发送数据，将数据发送给远程的客户端
	SendMsg(msgId uint32, data []byte) error

	//发送数据，将数据发送给远程的客户端(有缓冲)
	SendBuffMsg(msgId uint32, data []byte) error //添加带缓冲发送消息接口

	//设置链接属性
	SetProperty(key string, value interface{})

	//获取链接属性
	GetProperty(key string) (interface{}, error)

	//移除链接属性
	RemoveProperty(key string)

	//判断当前链接是否存活
	IsAlive() bool

	//设置心跳检测器
	SetHeartBeat(checker IHeartbeatChecker)
}
