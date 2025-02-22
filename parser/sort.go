package parser

import "strconv"

type ByNmSize struct {
	NmParsers []*NmParser
	IsAsc     bool
}

func (a ByNmSize) Len() int      { return len(a.NmParsers) }
func (a ByNmSize) Swap(i, j int) { a.NmParsers[i], a.NmParsers[j] = a.NmParsers[j], a.NmParsers[i] }
func (a ByNmSize) Less(i, j int) bool {
	if a.IsAsc {
		return a.NmParsers[i].Size < a.NmParsers[j].Size
	}
	return a.NmParsers[i].Size > a.NmParsers[j].Size
}

type ByNmSymbol struct {
	NmParsers []*NmParser
	IsAsc     bool
}

func (a ByNmSymbol) Len() int      { return len(a.NmParsers) }
func (a ByNmSymbol) Swap(i, j int) { a.NmParsers[i], a.NmParsers[j] = a.NmParsers[j], a.NmParsers[i] }
func (a ByNmSymbol) Less(i, j int) bool {
	if a.IsAsc {
		return a.NmParsers[i].Symbol < a.NmParsers[j].Symbol
	}
	return a.NmParsers[i].Symbol > a.NmParsers[j].Symbol
}

type ByNmAddress struct {
	NmParsers []*NmParser
	IsAsc     bool
}

func (a ByNmAddress) Len() int      { return len(a.NmParsers) }
func (a ByNmAddress) Swap(i, j int) { a.NmParsers[i], a.NmParsers[j] = a.NmParsers[j], a.NmParsers[i] }
func (a ByNmAddress) Less(i, j int) bool {
	ai, _ := strconv.ParseInt(a.NmParsers[i].Address, 16, 64)
	aj, _ := strconv.ParseInt(a.NmParsers[j].Address, 16, 64)
	if a.IsAsc {
		return ai < aj
	}
	return ai > aj
}

// --------------------------------------------------------------------------------

type ByPkgSize struct {
	PkgInfos []*PkgInfo
	IsAsc    bool
}

func (a ByPkgSize) Len() int      { return len(a.PkgInfos) }
func (a ByPkgSize) Swap(i, j int) { a.PkgInfos[i], a.PkgInfos[j] = a.PkgInfos[j], a.PkgInfos[i] }
func (a ByPkgSize) Less(i, j int) bool {
	if a.IsAsc {
		return a.PkgInfos[i].Size < a.PkgInfos[j].Size
	}
	return a.PkgInfos[i].Size > a.PkgInfos[j].Size
}

type ByPkgName struct {
	PkgInfos []*PkgInfo
	IsAsc    bool
}

func (a ByPkgName) Len() int      { return len(a.PkgInfos) }
func (a ByPkgName) Swap(i, j int) { a.PkgInfos[i], a.PkgInfos[j] = a.PkgInfos[j], a.PkgInfos[i] }
func (a ByPkgName) Less(i, j int) bool {
	if a.IsAsc {
		return a.PkgInfos[i].PkgName < a.PkgInfos[j].PkgName
	}
	return a.PkgInfos[i].PkgName > a.PkgInfos[j].PkgName
}

type ByPkgLines struct {
	PkgInfos []*PkgInfo
	IsAsc    bool
}

func (a ByPkgLines) Len() int      { return len(a.PkgInfos) }
func (a ByPkgLines) Swap(i, j int) { a.PkgInfos[i], a.PkgInfos[j] = a.PkgInfos[j], a.PkgInfos[i] }
func (a ByPkgLines) Less(i, j int) bool {
	if a.IsAsc {
		return a.PkgInfos[i].Lines > a.PkgInfos[j].Lines
	}
	return a.PkgInfos[i].Lines < a.PkgInfos[j].Lines
}
