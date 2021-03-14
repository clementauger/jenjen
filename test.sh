set -x
set -e

go install ./...

function fail() { echo "test failed"; exit 1; }

function test() {
  jenjen -template="$1" - "$2" | grep -e "$3" || fail;
}

TPL="github.com/clementauger/jenjen/_examples/mymap"

test $TPL "int => string" "package jenjen" # package name is the directory base when the cwd is not a go package.
test $TPL "int => string" "// jenjen 0\.0\.0-dev"

test $TPL "int => string" "Rm(k string)"
test $TPL "int => string" "Set(k string, v string)"
test $TPL "int => string" "type MyMap map\[string\]string"
test $TPL "int => string" "var somethingelse string"

test $TPL "MyMap => TestStruct" "type TestStruct map\[string\]int"
test $TPL "MyMap => TestStruct" "var somethingelse int"
test $TPL "MyMap => TestStruct" "NewMyMap() TestStruct"
test $TPL "MyMap => TestStruct" "func (l \*TestStruct) Set("
test $TPL "MyMap => TestStruct" "func (l \*TestStruct) Rm("

test $TPL "MyMap:int => string" "var somethingelse int"
test $TPL "MyMap:int => string" "type MyMap map\[string\]string"

test $TPL "MyMap.Rm:int => string" "type MyMap map\[string\]int"
test $TPL "MyMap.Rm:int => string" "Set(k string, v int)"
test $TPL "MyMap.Rm:int => string" "Rm(k string)"

test $TPL "int => []byte" "Rm(k string)"
test $TPL "int => []byte" "Set(k string, v \[\]byte)"
test $TPL "int => []byte" "type MyMap map\[string\]\[\]byte"
test $TPL "int => []byte" "var somethingelse \[\]byte"

test $TPL "int => bytes.Buffer" "import \"bytes\""
test $TPL "int => bytes.Buffer" "Rm(k string)"
test $TPL "int => bytes.Buffer" "Set(k string, v bytes\.Buffer)"
test $TPL "int => bytes.Buffer" "type MyMap map\[string\]bytes\.Buffer"
test $TPL "int => bytes.Buffer" "var somethingelse bytes\.Buffer"
