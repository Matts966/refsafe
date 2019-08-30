package main

import (
	"reflect"
	"unsafe"
)

func test1(i *interface{}) {
	rv := reflect.ValueOf(i)
	rv.Addr() // want `CanAddr should be called before calling Addr`
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
	rv.Interface() // want `CanInterface should be called before calling Interface`
}

func test5(i *interface{}) {
	rv := reflect.ValueOf(i)
	var rv2 reflect.Value
	rv.Set(rv2) // want `CanSet should be called before calling Set`
}

func test6(i *interface{}) {
	rv := reflect.ValueOf(i)
	var rv2 unsafe.Pointer
	if rv.CanSet() {
		rv.SetPointer(rv2) // want `Kind should be UnsafePointer when calling SetPointer`
	}
}

func test7(i *interface{}) {
	rv := reflect.ValueOf(i)
	var rv2 unsafe.Pointer
	if rv.CanSet() && rv.Kind() == reflect.Interface {
		rv.SetPointer(rv2) // want `Kind should be UnsafePointer when calling SetPointer`
	}
}

func test8(i *interface{}) {
	rv := reflect.ValueOf(i)
	var rv2 unsafe.Pointer
	if rv.CanSet() && rv.Kind() == reflect.UnsafePointer {
		rv.SetPointer(rv2)
	}
}

func test9(i *interface{}) {
	rv := reflect.ValueOf(i)
	var rv2 string
	if rv.CanSet() && rv.Kind() == reflect.UnsafePointer {
		rv.SetString(rv2) // want `Kind should be String when calling SetString`
	}
}
