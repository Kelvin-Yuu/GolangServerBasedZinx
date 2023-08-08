package ziface

/*
路由接口， 这里面路由是 使用框架者给该链接自定的 处理业务方法
路由里的IRequest 则包含用该链接的链接信息和该链接的请求数据信息
*/

type IRouter interface {
	//在处理conn业务之前的钩子方法Hook
	PreHandle(request IRequest)
	//在处理conn业务的主方法Hook
	Handle(request IRequest)
	//在处理conn业务之后的钩子方法Hook
	PostHandle(request IRequest)
}

/*
方法切片集合式路由
仅保存路由方法集合，具体执行交给每个请求的IRequest
*/
type RouterHandler func(request IRequest)

type IRouterSlices interface {
	// 添加全局组件
	Use(Handlers ...RouterHandler)

	// 添加业务处理器集合
	AddHandler(msgId uint32, handlers ...RouterHandler)

	// 路由分组管理，并返回一个组管理器
	Group(start, end uint32, handlers ...RouterHandler) IGroupRouterSlices

	// 获取当前的所有注册在MsgId的处理器集合
	GetHandlers(MsgId uint32) ([]RouterHandler, bool)
}

type IGroupRouterSlices interface {
	// 添加全局组件
	Use(Handlers ...RouterHandler)

	// 添加业务处理器集合
	AddHandler(msgId uint32, Handlers ...RouterHandler)
}
