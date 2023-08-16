package ziface

/*
	封包、拆包 模块
	直接面向TCP连接中的数据流，用于处理TCP粘包的问题
*/

type IDataPack interface {
	//获取包头的长度方法
	GetHeadLen() uint32

	//封包方法
	Pack(msg IMessage) ([]byte, error)

	//拆包方式
	Unpack([]byte) (IMessage, error)
}

const (
	// Zinx standard packing and unpacking method (Zinx 标准封包和拆包方式)
	ZinxDataPack    string = "zinx_pack_tlv_big_endian"
	ZinxDataPackOld string = "zinx_pack_ltv_little_endian"

	//...(+)
	//// Custom packing method can be added here(自定义封包方式在此添加)
)

const (
	// Zinx default standard message protocol format(Zinx 默认标准报文协议格式)
	ZinxMessage string = "zinx_message"
)
