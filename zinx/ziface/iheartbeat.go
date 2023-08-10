package ziface

type IHeartbeatChecker interface {
	//设置用户自定义的远程连接不存活时的处理方法
	SetOnRemoteNotAlive(OnRemoteNotAlive)
	//设置用户自定义的心跳检测消息处理方法
	SetHeartbeatMsgFunc(HeartBeatMsgFunc)
	//设置用户自定义的远程连接不存活时的处理方法
	SetHeartbeatFunc(HeartBeatFunc)
	//绑定路由
	BindRouter(uint32, IRouter)
	BindRouterSlices(uint32, ...RouterHandler)
	Start()
	Stop()
	//发送心跳报文
	SendHeartBeatMsg() error
	//绑定当前Connection
	BindConn(IConnection)
	Clone() IHeartbeatChecker
	MsgID() uint32
	Router() IRouter
	RouterSlices() []RouterHandler
}

// 用户自定义的心跳检测消息处理方法
type HeartBeatMsgFunc func(IConnection) []byte

// HeartBeatFunc 用户自定义心跳函数
type HeartBeatFunc func(IConnection) error

// 用户自定义的远程连接不存活时的处理方法
type OnRemoteNotAlive func(IConnection)

type HeartBeatOption struct {
	MakeMsg          HeartBeatMsgFunc //用户自定义的心跳检测消息处理方法
	OnRemoteNotAlive OnRemoteNotAlive //用户自定义的远程连接不存活时的处理方法
	HeartBeatMsgID   uint32           //用户自定义的心跳检测消息ID
	Router           IRouter          //用户自定义的心跳检测消息业务处理路由
	RouterSlices     []RouterHandler  //新版本的路由处理函数的集合
}

const (
	HeartBeatDefaultMsgID uint32 = 99999
)
