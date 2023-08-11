package znet

import (
	"zinx_server/zinx/ziface"
	"zinx_server/zinx/zinterceptor"
)

/*
	拦截器管理
*/

// 责任链构造器
type chainBuilder struct {
	body       []ziface.IInterceptor
	head, tail ziface.IInterceptor
}

func newChainBuilder() *chainBuilder {
	return &chainBuilder{
		body: make([]ziface.IInterceptor, 0),
	}
}

func (ic *chainBuilder) Head(interceptor ziface.IInterceptor) {
	ic.head = interceptor
}

func (ic *chainBuilder) Tail(interceptor ziface.IInterceptor) {
	ic.tail = interceptor
}

func (ic *chainBuilder) AddInterceptor(interceptor ziface.IInterceptor) {
	ic.body = append(ic.body, interceptor)
}

// 按顺序执行当前链中的所有拦截器
func (ic *chainBuilder) Execute(req ziface.IcReq) ziface.IcResp {
	// 在builder中放入所有拦截器
	var interceptors []ziface.IInterceptor

	if ic.head != nil {
		interceptors = append(interceptors, ic.head)
	}

	if len(ic.body) > 0 {
		interceptors = append(interceptors, ic.body...)
	}

	if ic.tail != nil {
		interceptors = append(interceptors, ic.tail)
	}

	//创建一个责任器链并执行每个拦截器
	chain := zinterceptor.NewChain(interceptors, 0, req)

	return chain.Proceed(req)
}
