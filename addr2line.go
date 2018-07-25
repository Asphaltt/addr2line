package addr2line

import (
	"debug/dwarf"
	"debug/elf"
	"io"
	"sort"

	"github.com/ianlancetaylor/demangle"
)

//Addr2LineEntry represents debug information for a address in the debug symbol.
type Addr2LineEntry struct {
	//Address in an executable or an offset in a section of a relocatable object
	Address uint

	//SoPath is the name of the library for which addresses should be translated
	SoPath string

	//Func is the name of function at Addr2LineEntry.Address in Addr2LineEntry.SoPath
	Func string

	//File is the name of the soure file
	//in which the Func is located in Addr2LineEntry.Address of Addr2LineEntry.SoPath library.
	File string

	//Line is the number of line in Addr2LineEntry.File at Address
	Line uint
}

type addr2LineSymbols struct {
	dwarfData  *dwarf.Data
	sortedKeys []int
	symbols    map[int]elf.Symbol
}

func openElf(r io.ReaderAt) (*elf.File, error) {
	f, err := elf.NewFile(r)
	if err != nil {
		return nil, err
	}
	return f, nil
}

var globalAddr2LineMapCaching = make(map[string]*addr2LineSymbols)

func makeAddr2LineMap(soPath string) (*addr2LineSymbols, error) {
	_, ok := globalAddr2LineMapCaching[soPath]

	if ok == false {
		fp, err := elf.Open(soPath)
		if err != nil {
			return nil, err
		}
		defer fp.Close()

		syms, err := fp.Symbols()
		if err != nil {
			return nil, err
		}

		dwarf, err := fp.DWARF()
		if err != nil {
			return nil, err
		}

		symbolsMap := make(map[int]elf.Symbol)
		var sortedAddrKeys []int

		for _, v := range syms {
			if v.Info == 0x12 && v.Value > 0 {
				symbolsMap[int(v.Value)] = v
				sortedAddrKeys = append(sortedAddrKeys, int(v.Value))
			}
		}

		sort.Ints(sortedAddrKeys)

		globalAddr2LineMapCaching[soPath] = &addr2LineSymbols{
			dwarfData:  dwarf,
			sortedKeys: sortedAddrKeys,
			symbols:    symbolsMap,
		}
	}

	return globalAddr2LineMapCaching[soPath], nil
}

//GetAddr2LineEntry function A returns a structure Addr2LineEntry with a function name
//and a file name line number at address in so file with a debug symbol.
func GetAddr2LineEntry(soPath string, address uint, doDemangle bool) (*Addr2LineEntry, error) {
	lineSymbols, err := makeAddr2LineMap(soPath)

	if err != nil || lineSymbols == nil {
		return nil, err
	}

	//Search uses binary search to find and return the smallest index i in [0, n)
	fIdx := sort.Search(len(lineSymbols.sortedKeys), func(idx int) bool {
		if lineSymbols.sortedKeys[idx] > int(address) {
			return true
		}
		return false
	}) - 1

	fAddress := lineSymbols.sortedKeys[fIdx]
	var functionName string
	if doDemangle == true {
		functionName, err = demangle.ToString(lineSymbols.symbols[fAddress].Name)
		if err == demangle.ErrNotMangledName {
			functionName = lineSymbols.symbols[fAddress].Name
		}
	} else {
		functionName = lineSymbols.symbols[fAddress].Name
	}

	retEntry := &Addr2LineEntry{}
	var line dwarf.LineEntry
	r := lineSymbols.dwarfData.Reader()

	e, err := r.SeekPC(uint64(address))
	if err != nil {
		return nil, err
	}

	lr, err := lineSymbols.dwarfData.LineReader(e)
	if err != nil {
		return nil, err
	}

	err = lr.SeekPC(uint64(address), &line)
	if err != nil {
		return nil, err
	}

	retEntry.SoPath = soPath
	retEntry.Address = address
	retEntry.Func = functionName
	retEntry.File = line.File.Name
	retEntry.Line = uint(line.Line)

	return retEntry, nil
}
