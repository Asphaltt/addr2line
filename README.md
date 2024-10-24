# Addr2line Package in Go

This is a Golang package that provides functionality equivalent to **GNU addr2line**.

## Prerequisites

To support the demangling feature, the `addr2line` package relies on the [ianlancetaylor/demangle](https://github.com/ianlancetaylor/demangle) package. Be sure to install this package before using `addr2line`.

Install the `demangle` package using:

```bash
go get -u github.com/ianlancetaylor/demangle
```

## Usage

The `addr2line` package is simple to use.

Install the `addr2line` package using:

```bash
go get -u github.com/Asphaltt/addr2line
```

Example usage:

```go
import "github.com/Asphaltt/addr2line"

a2l, err := addr2line.New("/path/to/so/file/with/debug/symbols")
if err != nil {
    fmt.Println(err)
    return
}

e, err := a2l.Get(address, true)
if err != nil {
    fmt.Println(err)
} else {
    //t.Logf("%v\n", e)
    fmt.Printf("Library  : %s\n", e.SoPath)
    fmt.Printf("Address  : 0x%xu\n", e.Address)
    fmt.Printf("Function : %s\n", e.Func)
    fmt.Printf("File     : %s\n", e.File)
    fmt.Printf("Line     : %d\n", e.Line)
}
```

For more detailed examples, refer to `addr2line_test.go`.
