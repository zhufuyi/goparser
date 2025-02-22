package parser

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

type BinaryParser struct {
	NmParsers []*NmParser
	PkgInfos  []*PkgInfo
	TotalSize int
	MaxWidth  int
}

func NewBinaryParser(file string, grep string) (*BinaryParser, error) {
	nmParsers, totalSize, err := GetNmParsers(file, grep)
	if err != nil {
		return nil, err
	}

	pkgInfos, subPkgNameMap, err := GetPkgInfos(file, grep)
	if err != nil {
		return nil, err
	}

	binaryParser := &BinaryParser{TotalSize: totalSize, NmParsers: nmParsers}
	for i, info := range pkgInfos {
		for _, parser := range nmParsers {
			if info.IsMod {
				if strings.HasPrefix(parser.Symbol, info.PkgName) || strings.Contains(parser.Symbol, "type:.eq."+info.PkgName) {
					pkgInfos[i].Size += parser.Size
					pkgInfos[i].Lines += 1
				}
			} else {
				if strings.Contains(parser.Symbol, info.PkgName) {
					pkgInfos[i].Size += parser.Size
					pkgInfos[i].Lines += 1
				}
			}
		}
		pkgInfos[i].SizePercentage = float32(info.Size) / float32(totalSize) * 100
	}
	deleteDuplicateSize(pkgInfos, subPkgNameMap)
	binaryParser.PkgInfos = pkgInfos

	return binaryParser, nil
}

func (bp *BinaryParser) PrintNmParser(binaryFile string, topN int) {
	nmMaxWidth := []int{bp.MaxWidth, 8, 4, 11, 15}
	for i := 0; i < len(nmMaxWidth); i++ {
		nmMaxWidth[i] += 4
	}

	totalLine := len(bp.NmParsers)
	n := topN
	if topN > totalLine {
		n = totalLine
	}

	title := fmt.Sprintf("%-*s%-*s%-*s%-*s%-*s",
		nmMaxWidth[0], "Symbol",
		nmMaxWidth[1], "Address",
		nmMaxWidth[2], "Type",
		nmMaxWidth[3], "Size(bytes)",
		nmMaxWidth[4], "Percentage(size)")
	resultTip := fmt.Sprintf("parse binary file \"%s\" resuls:", binaryFile)
	fmt.Printf("\n%s\ntotal size: %s bytes,  total rows: %s,  show top %s rows:\n",
		resultTip,
		color.HiGreenString(strconv.Itoa(bp.TotalSize)),
		color.HiCyanString(strconv.Itoa(totalLine)),
		color.HiMagentaString(strconv.Itoa(n)))
	separators := strings.Repeat("-", len(title)-4)
	fmt.Println(color.HiBlackString(separators))
	fmt.Println(color.HiCyanString(title))
	fmt.Println(color.HiBlackString(separators))
	nmParsers := bp.NmParsers
	if len(bp.NmParsers) > topN {
		nmParsers = bp.NmParsers[:topN]
	}
	for _, nm := range nmParsers {
		symbol := nm.Symbol
		if len(symbol) >= nmMaxWidth[0] {
			size := nmMaxWidth[0] - 29
			symbol = symbol[:20] + " ... " + symbol[len(symbol)-size:]
		}
		fmt.Printf("%-*s%-*s%-*s%-*s%-*s\n",
			nmMaxWidth[0], symbol,
			nmMaxWidth[1], nm.Address,
			nmMaxWidth[2], nm.Type,
			nmMaxWidth[3], strconv.Itoa(nm.Size),
			nmMaxWidth[4], fmt.Sprintf("%.3f%%", nm.SizePercentage))
	}
	if len(nmParsers) > 0 {
		fmt.Println(color.HiBlackString(separators))
	}
}

func (bp *BinaryParser) PrintPkgInfo(_ string, topN int) {
	piMaxWidth := []int{bp.MaxWidth, 11, 11, 15}
	for i := 0; i < len(piMaxWidth); i++ {
		piMaxWidth[i] += 4
	}

	totalLine := len(bp.PkgInfos)
	n := topN
	if topN > totalLine {
		n = totalLine
	}
	pkgInfos := bp.PkgInfos
	depSize := 0
	modSize := 0
	sumSize := 0
	for _, info := range pkgInfos {
		if info.IsMod {
			modSize += info.Size
		} else {
			depSize += info.Size
		}
	}
	sumSize = depSize + modSize

	title := fmt.Sprintf("%-*s%-*s%-*s%-*s",
		piMaxWidth[0], "Package",
		piMaxWidth[1], "Count Rows",
		piMaxWidth[2], "Size(bytes)",
		piMaxWidth[3], "Percentage(size)")

	resultTip := "parse go mod package results:"
	fmt.Printf("\n%s\nsum size: %s bytes, dep size: %s bytes, mod size: %s bytes, percentage(sum/total): %s,\ntotal rows: %s, show top %s rows:\n",
		resultTip,
		color.HiGreenString(strconv.Itoa(sumSize)),
		color.HiGreenString(strconv.Itoa(depSize)),
		color.HiGreenString(strconv.Itoa(modSize)),
		color.HiGreenString(fmt.Sprintf("%.2f%%", float32(sumSize)/float32(bp.TotalSize)*100)),
		color.HiCyanString(strconv.Itoa(totalLine)),
		color.HiMagentaString(strconv.Itoa(n)),
	)
	separators := strings.Repeat("-", len(title)-4)
	fmt.Println(color.HiBlackString(separators))
	fmt.Println(color.HiCyanString(title))
	fmt.Println(color.HiBlackString(separators))
	if len(bp.PkgInfos) > topN {
		pkgInfos = bp.PkgInfos[:topN]
	}
	for _, info := range pkgInfos {
		pkgName := info.PkgName
		if info.IsMod {
			pkgName = strings.TrimRight(pkgName, "/") + " (mod)"
		}
		if len(pkgName) > piMaxWidth[0] {
			size := piMaxWidth[0] - 29
			pkgName = pkgName[:20] + " ... " + pkgName[len(pkgName)-size:]
		}
		fmt.Printf("%-*s%-*s%-*s%-*s\n",
			piMaxWidth[0], pkgName,
			piMaxWidth[1], strconv.Itoa(info.Lines),
			piMaxWidth[2], strconv.Itoa(info.Size),
			piMaxWidth[3], fmt.Sprintf("%.2f%%", float32(info.Size)/float32(sumSize)*100),
		)
	}
	if len(pkgInfos) > 0 {
		fmt.Println(color.HiBlackString(separators))
	}
}

