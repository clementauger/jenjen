package main

// jenjen 0.0.0-dev
// jenjen -template=github.com/clementauger/jenjen/_examples/mymap "MyMap:int => string" 

var somethingelse int

type MyMap map[string]string

func NewMyMap() MyMap {
	return MyMap{}
}

func (l *MyMap) Set(k string, v string) {
	(*l)[k] = v
}
func (l *MyMap) Rm(k string) {
	delete(*l, k)
}

//etc...
