# jenjen

Golang code generator.

Halfway between [gofmt -r](https://golang.org/cmd/gofmt/) and [genny](https://github.com/cheekybits/genny)

Unlike `genny` it does use all official golang parser and program loader APIs
for best compatibility and user experience.

Unlike `gofmt` it allows more precise directives.

# usage

```sh
jenjen 0.0.0-dev

golang code generator

jenjen [-h|-help|-t|-skip=..|-template=..|-dst=..|-dl=..|-dr=..] [directives] [-]

  -dl string
    	left delimiter of the template parser (default "{{")
  -dr string
    	right delimiter of the template parser (default "}}")
  -dst string
    	package path of the destination package (default ".")
  -h	show help
  -help
    	show help
  -skip string
    	comma separated list of glob to exclude files from the template package
  -suffix string
    	output file suffix
  -t	render output files using golang template engine (default true)
  -template string
    	package path to the template package
  -version
    	show version

  [directives]
	Directives is a comma separated list of replacements to apply such as [context:]search=>replace(,)+.
	The format of a directive is [context:]search=>replace
	Where search is a case sensitive string matching a type name existing within ast.Ident nodes of the template package.
	Where replace is a valid string value for an ast.Ident node,
	or a dash (-) to signify deletion of the node if it is a function, a method or a type.
	If replace is a fully qualified type path (package/path.type),
	the package path and its type component are identified,
	the package path will be added to the import lists.
	Where context is a case sensitive string matching a type or a function declaration as ast.Decl nodes.
	When context contains a dot, it match a method using the type.method notation.
	Each directive is applied sequentially on the loaded program AST.

  [-]
	do not write on disk, print on stdout

example

  jenjen -template=github.com/clementauger/jenjen/_examples/mymap - \
   "NewMyMap => NewFloat32Map, MyMap => Float32Map, Float32Map:int => float32, Float32Map.Rm:string => bytes.Buffer"

  This example uses 4 directives:

  - NewMyMap => NewFloat32Map
	Rename the function NewMyMap to NewFloat32Map
  - MyMap => Float32Map
	Rename the type MyMap to Float32Map
  - Float32Map:int => float32
	Within the type Float32Map replace all int by float32
  - Float32Map.Rm:string => bytes.Buffer
	Within the method Float32Map.Rm replace all string by bytes.Buffer
```

# example

From this repo examples folder

```sh
$ cat _examples/mymap/list.go
package mymap

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

$ jenjen -template=github.com/clementauger/jenjen/_examples/mymap - \
  "NewMyMap => NewFloat32Map, MyMap => Float32Map, Float32Map:int => float32, Float32Map.Rm:string => bytes.Buffer"
// source        /home/clementauger/gow/src/github.com/clementauger/jenjen/_examples/mymap/list.go
// destination   /home/clementauger/gow/src/github.com/clementauger/jenjen/jenjen_mymap.go
package main

import "bytes"

var somethingelse int

type Float32Map map[string]float32

func NewFloat32Map() Float32Map {
	return Float32Map{}
}

func (l *Float32Map) Set(k string, v float32) {
	(*l)[k] = v
}
func (l *Float32Map) Rm(k bytes.Buffer) {
	delete(*l, k)
}

//etc...
```

Using genny template

```sh
$ jenjen -template=github.com/cheekybits/genny/examples/go-generate -skip="gen-*" -\
  "KeyType => -, ValueType => -, KeyType => string, ValueType => []byte, KeyTypeValueTypeMap => StrBytesMap, NewKeyTypeValueTypeMap => NewStrBytesMap"
// source        /home/clementauger/gow/src/github.com/cheekybits/genny/examples/go-generate/go-generate.go
// destination   /home/clementauger/gow/src/github.com/clementauger/jenjen/jenjen_gogenerate.go
package main

//go:generate genny -in=$GOFILE -out=gen-$GOFILE gen "KeyType=string,int ValueType=string,int"

type StrBytesMap map[string][]byte

func NewStrBytesMap() map[string][]byte {
	return make(map[string][]byte)
}
```

## todo

- add versionning support `package/path@version`
- add tag support `-tag=onlythis,+addthis,-rmthis`
- support search of complex types like `map[K]V`
- write tests.
- maybe write it as a lib.
