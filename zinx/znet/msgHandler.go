package znet

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"sync"
	"zinx_server/zinx/zconf"
	"zinx_server/zinx/ziface"
	"zinx_server/zinx/zlog"
)

/* -------------------------------------------------------------------------- */
/*                                  消息处理模块的实现                                 */
/* -------------------------------------------------------------------------- */

const (
	// (如果不启动Worker协程池，则会给MsgHandler分配一个虚拟的WorkerID，这个workerID为0, 便于指标统计
	// 启动了Worker协程池后，每个worker的ID为0,1,2,3...)
	WorkerIDWithoutWorkerPool int = 0
)

// 对消息的处理回调模块
type MsgHandle struct {
	//存放每个MsgID所对应的处理方法
	Apis map[uint32]ziface.IRouter
	//负责Worker取任务的消息队列
	TaskQueue []chan ziface.IRequest

	// 责任链构造器
	builder      *chainBuilder
	RouterSlices *RouterSlices

	//业务工作Worker池的worker数量
	WorkerPoolSize uint32
	// 空闲worker集合，用于zconf.WorkerModeBind
	freeWorkers  map[uint32]struct{}
	freeWorkerMu sync.Mutex
}

// 默认必经的数据处理拦截器
func (mh *MsgHandle) Intercept(chain ziface.IChain) ziface.IcResp {
	request := chain.Request()
	if request != nil {
		switch request.(type) {
		case ziface.IRequest:
			iRequest := request.(ziface.IRequest)
			if zconf.GlobalObject.WorkerPoolSize > 0 {
				// 已经启动工作池机制，将消息交给worker处理
				mh.SendMsgToTaskQueue(iRequest)
			} else {
				// 从绑定好的消息和对应的处理方法中执行对应的Handler方法
				if !zconf.GlobalObject.RouterSlicesMode {
					go mh.doMsgHandle(iRequest, WorkerIDWithoutWorkerPool)
				} else if zconf.GlobalObject.RouterSlicesMode {
					go mh.doMsgHandlerSlices(iRequest, WorkerIDWithoutWorkerPool)
				}
			}

		}
	}
	return chain.Proceed(chain.Request())
}

// 初始化/创建MsgHandle方法
func newMsgHandle() *MsgHandle {
	var freeWorkers map[uint32]struct{}
	if zconf.GlobalObject.WorkerMode == zconf.WorkerModeBind {
		//为每个链接分配一个worker，避免同一worker处理多个链接时的互相影响
		//同时可以减少MaxWorkerTaskLen，比如设置为50，因为每个worker的负担减轻了
		zconf.GlobalObject.WorkerPoolSize = uint32(zconf.GlobalObject.MaxConn)
		freeWorkers = make(map[uint32]struct{}, zconf.GlobalObject.WorkerPoolSize)
		for i := uint32(0); i < zconf.GlobalObject.WorkerPoolSize; i++ {
			freeWorkers[i] = struct{}{}
		}
	}
	handle := &MsgHandle{
		Apis: make(map[uint32]ziface.IRouter),
		// 一个worker对应一个queue
		TaskQueue:      make([]chan ziface.IRequest, zconf.GlobalObject.WorkerPoolSize),
		WorkerPoolSize: zconf.GlobalObject.WorkerPoolSize,
		freeWorkers:    freeWorkers,
		RouterSlices:   NewRouterSlices(),
		builder:        newChainBuilder(),
	}
	// 此处必须把msgHandler 添加到责任链中，并且是责任链的最后一环，在msghandler中进行解码后由router做数据分发
	handle.builder.Tail(handle)
	return handle
}

// 为消息添加具体的处理逻辑
func (mh *MsgHandle) AddRouter(msgID uint32, router ziface.IRouter) {
	//1 判断 当前msg绑定的API处理方式是否已经存在
	if _, ok := mh.Apis[msgID]; ok {
		//id已经注册了
		panic("repeat api, msgID = " + strconv.Itoa(int(msgID)))
	}

	//2 添加msg与API的绑定关系
	mh.Apis[msgID] = router
	fmt.Println("Add api MsgID =", msgID, " successed!")
}

// 切片路由添加
func (mh *MsgHandle) AddRouterSlices(msgId uint32, handler ...ziface.RouterHandler) ziface.IRouterSlices {
	mh.RouterSlices.AddHandler(msgId, handler...)
	return mh.RouterSlices
}

func (mh *MsgHandle) Group(start, end uint32, Handlers ...ziface.RouterHandler) ziface.IGroupRouterSlices {
	return NewGroup(start, end, mh.RouterSlices, Handlers...)
}

func (mh *MsgHandle) Use(Handlers ...ziface.RouterHandler) ziface.IRouterSlices {
	mh.RouterSlices.Use(Handlers...)
	return mh.RouterSlices
}

