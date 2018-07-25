# Addr2line package implemented in golang
GNU addr2line과 동일한 기능을 하는 golang package입니다.

## 필요사항
addr2line package는 demangle 기능을 지원하기 위해서 [ianlancetaylor/demangle package](github.com/ianlancetaylor/demangle)를 사용합니다.
[ianlancetaylor/demangle package](github.com/ianlancetaylor/demangle)를 먼저 설치해야 합니다.
```
go get -u github.com/ianlancetaylor/demangle
```

## 사용법
addr2line package는 쉽게 사용할 수 있습니다.
package를 설치한 뒤에 import하고 GetAddr2LineEntry 함수만 호출하면 됩니다.
```
go get -u github.com/daludaluking/addr2line
```
```
import "github.com/daludaluking/addr2line"

e, err := addr2line.GetAddr2LineEntry(soFilePathWithDebugSymbols, address, true)
if err != nil {
	fmt.Println(err)
} else {
	//t.Logf("%v\n", e)
	fmt.Printf("library  : %s\n", e.SoPath)
	fmt.Printf("address  : 0x%xu\n", e.Address)
	fmt.Printf("function : %s\n", e.Func)
	fmt.Printf("file     : %s\n", e.File)
	fmt.Printf("line     : %d\n", e.Line)
}
```
보다 자세한 예제는 addr2line_test.go를 참고하세요.
