package addr2line

import (
	"debug/dwarf"
	"debug/elf"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/ianlancetaylor/demangle"
)

// Addr2LineEntry represents debug information for a address in the debug symbol.
type Addr2LineEntry struct {
	// Address in an executable or an offset in a section of a relocatable object
	Address uint64

	// SoPath is the name of the library for which addresses should be translated
	SoPath string

	// Func is the name of function at Addr2LineEntry.Address in
	// Addr2LineEntry.SoPath
	Func string

	// File is the name of the source file in which the Func is located in
	// Addr2LineEntry.Address of Addr2LineEntry.SoPath library.
	File string

	// Line is the number of line in Addr2LineEntry.File at Address
	Line uint

	// Inline is the flag that indicates whether the function is inlined or not.
	Inline bool
}

// Addr2Line is a struct that contains the debug information of an ELF file.
type Addr2Line struct {
	soPath  string
	dwarf   *dwarf.Data
	addrs   []uint64 // sorted addresses
	symbols map[uint64]elf.Symbol
}

// New parses the ELF file at soPath, extracts the debug information and
// symbols. Then returns an Addr2Line struct.
func New(soPath string) (*Addr2Line, error) {
	fp, err := os.Open(soPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", soPath, err)
	}
	defer fp.Close()

	return NewAt(fp, soPath)
}

// NewAt parses the ELF file from the reader r, extracts the debug information
// and symbols. Then returns an Addr2Line struct.
//
// soPath is an optional parameter to provide more specific error messages.
func NewAt(r io.ReaderAt, soPath string) (*Addr2Line, error) {
	fp, err := elf.NewFile(r)
	if err != nil {
		return nil, fmt.Errorf("failed to open reader: %w", err)
	}
	defer fp.Close()

	syms, err := fp.Symbols()
	if err != nil {
		return nil, fmt.Errorf("failed to get symbols from %s: %w", soPath, err)
	}

	dwarf, err := fp.DWARF()
	if err != nil {
		return nil, fmt.Errorf("failed to get DWARF data from %s: %w", soPath, err)
	}

	var addrs []uint64
	symbols := make(map[uint64]elf.Symbol)

	for _, v := range syms {
		if v.Info != 0 && len(v.Name) > 0 && v.Value > 0 {
			symbols[v.Value] = v
			addrs = append(addrs, v.Value)
		}
	}

	slices.Sort(addrs)

	var a2l Addr2Line
	a2l.soPath = soPath
	a2l.dwarf = dwarf
	a2l.addrs = addrs
	a2l.symbols = symbols

	return &a2l, nil
}

func (a2l *Addr2Line) FindBySymbol(symbol string) (*Addr2LineEntry, error) {
	for _, entry := range a2l.symbols {
		if entry.Name == symbol {
			return a2l.Get(entry.Value, false)
		}
	}

	return nil, fmt.Errorf("%s not found", symbol)
}

// Get returns the Addr2LineEntry for the given address.
func (a2l *Addr2Line) Get(address uint64, doDemangle bool) (*Addr2LineEntry, error) {
	// Search uses binary search to find and return the smallest index i in [0, n)
	idx := sort.Search(len(a2l.addrs), func(idx int) bool {
		if a2l.addrs[idx] > address {
			return true
		}
		return false
	}) - 1

	if idx < 0 {
		return nil, fmt.Errorf("index is invalid\n")
	}

	addr := a2l.addrs[idx]
	var functionName string
	var err error

	if doDemangle {
		name := a2l.symbols[addr].Name
		if strings.HasPrefix(name, "__dl__Z") {
			name = name[5:]
		}
		functionName, err = demangle.ToString(name)
		if errors.Is(err, demangle.ErrNotMangledName) {
			functionName = a2l.symbols[addr].Name
		}
	} else {
		functionName = a2l.symbols[addr].Name
	}

	var line dwarf.LineEntry
	var isInline bool

	dwarfEntry := a2l.dwarf.Reader()
	e, err := dwarfEntry.SeekPC(address)
	if err != nil {
		return nil, err
	}

	for {
		entry, err := dwarfEntry.Next()
		if err != nil {
			continue
		}

		if entry == nil || entry.Tag == dwarf.TagCompileUnit {
			break
		}

		rng, err := a2l.dwarf.Ranges(entry)
		if err != nil {
			continue
		}

		if len(rng) == 1 {
			if rng[0][0] <= address && rng[0][1] > address && entry.Tag == dwarf.TagInlinedSubroutine {
				isInline = true
				break
			}
		}
	}

	lr, err := a2l.dwarf.LineReader(e)
	if err != nil {
		return nil, err
	}

	if isInline {
		err := lr.SeekPC(address, &line)
		if err != nil {
			return nil, err
		}

		// in the case of inline, first check the line number of the previous entry..
		err = lr.SeekPC(uint64(line.Address-1), &line)
		if err != nil {
			return nil, err
		}

		// If the line number of the previous entry is 0, then the line number of the next entry...
		if line.Line == 0 {
			err := lr.SeekPC(address, &line)
			if err != nil {
				return nil, err
			}

			var line2 dwarf.LineEntry
			lr.Next(&line2)
			if line2.Line != 0 {
				line = line2
			}
		}

	} else {
		err = lr.SeekPC(address, &line)
		if err != nil {
			return nil, err
		}
	}

	var entry Addr2LineEntry
	entry.SoPath = a2l.soPath
	entry.Address = address
	entry.Func = functionName
	entry.File = line.File.Name
	entry.Line = uint(line.Line)
	entry.Inline = isInline
	return &entry, nil
}
