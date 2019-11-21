package list

// {{.version}}
// {{.cli}}

type List []int

func NewList() List {
	return List{}
}

func (l *List) Append(n ...int) {
	*l = append(*l, n...)
}
func (l *List) Pop() int {
	k := len(*l)
	if k < 1 {
		panic("list is empty")
	}
	n := (*l)[k]
	*l = (*l)[:k-1]
	return n
}
func (l *List) Push(n int) {
	*l = append(*l, n)
}

//etc...
