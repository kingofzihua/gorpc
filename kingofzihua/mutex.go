package main

import (
	"fmt"
	"sync"
)

type T1 struct {
	Name string
	sync.Mutex
}

type T2 struct {
	Name string
	*sync.Mutex
}

func main() {
	t1 := &T1{}
	t2 := &T2{}

	t1.Lock()
	t1.Unlock()
	t2.Lock()
	t2.Unlock()

	fmt.Printf("%v %v", &t1, &t2)
}
