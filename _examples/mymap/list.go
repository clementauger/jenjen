package mymap

// {{.version}}
// {{.cli}}

var somethingelse int

type MyMap map[string]int

func NewMyMap() MyMap {
	return MyMap{}
}

func (l *MyMap) Set(k string, v int) {
	(*l)[k] = v
}
func (l *MyMap) Rm(k string) {
	delete(*l, k)
}

//etc...
