package zasync_op

import "zinx_server/zinx/zlog"

type AsyncWorker struct {
	taskQ chan func()
}

func (aw *AsyncWorker) process(asyncOp func()) {
	if asyncOp == nil {
		zlog.Error("Async operation is empty.")
		return
	}

	if aw.taskQ == nil {
		zlog.Error("Task queue has not been initialized.")
		return
	}

	aw.taskQ <- func() {
		defer func() {
			if err := recover(); err != nil {
				zlog.Ins().ErrorF("async process panic: %v", err)
			}
		}()
		// 执行异步操作
		asyncOp()
	}
}

func (aw *AsyncWorker) loopExecTask() {
	if aw.taskQ == nil {
		zlog.Error("Task queue has not been initialized.")
		return
	}

	for {
		task := <-aw.taskQ
		if task != nil {
			task()
		}
	}
}
