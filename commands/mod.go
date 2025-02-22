package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const goPkgDevURL = "https://pkg.go.dev/"

var cacheGoLibDir = getCacheGoLibDir()

func parseGoModCMD() *cobra.Command {
	var (
		modFile          string // go.mod文件路径
		refreshInterval  int    // 多少小时间隔更新一次缓存，单位:小时
		isForceUpdate    bool   // 是否强制更新缓存，refreshInterval参数失效
		requestFrequency int    // 请求频率限制，单位:毫秒
	)

	cmd := &cobra.Command{
		Use:   "mod",
		Short: "Parse go.mod and check the package version is up-to-date with the latest release",
		Long:  "Parse go.mod and check the package version is up-to-date with the latest release.",
		Example: color.HiBlackString(`  # Show package version information
  goparser mod --mod-file=./go.mod`),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := parserGoMod(modFile, refreshInterval, isForceUpdate, requestFrequency)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().StringVarP(&modFile, "mod-file", "f", "./go.mod", "go mod file path")
	_ = cmd.MarkFlagRequired("mod-file")
	cmd.Flags().IntVarP(&refreshInterval, "refresh-interval", "r", 24*15, "refresh cache interval, unit: hours")
	cmd.Flags().BoolVarP(&isForceUpdate, "is-force-update", "u", false, "whether to force update cache, refresh-interval parameter invalid")
	cmd.Flags().IntVarP(&requestFrequency, "request-frequency", "l", 1100, "request frequency limit, unit: milliseconds")

	return cmd
}

// -----------------------------------------------------------------------------

type GoLib struct {
	URL                string    `json:"url"`
	CurrentVersion     string    `json:"currentVersion"`
	CurrentVersionDate string    `json:"currentVersionDate"`
	LatestVersion      string    `json:"latestVersion"`
	LatestVersionDate  string    `json:"latestVersionDate"`
	VersionInterval    int       `json:"versionInterval"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type ByVersionInterval struct {
	GoLibs []*GoLib
	IsAsc  bool
}

func (a ByVersionInterval) Len() int      { return len(a.GoLibs) }
func (a ByVersionInterval) Swap(i, j int) { a.GoLibs[i], a.GoLibs[j] = a.GoLibs[j], a.GoLibs[i] }
func (a ByVersionInterval) Less(i, j int) bool {
	if a.IsAsc {
		return a.GoLibs[i].VersionInterval < a.GoLibs[j].VersionInterval
	}
	return a.GoLibs[i].VersionInterval > a.GoLibs[j].VersionInterval
}

func newVersionInterval(goLibsMap map[string]*GoLib, goModDirectLibs []*GoLib) *ByVersionInterval {
	goLibs := make([]*GoLib, 0, len(goLibsMap))

	for _, goModDirectLib := range goModDirectLibs {
		if v, ok := goLibsMap[goModDirectLib.URL]; ok {
			goLibs = append(goLibs, v)
		}
	}

	return &ByVersionInterval{
		GoLibs: goLibs,
	}
}

func parserGoMod(modFile string, refreshInterval int, isForceUpdate bool, requestFrequency int) (string, error) {
	// 从本地缓存获取依赖库信息
	goLibMapCache, _, err := getGoLibsFromCache()
	if err != nil {
		return "", err
	}

	// 从go.mod文件中解析依赖库信息
	data, err := os.ReadFile(modFile)
	if err != nil {
		return "", err
	}
	goModDirectLibs, err := getGoModDirectLibs(data)
	if err != nil {
		return "", err
	}

	isNeedUpdate := false
	for _, directLib := range goModDirectLibs {
		if isNeedRefresh(goLibMapCache, directLib.URL, refreshInterval) || isForceUpdate {
			p := NewWaitPrinter(time.Millisecond * 200)
			p.LoopPrint(fmt.Sprintf("parsing %s ", color.HiGreenString(directLib.URL)))
			time.Sleep(time.Millisecond * time.Duration(requestFrequency)) // 防止请求过快导致被ban
			latestVersion, latestVersionDate, err := getVersionFromGoPkgDev(directLib.URL)
			if err != nil {
				p.StopPrint(fmt.Sprintf("parse library %s version error: %v\n", directLib.URL, err))
				continue
			}
			p.StopPrint("")

			goLibMapCache[directLib.URL] = &GoLib{
				URL:                directLib.URL,
				CurrentVersion:     directLib.CurrentVersion,
				CurrentVersionDate: directLib.CurrentVersionDate,
				LatestVersion:      latestVersion,
				LatestVersionDate:  latestVersionDate,
				VersionInterval:    calculateInterval(latestVersion, directLib.CurrentVersion),
				UpdatedAt:          time.Now(),
			}
			isNeedUpdate = true
		}
	}

	if isNeedUpdate {
		err = setGoLibsToCache(goLibMapCache)
		if err != nil {
			return "", err
		}
	}

	// 按版本间隔排序
	byVersionInterval := newVersionInterval(goLibMapCache, goModDirectLibs)
	sort.Sort(byVersionInterval)

	title := fmt.Sprintf("%-50s %-25s %-25s %s\n", "Package", "Used Version", "Latest Version", "Interval")
	separators := strings.Repeat("-", len(title)) + "\n"
	result := color.HiBlackString(separators) + color.HiCyanString(title) + color.HiBlackString(separators)
	for _, lib := range byVersionInterval.GoLibs {
		currentVersion := lib.CurrentVersion
		if currentVersion == "v0.0.0" {
			currentVersion += " (" + lib.CurrentVersionDate + ")"
		}
		latestVersion := lib.LatestVersion
		latestVersion += " (" + lib.LatestVersionDate + ")"
		libURL := lib.URL
		if len(libURL) > 50 {
			libURL = lib.URL[:20] + " ... " + lib.URL[len(lib.URL)-20:]
		}
		result += fmt.Sprintf("%-50s %-25s %-25s %d\n", libURL, currentVersion, latestVersion, lib.VersionInterval)
	}
	result += color.HiBlackString(separators)

	return result, nil
}

// --------------------------------------------------------------------------------

func getHtml(url string) ([]byte, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %v", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP response status code is not 200, it is %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body failed: %v", err)
	}

	return body, nil
}

func parseHtml(html []byte) (string, string, error) {
	// parse html
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return "", "", fmt.Errorf("NewDocumentFromReader error: %v", err)
	}

	// get version
	version := doc.Find("a[aria-label^='Version:']").Text()
	version = trimStr(version)

	// get published date
	publishedText := doc.Find("span[data-test-id='UnitHeader-commitTime']").Text()
	publishedDate := strings.TrimPrefix(publishedText, "Published: ")
	publishedDate = trimStr(publishedDate)
	date, err := time.Parse("Jan 2, 2006", publishedDate)
	if err == nil {
		publishedDate = date.Format("2006-01-02")
	}

	return version, publishedDate, nil
}

func getVersionFromGoPkgDev(lib string) (string, string, error) {
	html, err := getHtml(goPkgDevURL + lib)
	if err != nil {
		return "", "", err
	}

	version, publishedDate, err := parseHtml(html)
	if err != nil {
		return "", "", err
	}
	return version, publishedDate, nil
}

func trimStr(s string) string {
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\t", "")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "Version:", "")
	s = strings.ReplaceAll(s, "Published:", "")
	s = strings.TrimSpace(s)
	ss := strings.Split(s, "+")
	ss = strings.Split(ss[0], "-")
	return ss[0]
}

// -------------------------------------------------------------------------------

func getGoLibsFromCache() (map[string]*GoLib, []*GoLib, error) {
	goLibMap := make(map[string]*GoLib)

	goLibFile := fmt.Sprintf("%s/go.lib.json", cacheGoLibDir)
	if !isExists(goLibFile) {
		err := os.MkdirAll(cacheGoLibDir, 0755)
		if err != nil {
			return nil, nil, err
		}
		err = os.WriteFile(goLibFile, []byte("{}"), 0644)
		if err != nil {
			return nil, nil, err
		}
		return goLibMap, []*GoLib{}, nil
	}

	data, err := os.ReadFile(goLibFile)
	if err != nil {
		return nil, nil, fmt.Errorf("red go.lib.json failed: %v", err)
	}

	err = json.Unmarshal(data, &goLibMap)
	if err != nil {
		return nil, nil, fmt.Errorf("unmarshal go.lib.json failed: %v", err)
	}

	goLibs := make([]*GoLib, 0, len(goLibMap))
	for _, goLib := range goLibMap {
		goLibs = append(goLibs, goLib)
	}

	return goLibMap, goLibs, nil
}

func setGoLibsToCache(goLibMap map[string]*GoLib) error {
	goLibFile := fmt.Sprintf("%s/go.lib.json", cacheGoLibDir)
	data, err := json.Marshal(goLibMap)
	if err != nil {
		return fmt.Errorf("marshal go.lib.json failed: %v", err)
	}

	err = os.WriteFile(goLibFile, data, 0644)
	if err != nil {
		return fmt.Errorf("write go.lib.json failed: %v", err)
	}

	return nil
}

func isNeedRefresh(goLibMap map[string]*GoLib, url string, refreshInterval int) bool {
	lib, ok := goLibMap[url]
	if !ok {
		return true
	}

	if int(time.Now().Sub(lib.UpdatedAt).Hours()) > refreshInterval {
		return true
	}

	if lib.LatestVersion == "v0.0.0" {
		latestVersionTime, err := time.Parse("2006-01-02", lib.LatestVersionDate)
		if err != nil {
			return true
		}
		if int(time.Now().Sub(latestVersionTime).Hours()) > refreshInterval {
			return true
		}
	}

	return false
}

func calculateInterval(latestVersion string, currentVersion string) int {
	latestVersionSize := convertVersionToInt(latestVersion)
	currentVersionSize := convertVersionToInt(currentVersion)
	return latestVersionSize - currentVersionSize
}

func convertVersionToInt(version string) int {
	ss := strings.Split(version, "-")
	currentVersion := ss[0]
	currentVersion = strings.TrimPrefix(currentVersion, "v")
	ss = strings.Split(currentVersion, ".")
	if len(ss) < 3 {
		return 0
	}

	return atoi(ss[0])*10000 + atoi(ss[1])*100 + atoi(ss[2])
}

func atoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

func isExists(f string) bool {
	_, err := os.Stat(f)
	if err != nil {
		return !os.IsNotExist(err)
	}
	return true
}

func getCacheGoLibDir() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s/.goparser", dir)
}

// ---------------------------------------------------

func getGoModDirectLibs(data []byte) ([]*GoLib, error) {
	var goLibs []*GoLib

	// regular expression matching direct dependency
	re := regexp.MustCompile(`([^\s]+)\s+v([\d\.\-\w]+)(.*)`)
	matches := re.FindAllStringSubmatch(string(data), -1)
	if matches == nil {
		return goLibs, nil
	}
	for _, match := range matches {
		if len(match) >= 4 {
			if strings.Contains(match[0], "// indirect") {
				continue
			}
			libURL := strings.TrimSpace(match[1])
			if !strings.Contains(libURL, "/") {
				continue
			}
			version, date := handleVersion("v" + strings.TrimSpace(match[2]))
			goLibs = append(goLibs, &GoLib{
				URL:                libURL,
				CurrentVersion:     version,
				CurrentVersionDate: date,
			})
		}
	}

	return goLibs, nil
}

func handleVersion(version string) (string, string) {
	if strings.Contains(version, "v0.0.0") {
		ss := strings.Split(version, "-")
		if len(ss) != 3 {
			return version, ""
		}
		date, err := time.Parse("20060102150405", ss[1])
		if err != nil {
			return ss[0], ""
		}
		return ss[0], date.Format("2006-01-02")
	}
	ss := strings.Split(version, "+")
	return ss[0], ""
}
