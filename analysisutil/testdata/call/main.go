package main

type st struct {
	o bool
}

func (s *st) open() {
	s.o = true
}

func (*st) doSomething() {}

func (s *st) close() {
	s.o = false
}

func test1() {
	var s st
	s.doSomething()
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
