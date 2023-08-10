package ziface

// 拦截器输入数据
type IcReq interface{}

// 拦截器输出数据
type IcResp interface{}

// 拦截器
type IInterceptor interface {
	//拦截器的拦截处理方法,由开发者定义
	Interceptor(IChain) IcResp
}

// 责任链
type IChain interface {
	Request() IcReq        // 获取当前责任链中的请求数据(当前拦截器)
	GetIMessage() IMessage // 从Chain中获取IMessage
	Proceed(IcReq) IcResp  // 进入并执行下一个拦截器，且将请求数据传递给下一个拦截器
	ProceedWithIMessage(IMessage, IcReq) IcResp
}
