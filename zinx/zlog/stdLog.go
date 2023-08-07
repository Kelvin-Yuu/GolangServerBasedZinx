package zlog

var StdLog = NewLog("", BitDefault)

func Flags() int {
	return StdLog.Flags()
}

func ResetFlags(flag int) {
	StdLog.ResetFlags(flag)
}

func AddFlag(flag int) {
	StdLog.AddFlag(flag)
}

func SetPrefix(prefix string) {
	StdLog.SetPrefix(prefix)
}

func SetLogFile(fileDir string, fileName string) {
	StdLog.SetLogFile(fileDir, fileName)
}

// SetMaxAge 最大保留天数
func SetMaxAge(ma int) {
	StdLog.SetMaxAge(ma)
}

// SetMaxSize 单个日志最大容量 单位：字节
func SetMaxSize(ms int64) {
	StdLog.SetMaxSize(ms)
}

// SetCons 同时输出控制台
func SetCons(b bool) {
	StdLog.SetCons(b)
}

// SetLogLevel sets the log level of StdLog
func SetLogLevel(logLevel int) {
	StdLog.SetLogLevel(logLevel)
}

func Debugf(format string, v ...interface{}) {
	StdLog.Debugf(format, v...)
}

func Debug(v ...interface{}) {
	StdLog.Debug(v...)
}

func Infof(format string, v ...interface{}) {
	StdLog.Infof(format, v...)
}

func Info(v ...interface{}) {
	StdLog.Info(v...)
}

func Warnf(format string, v ...interface{}) {
	StdLog.Warnf(format, v...)
}

func Warn(v ...interface{}) {
	StdLog.Warn(v...)
}

func Errorf(format string, v ...interface{}) {
	StdLog.Errorf(format, v...)
}

func Error(v ...interface{}) {
	StdLog.Error(v...)
}

func Fatalf(format string, v ...interface{}) {
	StdLog.Fatalf(format, v...)
}

func Fatal(v ...interface{}) {
	StdLog.Fatal(v...)
}

func Panicf(format string, v ...interface{}) {
	StdLog.Panicf(format, v...)
}

func Panic(v ...interface{}) {
	StdLog.Panic(v...)
}

func Stack(v ...interface{}) {
	StdLog.Stack(v...)
}

func init() {
	// (因为StdLog对象 对所有输出方法做了一层包裹，所以在打印调用函数的时候，比正常的logger对象多一层调用
	// 一般的Logger对象 calldDepth=2, StdLog的calldDepth=3)
	StdLog.calldDepth = 3
}
