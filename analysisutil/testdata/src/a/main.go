package main

type st struct {
	o bool
}

func (s *st) open() bool {
	s.o = true
	return s.o
}

func (*st) doSomething() {}

func (s *st) close() {
	s.o = false
}

func test1() {
	var s st
	s.open()
	s.doSomething() // want `close should be called after calling doSomething`
}

func test2() {
	var s st
	s.open()
	s.doSomething()
	s.close()
}

func test3() {
	var s st
	if true {
		s.open()
	} else {
		s.open()
	}
	s.doSomething()
	s.close()
}

func test4() {
	var s st
	if true {
		s.open()
	} else {
		s.close()
	}
	s.doSomething() // want `open should be called before calling doSomething`
	s.close()
}

func test5() {
	var s st
	s.open()
	s.doSomething() // want `close should be called after calling doSomething`
	s.doSomething() // want `close should be called after calling doSomething`
}

func test6() {
	var s st
	if !s.open() {
		return
	}
	s.doSomething() // want `close should be called after calling doSomething`
}
