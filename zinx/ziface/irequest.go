package ziface

type HandleStep int

type IFuncRequest interface {
	CallFunc()
}

/*
	IRequest接口：实际上是吧客户端请求的链接信息和请求的数据，包装到一个Request中
*/

type IRequest interface {
	//得到当前链接
	GetConnection() IConnection

	//得到请求的消息数据
	GetData() []byte

	//得到请求的消息ID
	GetMsgID() uint32

	//获取请求消息的原始数据
	GetMessage() IMessage

	//获取解析完后序列化数据
	GetResponse() IcResp
	//设置解析完后序列化数据
	SetResponse(IcResp)

	//绑定这次请求由哪个路由处理
	BindRouter(router IRouter)

	//转进到下一个处理器开始执行 但是调用此方法的函数会根据先后顺序逆序执行
	Call()
	// 终止处理函数的运行 但调用此方法的函数会执行完毕
	Abort()
	// 指定接下来的Handle去执行哪个Handler函数
	// 慎用，会导致循环调用
	Goto(HandleStep)

	// 路由切片集合 操作
	BindRouterSlices([]RouterHandler)

	// 路由切片操作 执行下一个函数
	RouterSlicesNext()
}

type BaseRequest struct{}

func (br *BaseRequest) GetConnection() IConnection       { return nil }
func (br *BaseRequest) GetData() []byte                  { return nil }
func (br *BaseRequest) GetMsgID() uint32                 { return 0 }
func (br *BaseRequest) GetMessage() IMessage             { return nil }
func (br *BaseRequest) GetResponse() IcResp              { return nil }
func (br *BaseRequest) SetResponse(resp IcResp)          {}
func (br *BaseRequest) BindRouter(router IRouter)        {}
func (br *BaseRequest) Call()                            {}
func (br *BaseRequest) Abort()                           {}
func (br *BaseRequest) Goto(HandleStep)                  {}
func (br *BaseRequest) BindRouterSlices([]RouterHandler) {}
func (br *BaseRequest) RouterSlicesNext()                {}