// 启动一个Worker工作池（开启工作池的动作只能发生一次，一个框架只能有一个工作池）
func (mh *MsgHandle) StartWorkerPool() {
	//根据workerPoolSize 分别开启Worker，每个Work用一个go来承载
	for i := 0; i < int(mh.WorkerPoolSize); i++ {
		//一个Worker被启动

		//1 当前的worker对应的channel消息队列 开辟空间 第i个Worker，就用第i个TaskQueue
		mh.TaskQueue[i] = make(chan ziface.IRequest, zconf.GlobalObject.MaxWorkerTaskLen)
		//2 启动当前的Worker，阻塞等待消息从channel传递过来
		go mh.startOneWorker(i, mh.TaskQueue[i])
	}
}

// 启动一个Worker工作流程
func (mh *MsgHandle) startOneWorker(workerID int, taskQueue chan ziface.IRequest) {
	zlog.Ins().InfoF("WorkerID = %d is Started...", workerID)

	//不断阻塞等待对应消息队列消息
	for {
		select {
		//如果有消息过来，出列的就是一个客户端的Request，执行当前Request所绑定业务
		case request := <-taskQueue:
			switch req := request.(type) {
			case ziface.IRequest:
				if !zconf.GlobalObject.RouterSlicesMode {
					mh.doMsgHandle(req, workerID)
				} else if zconf.GlobalObject.RouterSlicesMode {
					mh.doMsgHandlerSlices(req, workerID)
				}
			}
		}
	}
}

// 将消息交给TaskQueue，由Worker进行处理
func (mh *MsgHandle) SendMsgToTaskQueue(request ziface.IRequest) {
	workerID := request.GetConnection().GetWorkerID()

	mh.TaskQueue[workerID] <- request
	zlog.Ins().DebugF("SendMsgToTaskQueue-->%s", hex.EncodeToString(request.GetData()))
}

// 执行责任链上的拦截器方法
func (mh *MsgHandle) Execute(request ziface.IRequest) {
	mh.builder.Execute(request)
}

// 注册责任链任务入口，每个拦截器处理完后，数据都会传递至下一个拦截器，使得消息可以层层处理层层传递，顺序取决于注册顺序
func (mh *MsgHandle) AddInterceptor(interceptor ziface.IInterceptor) {
	if mh.builder != nil {
		mh.builder.AddInterceptor(interceptor)
	}
}

// 占用workerID
func useWorker(conn ziface.IConnection) uint32 {
	mh, _ := conn.GetMsgHandler().(*MsgHandle)
	if mh == nil {
		zlog.Ins().ErrorF("useWorker failed, mh is nil")
		return 0
	}
	if zconf.GlobalObject.WorkerMode == zconf.WorkerModeBind {
		mh.freeWorkerMu.Lock()
		defer mh.freeWorkerMu.Unlock()

		for k := range mh.freeWorkers {
			delete(mh.freeWorkers, k)
			return k
		}
	}
	// 根据ConnID来分配当前的连接应该由哪个worker负责处理
	// 轮询的平均分配法则
	// 得到需要处理此条连接的workerID
	return uint32(conn.GetConnID() % uint64(mh.WorkerPoolSize))
}

func freeWorker(conn ziface.IConnection) {
	mh, _ := conn.GetMsgHandler().(*MsgHandle)
	if mh == nil {
		zlog.Ins().ErrorF("useWorker failed, mh is nil")
		return
	}
	if zconf.GlobalObject.WorkerMode == zconf.WorkerModeBind {
		mh.freeWorkerMu.Lock()
		defer mh.freeWorkerMu.Unlock()

		mh.freeWorkers[conn.GetWorkerID()] = struct{}{}
	}
}

// 调度/执行对应Router消息处理方法
func (mh *MsgHandle) doMsgHandle(request ziface.IRequest, workerID int) {
	defer func() {
		if err := recover(); err != nil {
			zlog.Ins().ErrorF("workerID: %d doMsgHandler panic: %v", workerID, err)
		}
	}()

	//1 从Request中找到MsgID
	handler, ok := mh.Apis[request.GetMsgID()]
	if !ok {
		zlog.Ins().ErrorF("api msgID = %d is not FOUND!", request.GetMsgID())
		return
	}

	//2 Request请求绑定Router对应关系
	request.BindRouter(handler)
	request.Call()
}

func (mh *MsgHandle) doMsgHandlerSlices(request ziface.IRequest, workerID int) {
	defer func() {
		if err := recover(); err != nil {
			zlog.Ins().ErrorF("workerID: %d doMsgHandler panic: %v", workerID, err)
		}
	}()

	msgId := request.GetMsgID()
	handlers, ok := mh.RouterSlices.GetHandlers(msgId)
	if !ok {
		zlog.Ins().ErrorF("api msgID = %d is not FOUND!", request.GetMsgID())
		return
	}

	request.BindRouterSlices(handlers)
	request.RouterSlicesNext()
}

func (mh *MsgHandle) doFuncHandler(request ziface.IFuncRequest, workerID int) {
	defer func() {
		if err := recover(); err != nil {
			zlog.Ins().ErrorF("workerID: %d doFuncRequest panic: %v", workerID, err)
		}
	}()
	// Execute the functional request (执行函数式请求)
	request.CallFunc()
}
