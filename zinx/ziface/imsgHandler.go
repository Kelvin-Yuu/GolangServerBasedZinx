package ziface

/* -------------------------------------------------------------------------- */
/*                                   消息管理抽象层                                  */
/* -------------------------------------------------------------------------- */
type IMsgHandle interface {
	////调度/执行对应Router消息处理方法
	//DoMsgHandle(request IRequest)

	//为消息添加具体的处理逻辑
	AddRouter(msgID uint32, router IRouter)

	AddRouterSlices(msgId uint32, handler ...RouterHandler) IRouterSlices
	Group(start, end uint32, Handlers ...RouterHandler) IGroupRouterSlices
	Use(Handlers ...RouterHandler) IRouterSlices

	//启动Worker工作池
	StartWorkerPool()

	//将消息发送给消息任务队列处理
	SendMsgToTaskQueue(request IRequest)

	//执行责任链上的拦截器方法
	Execute(request IRequest)

	// 注册责任链任务入口，每个拦截器处理完后，数据都会传递至下一个拦截器，使得消息可以层层处理层层传递，顺序取决于注册顺序
	AddInterceptor(interceptor IInterceptor)
}
