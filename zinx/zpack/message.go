package zpack

type Message struct {
	DataLen uint32
	ID      uint32
	Data    []byte
	rawData []byte
}

func NewMsgPackage(ID uint32, data []byte) *Message {
	return &Message{
		DataLen: uint32(len(data)),
		ID:      ID,
		Data:    data,
		rawData: data,
	}
}

func NewMessage(len uint32, data []byte) *Message {
	return &Message{
		DataLen: len,
		Data:    data,
		rawData: data,
	}
}

func NewMessageByMsgId(id uint32, len uint32, data []byte) *Message {
	return &Message{
		ID:      id,
		DataLen: len,
		Data:    data,
		rawData: data,
	}
}

func (msg *Message) Init(ID uint32, data []byte) {
	msg.ID = ID
	msg.Data = data
	msg.rawData = data
	msg.DataLen = uint32(len(data))
}

func (msg *Message) GetDataLen() uint32 {
	return msg.DataLen
}

func (msg *Message) GetMsgID() uint32 {
	return msg.ID
}

func (msg *Message) GetData() []byte {
	return msg.Data
}

func (msg *Message) GetRawData() []byte {
	return msg.rawData
}

func (msg *Message) SetDataLen(len uint32) {
	msg.DataLen = len
}

func (msg *Message) SetMsgID(msgID uint32) {
	msg.ID = msgID
}

func (msg *Message) SetData(data []byte) {
	msg.Data = data
}
