package util

import (
	"errors"
	"reflect"
)

var (
	golbalFuncs = make(map[string]reflect.Value)
)

/**
* 通过方法名来调用类方法
 */
func CallClassFuncByName(obj interface{}, name string, args ...interface{}) (err error, rv []interface{}) {

	objValue := reflect.ValueOf(obj)
	method := objValue.MethodByName(name)

	if !method.IsValid() {
		err = errors.New("CallClassFuncByName type: " + reflect.TypeOf(obj).String() + " func: " + name + " does not exist.")
		return
	}

	if len(args) != method.Type().NumIn() {
		err = errors.New("CallClassFuncByName type: " + reflect.TypeOf(obj).String() + " func: " + name + " The number of args is not adapted.")
		return
	}

	callArgs := make([]reflect.Value, len(args))
	for idx, value := range args {
		callArgs[idx] = reflect.ValueOf(value)
	}
	rs := method.Call(callArgs)
	rv = make([]interface{}, len(rs))
	for i := 0; i < len(rs); i++ {
		rv[i] = rs[i].Interface()
	}
	return
}

/**
* 绑定全局函数对应一个名字
 */
func BindGlobalFuncByName(name string, fn interface{}) (err error) {

	fnValue := reflect.ValueOf(fn)
	if fnValue.Type().Kind() != reflect.Func {
		err = errors.New("BindGlobalFuncByName: " + name + " is not a func.")
		return
	}
	golbalFuncs[name] = fnValue
	return
}

/**
* 通过名字调用全局函数
 */
func CallGlobalFuncByName(name string, args ...interface{}) (err error, rv []interface{}) {
	if _, ok := golbalFuncs[name]; !ok {
		err = errors.New("CallGlobalFuncByName: " + name + " is not exist.")
		return
	}

	if len(args) != golbalFuncs[name].Type().NumIn() {
		err = errors.New("CallGlobalFuncByName: " + name + " The number of params is not adapted.")
		return
	}

	callArgs := make([]reflect.Value, len(args))
	for idx, value := range args {
		callArgs[idx] = reflect.ValueOf(value)
	}
	rs := golbalFuncs[name].Call(callArgs)
	rv = make([]interface{}, len(rs))
	for i := 0; i < len(rs); i++ {
		rv[i] = rs[i].Interface()
	}
	return
}
