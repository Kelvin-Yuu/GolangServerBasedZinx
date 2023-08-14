package zconf

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"
	"zinx_server/zinx/zlog"
	"zinx_server/zinx/zutils/commandline/args"
	"zinx_server/zinx/zutils/commandline/uflag"
)

/*
	存储一切有关框架的全局参数，供其他模块使用
*/

const (
	ServerModeTcp       = "tcp"
	ServerModeWebsocket = "websocket"
	ServerModeKcp       = "kcp"
)

const (
	WorkerModeHash = "Hash" // By default, the round-robin average allocation rule is used.(默认使用取余的方式)
	WorkerModeBind = "Bind" // Bind a worker to each connection.(为每个连接分配一个worker)
)

type Config struct {

	/*
		Serve
	*/
	Host    string //当前服务器主机监听的IP
	TCPPort int    //当前服务器主机TCP监听的端口号
	WsPort  int    //当前服务器主机websocket监听端口
	KcpPort int    //当前服务器主机KCP监听端口号
	Name    string //当前服务器的名称

	/*
		Zinx
	*/
	Version          string //当前Zinx的版本号
	MaxConn          int    //当前服务器主机允许的最大链接数
	MaxPacketSize    uint32 //读写数据包的最大值
	WorkerPoolSize   uint32 //业务工作Worker池的数量
	MaxWorkerTaskLen uint32 //业务工作Worker对应负责的任务队列最大任务存储数量
	MaxMsgChanLen    uint32 //SendBuffMsg发送消息的缓冲最大长度
	WorkerMode       string //为链接分配worker的方式
	IOReadBuffSize   uint32 //每次IO最大的读取长度

	Mode string //"tcp"：TCP监听;"websocket"：websocket监听; 为空则同时开启

	RouterSlicesMode bool //路由模式  false为旧版本，true为新版本 默认旧

	/*
		logger
	*/
	LogDir string // The directory where log files are stored. The default value is "./log".(日志所在文件夹 默认"./log")

	// The name of the log file. If it is empty, the log information will be printed to stderr.
	// (日志文件名称   默认""  --如果没有设置日志文件，打印信息将打印至stderr)
	LogFile string

	LogSaveDays int   // 日志最大保留天数
	LogFileSize int64 // 日志单个日志最大容量 默认 64MB,单位：字节，记得一定要换算成MB（1024 * 1024）
	LogCons     bool  // 日志标准输出  默认 false
	// 日志隔离级别  -- 0：全开 1：关debug 2：关debug/info 3：关debug/info/warn ...
	LogIsolationLevel int

	/*
		Keepalive
	*/
	HeartbeatMax int //最长心跳检测间隔时间(单位：秒),超过改时间间隔，则认为超时，从配置文件读取

	/*
		TLS
	*/
	CertFile       string //证书文件名称 默认""
	PrivateKeyFile string //私钥文件名称 默认"" --如果没有设置证书和私钥文件，则不启用TLS加密
}

/*
定义一个全局的对外GlobalObj
*/
var GlobalObject *Config

// PathExists Check if a file exists.(判断一个文件是否存在)
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// 从zinx.json去加载用于自定义的参数
func (g *Config) Reload() {
	confFilePath := args.Args.ConfigFile
	if confFileExists, _ := PathExists(confFilePath); confFileExists != true {
		g.InitLogConfig()

		zlog.Ins().ErrorF("Config File %s is not exist!!", confFilePath)
		return
	}

	data, err := os.ReadFile(confFilePath)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(data, g)
	if err != nil {
		panic(err)
	}

	g.InitLogConfig()
}

func (g *Config) HeartbeatMaxDuration() time.Duration {
	return time.Duration(g.HeartbeatMax) * time.Second
}

// 显示Config信息
func (g *Config) Show() {
	objVal := reflect.ValueOf(g).Elem()
	objType := reflect.TypeOf(*g)

	fmt.Println("==== Zinx Global Config ====")
	for i := 0; i < objVal.NumField(); i++ {
		field := objVal.Field(i)
		typeField := objType.Field(i)

		fmt.Printf("%s: %v\n", typeField.Name, field.Interface())
	}
	fmt.Println("============================")
}

func (g *Config) InitLogConfig() {
	if g.LogFile != "" {
		zlog.SetLogFile(g.LogDir, g.LogFile)
		zlog.SetCons(g.LogCons)
	}
	if g.LogSaveDays > 0 {
		zlog.SetMaxAge(g.LogSaveDays)
	}
	if g.LogFileSize > 0 {
		zlog.SetMaxSize(g.LogFileSize)
	}
	if g.LogIsolationLevel > zlog.LogDebug {
		zlog.SetLogLevel(g.LogIsolationLevel)
	}
}

// 提供一个init方法，初始化当前的GlobalObject
func init() {
	pwd, err := os.Getwd()
	if err != nil {
		pwd = "."
	}

	args.InitConfigFlag(
		pwd+"/conf/zinx.json",
		"The configuration file defaults to <exeDir>/conf/zinx.json if it is not set.",
	)

	testing.Init()
	uflag.Parse()

	GlobalObject = &Config{
		Name:              "ZinxServerApp",
		Version:           "V1.0",
		TCPPort:           8999,
		WsPort:            9000,
		KcpPort:           9001,
		Host:              "0.0.0.0",
		MaxConn:           12000,
		MaxPacketSize:     4096,
		WorkerPoolSize:    10,
		MaxWorkerTaskLen:  1024,
		WorkerMode:        "",
		MaxMsgChanLen:     1024,
		LogDir:            pwd + "/log",
		LogFile:           "", // if set "", print to Stderr(默认日志文件为空，打印到stderr)
		LogIsolationLevel: 0,
		HeartbeatMax:      10, // The default maximum interval for heartbeat detection is 10 seconds. (默认心跳检测最长间隔为10秒)
		IOReadBuffSize:    1024,
		CertFile:          "",
		PrivateKeyFile:    "",
		Mode:              ServerModeTcp,
		RouterSlicesMode:  false,
	}

	//应该尝试从conf/zinx.json去加载一些用户自定义的参数
	GlobalObject.Reload()
}
