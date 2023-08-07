package zlog

import "testing"

func TestStdZLog(t *testing.T) {

	//测试 默认debug输出
	Debug("zinx debug content1")
	Debug("zinx debug content2")

	Debugf(" zinx debug a = %d\n", 10)

	//设置log标记位，加上长文件名称 和 微秒 标记
	ResetFlags(BitDate | BitLongFile | BitLevel)
	Info("zinx info content")

	//设置日志前缀，主要标记当前日志模块
	SetPrefix("MODULE")
	Error("zinx error content")

	//添加标记位
	AddFlag(BitShortFile | BitTime)
	Stack(" Zinx Stack! ")

	//设置日志写入文件
	SetLogFile("./log", "testfile.log")
	Debug("===> zinx debug content ~~666")
	Debug("===> zinx debug content ~~888")
	Error("===> zinx Error!!!! ~~~555~~~")

	//调试隔离级别
	Debug("=================================>")
	//1.debug
	SetLogLevel(LogInfo)
	Debug("===> 调试Debug：debug不应该出现")
	Info("===> 调试Debug：info应该出现")
	Warn("===> 调试Debug：warn应该出现")
	Error("===> 调试Debug：error应该出现")
	//2.info
	SetLogLevel(LogWarn)
	Debug("===> 调试Info：debug不应该出现")
	Info("===> 调试Info：info不应该出现")
	Warn("===> 调试Info：warn应该出现")
	Error("===> 调试Info：error应该出现")
	//3.warn
	SetLogLevel(LogError)
	Debug("===> 调试Warn：debug不应该出现")
	Info("===> 调试Warn：info不应该出现")
	Warn("===> 调试Warn：warn不应该出现")
	Error("===> 调试Warn：error应该出现")
	//4.error
	SetLogLevel(LogPanic)
	Debug("===> 调试Error：debug不应该出现")
	Info("===> 调试Error：info不应该出现")
	Warn("===> 调试Error：warn不应该出现")
	Error("===> 调试Error：error不应该出现")
}

func TestZLogger(t *testing.T) {
}
