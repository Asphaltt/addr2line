package addr2line

import (
	"flag"
	"os"
	"strconv"
	"testing"
)

var globalT *testing.T

func printUsage() {
	globalT.Error("usage: addr2line [so library path] [hexadecimal address]\n\tex) addr2line libc.so 0x493c\n")
}

func usage() {
	printUsage()
}

func TestGetAddr2LineEntry(t *testing.T) {
	globalT = t

	if len(os.Args) < 2 || os.Args[1] == "--help" {
		usage()
	}

	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 2 {
		usage()
		return
	}

	addressStr := flag.Arg(1)

	if len(addressStr) > 2 && addressStr[:2] == "0x" {
		addressStr = addressStr[2:]
	}

	addr, err := strconv.ParseInt(addressStr, 16, 64)
	if err != nil {
		t.Errorf("%v\n", err)
	}

	a2l, err := New(flag.Arg(0))
	if err != nil {
		t.Error(err)
		return
	}

	e, err := a2l.Get(uint64(addr), true)
	if err != nil {
		t.Error(err)
	} else {
		// t.Logf("%v\n", e)
		t.Logf("library  : %s\n", e.SoPath)
		t.Logf("address  : 0x%x\n", e.Address)
		t.Logf("function : %s\n", e.Func)
		t.Logf("file     : %s\n", e.File)
		t.Logf("line     : %d\n", e.Line)
		t.Logf("inline   : %v\n", e.Inline)
	}
}

func BenchmarkGetAddr2LineEntry(t *testing.B) {
	if len(os.Args) < 2 || os.Args[1] == "--help" {
		usage()
	}

	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 2 {
		usage()
		return
	}

	addressStr := flag.Arg(1)

	if len(addressStr) > 2 && addressStr[:2] == "0x" {
		addressStr = addressStr[2:]
	}

	addr, err := strconv.ParseInt(addressStr, 16, 64)
	if err != nil {
		t.Errorf("%v\n", err)
	}

	a2l, err := New(flag.Arg(0))
	if err != nil {
		t.Error(err)
		return
	}

	e, err := a2l.Get(uint64(addr), true)
	if err != nil {
		t.Error(err)
	} else {
		// t.Logf("%v\n", e)
		t.Logf("library  : %s\n", e.SoPath)
		t.Logf("address  : 0x%x\n", e.Address)
		t.Logf("function : %s\n", e.Func)
		t.Logf("file     : %s\n", e.File)
		t.Logf("line     : %d\n", e.Line)
		t.Logf("inline   : %v\n", e.Inline)
	}
}
