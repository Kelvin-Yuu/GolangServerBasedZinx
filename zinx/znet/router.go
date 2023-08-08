package znet

import (
	"strconv"
	"sync"
	"zinx_server/zinx/ziface"
)

// 实现router时，先嵌入这个BaseRouter基类，然后根据需求对这个基类的方法进行重写
type BaseRouter struct {
}

// 在处理conn业务之前的钩子方法Hook
func (br *BaseRouter) PreHandle(request ziface.IRequest) {

}

// 在处理conn业务的主方法Hook
func (br *BaseRouter) Handle(request ziface.IRequest) {

}

// 在处理conn业务之后的钩子方法HookSSS
func (br *BaseRouter) PostHandle(request ziface.IRequest) {

}

type RouterSlices struct {
	Apis     map[uint32][]ziface.RouterHandler
	Handlers []ziface.RouterHandler
	sync.RWMutex
}

func NewRouterSlices() *RouterSlices {
	return &RouterSlices{
		Apis:     make(map[uint32][]ziface.RouterHandler),
		Handlers: make([]ziface.RouterHandler, 0, 0),
	}
}

func (r *RouterSlices) Use(handlers ...ziface.RouterHandler) {
	r.Handlers = append(r.Handlers, handlers...)
}

func (r *RouterSlices) AddHandler(msgId uint32, Handlers ...ziface.RouterHandler) {
	//查找Apis中是否已经注册该handlers，已注册则panic直接返回
	if _, ok := r.Apis[msgId]; ok {
		panic("repeated api, msgId = " + strconv.Itoa(int(msgId)))
	}

	finalSize := len(r.Handlers) + len(Handlers)

	mergedHandlers := make([]ziface.RouterHandler, finalSize)
	copy(mergedHandlers, r.Handlers)
	copy(mergedHandlers[len(r.Handlers):], Handlers)

	r.Apis[msgId] = append(r.Apis[msgId], mergedHandlers...)
}

func (r *RouterSlices) Group(start, end uint32, Handlers ...ziface.RouterHandler) ziface.IGroupRouterSlices {
	return NewGroup(start, end, r, Handlers...)
}

func (r *RouterSlices) GetHandlers(MsgId uint32) ([]ziface.RouterHandler, bool) {
	r.RLock()
	defer r.RUnlock()

	handlers, ok := r.Apis[MsgId]
	return handlers, ok
}

type GroupRouter struct {
	start    uint32
	end      uint32
	Handlers []ziface.RouterHandler
	router   ziface.IRouterSlices
}

func NewGroup(start, end uint32, router *RouterSlices, Handlers ...ziface.RouterHandler) *GroupRouter {
	g := &GroupRouter{
		start:    start,
		end:      end,
		Handlers: make([]ziface.RouterHandler, 0, len(Handlers)),
		router:   router,
	}
	g.Handlers = append(g.Handlers, Handlers...)
	return g
}

func (g *GroupRouter) Use(Handlers ...ziface.RouterHandler) {
	g.Handlers = append(g.Handlers, Handlers...)
}

func (g *GroupRouter) AddHandler(MsgId uint32, Handlers ...ziface.RouterHandler) {
	if MsgId < g.start || MsgId > g.end {
		panic("add router to group err in msgId:" + strconv.Itoa(int(MsgId)))
	}

	finalSize := len(g.Handlers) + len(Handlers)
	mergedHandlers := make([]ziface.RouterHandler, finalSize)
	copy(mergedHandlers, g.Handlers)
	copy(mergedHandlers[len(g.Handlers):], Handlers)

	g.router.AddHandler(MsgId, mergedHandlers...)
}
