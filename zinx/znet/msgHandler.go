package znet

import (
	"fmt"
	"strconv"
	"zinx_server/zinx/zconf"
	"zinx_server/zinx/ziface"
	"zinx_server/zinx/zlog"
)

/* -------------------------------------------------------------------------- */
/*                                  消息处理模块的实现                                 */
/* -------------------------------------------------------------------------- */
type MsgHandle struct {
	//存放每个MsgID所对应的处理方法
	Apis map[uint32]ziface.IRouter
	//负责Worker取任务的消息队列
	TaskQueue []chan ziface.IRequest

	//业务工作Worker池的worker数量
	WorkerPoolSize uint32

	RouterSlices *RouterSlices
}

// 初始化/创建MsgHandle方法
func NewMsgHandle() *MsgHandle {
	return &MsgHandle{
		Apis:           make(map[uint32]ziface.IRouter),
		TaskQueue:      make([]chan ziface.IRequest, zconf.GlobalObject.WorkerPoolSize),
		WorkerPoolSize: zconf.GlobalObject.WorkerPoolSize,
	}
}

// 调度/执行对应Router消息处理方法
func (mh *MsgHandle) DoMsgHandle(request ziface.IRequest) {
	//1 从Request中找到MsgID
	handler, ok := mh.Apis[request.GetMsgId()]
	if !ok {
		fmt.Println("api msgID = ", request.GetMsgId(), " is NOT FOUND! Need Register!")
		return
	}

	//2 根据MsgID调度对应的Router业务
	handler.PreHandle(request)
	handler.Handle(request)
	handler.PostHandle(request)
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

func (mh *MsgHandle) doMsgHandlerSlices(request ziface.IRequest, workerID int) {
	defer func() {
		if err := recover(); err != nil {
			zlog.Ins().ErrorF("workerID: %d doMsgHandler panic: %v", workerID, err)
		}
	}()

	msgId := request.GetMsgId()
	handlers, ok := mh.RouterSlices.GetHandlers(msgId)
	if !ok {
		zlog.Ins().ErrorF("api msgID = %d is not FOUND!", request.GetMsgId())
		return
	}

	request.BindRouterSlices(handlers)
	request.RouterSlicesNext()
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
					mh.DoMsgHandle(req)
				} else if zconf.GlobalObject.RouterSlicesMode {
					mh.doMsgHandlerSlices(req, workerID)
				}
			}
			//mh.DoMsgHandle(request)
		}
	}
}

// 将消息交给TaskQueue，由Worker进行处理
func (mh *MsgHandle) SendMsgToTaskQueue(request ziface.IRequest) {
	//1 将消息平均分配给不同的Worker
	//根据客户端建立的ConnID来进行分配
	//基本的平均轮询分配法则
	workerID := request.GetConnection().GetConnID() % mh.WorkerPoolSize
	fmt.Println("Add ConnID=", request.GetConnection().GetConnID(), " Request MsgID=", request.GetMsgId(), " To WorkerID=", workerID)

	//2 将消息发送给对应的worker的TaskQueue即可
	mh.TaskQueue[workerID] <- request
}
