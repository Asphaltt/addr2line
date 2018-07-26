package addr2line

import (
	"fmt"
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

	Inline bool
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

var globalAddr2LineMapCaching         = make(map[string]*addr2LineSymbols)
var globalErrorDebugSymbolFilesCaching = make(map[string]bool)

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
/*
		for _, s := range syms {
			fmt.Printf("%#v\n", s)
		}
*/
		dwarf, err := fp.DWARF()
		if err != nil {
			return nil, err
		}

		symbolsMap := make(map[int]elf.Symbol)
		var sortedAddrKeys []int
		for _, v := range syms {
			if v.Info != 0 && len(v.Name) > 0 && v.Value > 0 {
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
	if _, ok := globalErrorDebugSymbolFilesCaching[soPath]; ok == true {
		return nil, fmt.Errorf("skip because of no symbol section (%s)", soPath)
	}

	lineSymbols, err := makeAddr2LineMap(soPath)

	if err != nil || lineSymbols == nil {
		globalErrorDebugSymbolFilesCaching[soPath] = true
		return nil, fmt.Errorf("%v (%s)", err, soPath)
	}

	//Search uses binary search to find and return the smallest index i in [0, n)
	fIdx := sort.Search(len(lineSymbols.sortedKeys), func(idx int) bool {
		if lineSymbols.sortedKeys[idx] > int(address) {
			return true
		}
		return false
	}) - 1

	if fIdx < 0 {
		//fmt.Println(lineSymbols, soPath, address, fIdx)
		return nil, fmt.Errorf("index is invalid\n")
	}

	fAddress := lineSymbols.sortedKeys[fIdx]
	var functionName string
	if doDemangle == true {
		fName := lineSymbols.symbols[fAddress].Name
		if len(fName) > 7 && fName[:7] == "__dl__Z" {
			fName = fName[5:]
		}
		functionName, err = demangle.ToString(fName)
		if err == demangle.ErrNotMangledName {
			functionName = lineSymbols.symbols[fAddress].Name
		}
	} else {
		functionName = lineSymbols.symbols[fAddress].Name
	}

	retEntry := &Addr2LineEntry{}
	var line dwarf.LineEntry
	var inline bool = false
	r := lineSymbols.dwarfData.Reader()
	e, err := r.SeekPC(uint64(address))
	if err != nil {
		return nil, err
	}
	/*
	rg2, err := lineSymbols.dwarfData.Ranges(e)
	fmt.Printf("range : %#v\n", rg2)
	fmt.Printf("start : %#v\n", e)
	*/
	for {
		cu, err := r.Next()
		if err != nil {
			continue
		}

		if cu == nil {
			break
		}

		if cu.Tag == dwarf.TagCompileUnit {
			break
		}
		rg, err := lineSymbols.dwarfData.Ranges(cu)

		/*
		fmt.Printf("%#v\n", rg)
		fmt.Printf("%#v\n\n", cu)
		*/

		if err != nil {
			continue
		}
		if len(rg) == 1 {
			//fmt.Printf("%#v\n", rg)
			//fmt.Printf("%#v\n\n", cu)
			if rg[0][0] <= uint64(address) && rg[0][1] > uint64(address) && cu.Tag == dwarf.TagInlinedSubroutine {
				//fmt.Println("######### inline #########")
				inline = true
				break
			}
		}
	}

	//fmt.Println("----------------------- end ---------------------")

	//e, err := r.SeekPC(uint64(address))
	//if err != nil {
	//	return nil, err
	//}

	lr, err := lineSymbols.dwarfData.LineReader(e)
	if err != nil {
		return nil, err
	}
/*
	lpos := lr.Tell()
	for {
		err := lr.Next(&line)
		if err == io.EOF {
			break
		}
		fmt.Printf("-- %#v\n", line.File)
		fmt.Printf("-- %#v\n", line)
	}
	lr.Seek(lpos)
*/
	if inline == true {
		err := lr.SeekPC(uint64(address), &line)
		if err != nil {
			return nil, err
		}
		//inline의 경우 prev entry의 line number를 먼저 확인하고..
		err = lr.SeekPC(uint64(line.Address-1), &line)
		if err != nil {
                        return nil, err
                }
		//prev entry의 line number가 0이면 다음 entry의 line number를...
		if line.Line == 0 {
			err := lr.SeekPC(uint64(address), &line)
			if err != nil {
				return nil, err
			}
			var line2 dwarf.LineEntry
			lr.Next(&line2)
			if line2.Line != 0 {
				line = line2
			}
		}
	}else{
		err = lr.SeekPC(uint64(address), &line)
		if err != nil {
			return nil, err
		}
	}
	//fmt.Printf("%#v\n", line)
	retEntry.SoPath = soPath
	retEntry.Address = address
	retEntry.Func = functionName
	retEntry.File = line.File.Name
	retEntry.Line = uint(line.Line)
	retEntry.Inline = inline
	return retEntry, nil
}
