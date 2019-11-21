package lib

// jenjen 0.0.0-dev
// jenjen -template=github.com/clementauger/jenjen/_examples/mylist "int => string" 

type List []string

func NewList() List {
	return List{}
}

func (l *List) Append(n ...string) {
	*l = append(*l, n...)
}
func (l *List) Pop() string {
	k := len(*l)
	if k < 1 {
		panic("list is empty")
	}
	n := (*l)[k]
	*l = (*l)[:k-1]
	return n
}
func (l *List) Push(n string) {
	*l = append(*l, n)
}

//etc...