// ------------------------------------------------------------------------------------------

type NmParser struct {
	Symbol         string  `json:"symbol"`
	Address        string  `json:"address"`
	Type           string  `json:"type"`
	Size           int     `json:"size"`
	SizePercentage float32 `json:"sizePercentage"`
}

func GetNmParsers(file string, grep string) ([]*NmParser, int, error) {
	data, err := Exec("go", "tool", "nm", "-size", file)
	if err != nil {
		return nil, 0, err
	}

	var nmParsers []*NmParser
	var totalSize int
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		ss := strings.Fields(line)
		if len(ss) >= 4 {
			size, _ := strconv.Atoi(ss[1])
			totalSize += size
			if grep != "" && !strings.Contains(line, grep) {
				continue
			}
			nmParsers = append(nmParsers, &NmParser{
				Address: ss[0],
				Size:    size,
				Type:    ss[2],
				Symbol:  strings.Join(ss[3:], ""),
			})
		}
	}

	for i := 0; i < len(nmParsers); i++ {
		nmParsers[i].SizePercentage = float32(nmParsers[i].Size) / float32(totalSize) * 100
	}

	return nmParsers, totalSize, nil
}

// ------------------------------------------------------------------------------------------

type PkgInfo struct {
	PkgName        string  `json:"pkgName"`
	Lines          int     `json:"lines"` // match lines of nm lines
	Size           int     `json:"size"`
	SizePercentage float32 `json:"sizePercentage"`
	Version        string  `json:"version"`
	IsMod          bool    `json:"isMod"`
}

func GetPkgInfos(file string, grep string) ([]*PkgInfo, map[string][]string, error) {
	data, err := Exec("go", "version", "-m", file)
	if err != nil {
		return nil, nil, err
	}

	var pkgInfos []*PkgInfo
	var pkgNames []string
	subPkgNameMap := make(map[string][]string)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if grep != "" && !strings.Contains(line, grep) {
			continue
		}
		ss := strings.Fields(line)
		if len(ss) == 4 && ss[0] == "dep" {
			pkgInfos = append(pkgInfos, &PkgInfo{
				PkgName: ss[1],
				Version: ss[2],
			})
			findSubPkgNames(ss[1], pkgNames, subPkgNameMap)
			pkgNames = append(pkgNames, ss[1])
		} else if len(ss) > 2 && ss[0] == "mod" {
			pkgInfos = append(pkgInfos, &PkgInfo{
				PkgName: ss[1] + "/",
				IsMod:   true,
			})
		} else if len(ss) == 3 && ss[0] == "dep" && ss[2] == "(devel)" {
			pkgInfos = append(pkgInfos, &PkgInfo{
				PkgName: ss[1] + "/",
				IsMod:   true,
			})
		}
	}

	return pkgInfos, subPkgNameMap, nil
}

func findSubPkgNames(currentName string, pkgNames []string, subPkgNameMap map[string][]string) {
	for _, name := range pkgNames {
		if len(name) > len(currentName) {
			if strings.Contains(name, currentName) {
				if v, ok := subPkgNameMap[currentName]; ok {
					v = append(v, name)
					subPkgNameMap[currentName] = v
				} else {
					subPkgNameMap[currentName] = []string{name}
				}
			}
		} else if strings.Contains(currentName, name) {
			if v, ok := subPkgNameMap[name]; ok {
				v = append(v, currentName)
				subPkgNameMap[name] = v
			} else {
				subPkgNameMap[name] = []string{currentName}
			}
		}
	}
}

func deleteDuplicateSize(pkgInfos []*PkgInfo, subPkgNameMap map[string][]string) {
	for subName, names := range subPkgNameMap {
		subNameIndex := -1
		countSize := 0
		lines := 0
		for i, info := range pkgInfos {
			if subName == info.PkgName {
				subNameIndex = i
			}
			for _, name := range names {
				if name == info.PkgName {
					countSize += info.Size
					lines += info.Lines
				}
			}
		}
		if subNameIndex >= 0 {
			pkgInfos[subNameIndex].Size -= countSize
			pkgInfos[subNameIndex].Lines -= lines
		}
	}
}
