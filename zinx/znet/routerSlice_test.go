package znet

import (
	"fmt"
	"testing"
	"zinx_server/zinx/ziface"
)

func A1(request ziface.IRequest) {
	fmt.Println("test A1")
}
func A2(request ziface.IRequest) {
	fmt.Println("test A2")
}
func A3(request ziface.IRequest) {
	fmt.Println("test A3")
}
func A4(request ziface.IRequest) {
	fmt.Println("test A4")
}
func A5(request ziface.IRequest) {
	fmt.Println("test A5")
}
func A6(request ziface.IRequest) {
	fmt.Println("test A6")
}

func TestRouterAdd(t *testing.T) {
	router := NewRouterSlices()
	router.Use(A3)
	router.AddHandler(1, A1, A2)

	testGroup := router.Group(2, 5, A5)
	{
		testGroup.AddHandler(2, A4)
		testGroup.AddHandler(5, A6)
	}

	fmt.Println(router.Apis)

	for _, v := range router.Apis {
		for _, v2 := range v {
			v2(&Request{
				index: -1,
			})
		}
		fmt.Println("==================")

	}
}
