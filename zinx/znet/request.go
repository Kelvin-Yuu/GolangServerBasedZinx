package znet

import (
	"sync"
	"zinx_server/zinx/zconf"
	"zinx_server/zinx/ziface"
)

const (
	PRE_HANDLE  ziface.HandleStep = iota // PreHandle for pre-processing
	HANDLE                               // Handle for processing
	POST_HANDLE                          // PostHandle for post-processing

	HANDLE_OVER
)

type Request struct {
	ziface.BaseRequest
	conn     ziface.IConnection     // 已经和客户端建立好的链接
	msg      ziface.IMessage        // 客户端请求的数据
	router   ziface.IRouter         // 请求处理的函数
	steps    ziface.HandleStep      // 用来控制路由函数执行
	stepLock *sync.RWMutex          // 并发互斥
	needNext bool                   // 是否需要执行下一个路由函数
	icResp   ziface.IcResp          // 拦截器返回数据
	handlers []ziface.RouterHandler // 路由函数切片
	index    int8                   // 路由函数切片索引
}

// 得到当前链接
func (r *Request) GetConnection() ziface.IConnection {
	return r.conn
}

// 得到请求的消息数据
func (r *Request) GetData() []byte {
	return r.msg.GetData()
}

// 得到请求的消息ID
func (r *Request) GetMsgID() uint32 {
	return r.msg.GetMsgID()
}

func (r *Request) GetMessage() ziface.IMessage {
	return r.msg
}

func (r *Request) BindRouter(router ziface.IRouter) {
	r.router = router
}

func (r *Request) next() {
	if r.needNext == false {
		r.needNext = true
		return
	}
	r.stepLock.Lock()
	r.steps++
	r.stepLock.Unlock()
}

func (r *Request) Goto(step ziface.HandleStep) {
	r.stepLock.Lock()
	r.steps = step
	r.needNext = false
	r.stepLock.Unlock()
}

func (r *Request) Call() {
	if r.router == nil {
		return
	}

	for r.steps < HANDLE_OVER {
		switch r.steps {
		case PRE_HANDLE:
			r.router.PreHandle(r)
		case HANDLE:
			r.router.Handle(r)
		case POST_HANDLE:
			r.router.PostHandle(r)

		}

	}
	r.steps = PRE_HANDLE
}

func (r *Request) Abort() {
	if zconf.GlobalObject.RouterSlicesMode {
		r.index = int8(len(r.handlers))
	} else {
		r.stepLock.Lock()
		r.steps = HANDLE_OVER
		r.stepLock.Unlock()
	}
}

// 路由切片集合 操作
func (r *Request) BindRouterSlices(handlers []ziface.RouterHandler) {
	r.handlers = handlers
}

// 路由切片操作 执行下一个函数
func (r *Request) RouterSlicesNext() {
	r.index++
	for r.index < int8(len(r.handlers)) {
		r.handlers[r.index](r)
		r.index++
	}
}

func (r *Request) GetResponse() ziface.IcResp {
	return r.icResp
}

func (r *Request) SetResponse(response ziface.IcResp) {
	r.icResp = response
}

func NewRequest(conn ziface.IConnection, msg ziface.IMessage) ziface.IRequest {
	req := new(Request)
	req.steps = PRE_HANDLE
	req.conn = conn
	req.msg = msg
	req.stepLock = new(sync.RWMutex)
	req.needNext = true
	req.index = -1
	return req
}
