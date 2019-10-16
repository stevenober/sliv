package chanrpc

import (
	"sync"
)

type chanrpc struct {
}

var ins *chanrpc
var once sync.Once

func GetIns() *chanrpc {
	once.Do(func() { ins = &chanrpc{} })
	return ins
}

// func (c* chanrpc)
