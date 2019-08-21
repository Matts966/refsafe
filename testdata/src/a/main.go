package main

import "reflect"

func test1(i *interface{}) {
	rv := reflect.ValueOf(i)
	rv.CanAddr()
	rv.Addr()
}

func test2(i *interface{}) {
	rv := reflect.ValueOf(i)
	rv.Addr() // want `reflect.CanAddr should be called before calling reflect.Addr`
}
