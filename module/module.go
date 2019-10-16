package module

import (
	// "sliv/chanrpc"
	"sync"
)

/**
* 成员状态
* 不可用
* 添加中
* 运行中
* 移除中
 */
const (
	MEM_UNUSE = iota
	MEM_ADDING
	MEM_RUNNING
	MEM_RMVING
)

type member struct {
	memStatus int
	memName   string
}

type module struct {
	memMap map[string]member
}

var (
	ins  *module
	once sync.Once
)

func GetIns() *module {
	once.Do(func() { ins = &module{} })
	return ins
}

func (m *module) AddMember(addr string) {

}

func (m *module) RemoveMember(addr string) {

}
