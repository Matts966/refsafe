package main

import (
	"reflect"
	"unsafe"
)

func test1(i *interface{}) {
	rv := reflect.ValueOf(i)
	rv.CanAddr()
	rv.Addr() // want `(reflect.Value).CanAddr should be true when invoking (reflect.Value).Addr`
}

func test2(i *interface{}) {
	rv := reflect.ValueOf(i)
	if !rv.CanAddr() {
		return
	}
	rv.Addr()
}

func test3(i *interface{}) {
	rv := reflect.ValueOf(i)
	if rv.CanAddr() {
		rv.Addr() 
	}
}

func test4(i *interface{}) {
	rv := reflect.ValueOf(i)
	var rv2 unsafe.Pointer
	if rv.CanSet() {
		rv.SetPointer(rv2) // want `(reflect.Value).Kind should be reflect.UnsafePointer when invoking (reflect.Value).SetPointer`
	}
}

func test5(i *interface{}) {
	rv := reflect.ValueOf(i)
	var rv2 unsafe.Pointer
	if rv.CanSet() && rv.Kind() == reflect.UnsafePointer {
		rv.SetPointer(rv2)
	}
}

