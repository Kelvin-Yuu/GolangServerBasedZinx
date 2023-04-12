package main

import (
	"fmt"
	"zinx_server/myDemo/protobufDemo/pb"

	"google.golang.org/protobuf/proto"
)

func main() {
	person := &pb.Person{
		Name:   "KelvinU",
		Age:    24,
		Emails: []string{"kelvinu123456@gmail.com", "2726610715@qq.com"},
		Phones: []*pb.PhoneNumber{
			&pb.PhoneNumber{
				Number: "13100871314",
				Type:   pb.PhoneType_HOME,
			},
			&pb.PhoneNumber{
				Number: "17543430125",
				Type:   pb.PhoneType_MOBILE,
			},
			&pb.PhoneNumber{
				Number: "13176211345",
				Type:   pb.PhoneType_WORK,
			},
		},
	}

	//将person对象（protobuf的message）序列化，得到一个不可读的二进制文件
	data, err := proto.Marshal(person)
	//data 就是要进行网络传输的数据，对端需要按照Message Person格式进行解析
	if err != nil {
		fmt.Println("marshal err:", err)
	}

	//解码
	newPerson := &pb.Person{}

	if err := proto.Unmarshal(data, newPerson); err != nil {
		fmt.Println("Unmarshal err:", err)
	}
	fmt.Println("原始数据：", person)
	fmt.Println("解码数据：", newPerson)
}
