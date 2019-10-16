package util

import (
	"fmt"
	"testing"
)

type TestCalss struct{}

func (ts *TestCalss) TestFunc() {
	fmt.Println("执行成员公有测试函数")
}

func TestCallClassFuncByName(t *testing.T) {
	test := &TestCalss{}
	CallClassFuncByName(test, "TestFunc")
}

func globalTest() {
	fmt.Println("全局函数测试")
}

func TestCallGlobalFuncByName(t *testing.T) {
	BindGlobalFuncByName("globalTest", globalTest)
	CallGlobalFuncByName("globalTest")
}
