package ziface

/*
	IRequest接口：实际上是吧客户端请求的链接信息和请求的数据，包装到一个Request中
*/

type IRequest interface {
	//得到当前链接
	GetConnection() IConnection

	//得到请求的消息数据
	GetData() []byte

	//得到请求的消息ID
	GetMsgId() uint32

	// 路由切片集合 操作
	BindRouterSlices([]RouterHandler)

	// 路由切片操作 执行下一个函数
	RouterSlicesNext()
}

type BaseRequest struct{}

func (br *BaseRequest) GetConnection() IConnection { return nil }
func (br *BaseRequest) GetData() []byte            { return nil }
func (br *BaseRequest) GetMsgID() uint32           { return 0 }
func (br *BaseRequest) GetMessage() IMessage       { return nil }

// func (br *BaseRequest) GetResponse() IcResp              { return nil }
// func (br *BaseRequest) SetResponse(resp IcResp)          {}
func (br *BaseRequest) BindRouter(router IRouter) {}
func (br *BaseRequest) Call()                     {}
func (br *BaseRequest) Abort()                    {}

// func (br *BaseRequest) Goto(HandleStep)                  {}
func (br *BaseRequest) BindRouterSlices([]RouterHandler) {}
func (br *BaseRequest) RouterSlicesNext()                {}
