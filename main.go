package main

import (
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/term"
)

// 文件类型枚举
type FileType int

const (
	FileTypeOther FileType = iota
	FileTypeDirectory
	FileTypeExecutable
	FileTypeSymbolicLink
	FileTypeArchive
	FileTypeMedia
	FileTypeBackup
)

// 配置文件类型
var (
	executableExtensions = []string{".exe", ".bat", ".cmd", ".ps1", ".sh", ".js", ".py", ".rb", ".pl", ".cs", ".vbs"}
	archiveExtensions    = []string{".7z", ".zip", ".rar", ".tar", ".gz", ".xz", ".bz2", ".cab", ".img", ".iso", ".jar", ".pea", ".rpm", ".tgz", ".z", ".deb", ".arj", ".lzh", ".lzma", ".lzma2", ".war", ".zst", ".part", ".s7z", ".split"}
	mediaExtensions      = []string{".aac", ".amr", ".caf", ".m3u", ".midi", ".mod", ".mp1", ".mp2", ".mp3", ".ogg", ".opus", ".ra", ".wma", ".wav", ".wv", ".3gp", ".3g2", ".asf", ".avi", ".flv", ".m4v", ".mkv", ".mov", ".mp4", ".mpeg", ".mpg", ".mpe", ".mts", ".rm", ".rmvb", ".swf", ".vob", ".webm", ".wmv", ".ai", ".avage", ".art", ".blend", ".cgm", ".cin", ".cur", ".cut", ".dcx", ".dng", ".dpx", ".emf", ".fit", ".fits", ".fpx", ".g3", ".hdr", ".ief", ".jbig", ".jfif", ".jls", ".jp2", ".jpc", ".jpx", ".jpg", ".jpeg", ".jxl", ".pbm", ".pcd", ".pcx", ".pgm", ".pict", ".png", ".pnm", ".ppm", ".psd", ".ras", ".rgb", ".svg", ".tga", ".tif", ".tiff", ".wbmp", ".xpm"}
	backupExtensions     = []string{".bak", ".backup", ".orig", ".old", ".tmp", ".temp", ".swap", ".chklist", ".chk", ".ms", ".diz", ".wbk", ".xlk", ".cdr_", ".nch", ".ftg", ".gid", ".syd"}

	// ANSI颜色代码
	ansiReset = "\033[0m"
	colorMap  = map[FileType]string{
		FileTypeDirectory:    "\033[94m", // 亮蓝色
		FileTypeExecutable:   "\033[32m", // 绿色
		FileTypeSymbolicLink: "\033[96m", // 亮青色
		FileTypeArchive:      "\033[91m", // 红色
		FileTypeMedia:        "\033[95m", // 紫色
		FileTypeBackup:       "\033[90m", // 灰色
		FileTypeOther:        ansiReset,  // 重置
	}

	// 文件类型标识符
	typeIndicators = map[FileType]string{
		FileTypeDirectory:    "/",
		FileTypeExecutable:   "*",
		FileTypeSymbolicLink: "@",
		FileTypeArchive:      "#",
		FileTypeMedia:        "~",
		FileTypeBackup:       "%",
		FileTypeOther:        "",
	}

	spaceLength = 2
)

// 命令行参数结构
type LSArgs struct {
	Path         string
	LongFormat   bool
	ShowFileType bool
	SetColor     bool
	ShowHelp     bool
	SearchTerm   string
	IgnoreCase   bool
	StrictCase   bool
	FilterType   string
}

// 辅助函数：创建渐变文本
func addGradient(text string, startRGB, endRGB [3]int) string {
	if isOutputRedirected() {
		return text
	}

	result := ""
	chars := []rune(text)
	for i, char := range chars {
		ratio := float64(i) / float64(len(chars)-1)
		if len(chars) == 1 {
			ratio = 0
		}
		
		r := int(float64(startRGB[0]) + (float64(endRGB[0])-float64(startRGB[0]))*ratio)
		g := int(float64(startRGB[1]) + (float64(endRGB[1])-float64(startRGB[1]))*ratio)
		b := int(float64(startRGB[2]) + (float64(endRGB[2])-float64(startRGB[2]))*ratio)

		result += fmt.Sprintf("\033[38;2;%d;%d;%dm%c", r, g, b, char)
	}
	return result + ansiReset
}

// 辅助函数：创建超链接
func createHyperlink(text, url string) string {
	if isOutputRedirected() {
		return text
	}
	return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, text)
}

// 获取帮助文本
func getHelpText() string {
	startRGB := [3]int{0, 150, 255}
	endRGB := [3]int{50, 255, 50}
	gradientTitle := addGradient("Enhanced-ls for PowerShell v0.02", startRGB, endRGB)
	link := createHyperlink(gradientTitle, "https://github.com/Geekstrange/enhanced-ls-for-powershell")

	return fmt.Sprintf(`
        %s

%s[96mOptions:%s
    %s[32m-f%s      append indicator (one of */@/#/~/%%) to entries.
    %s[32m-f id%s   only show entries of specified type (id: one of */@/#/~/%%)
    %s[32m-c,C%s    color the output.
    %s[32m-l,L%s    display items in a formatted table with borders.
    %s[32m-s%s      search files (case-insensitive).
    %s[32m-S%s      search files (case-sensitive).
    %s[32m-h%s      display this help message.

%s[96mFile Type Indicators:%s
    %s[94m/%s       Directory
    %s[94m*%s       Executable
    %s[94m@%s       Symbolic Link
    %s[94m#%s       Archive (compressed file)
    %s[94m~%s       Media file (audio/video/image)
    %s[94m%%%s       Backup/Temporary file

%s[96mExamples:%s
    %s[93m-f%s      Show all files with type indicators
    %s[93m-f #%s    Show only archive files
    %s[93m-f *%s    Show only executables
    %s[93m-f @ -c%s Show symbolic links with color
`,
		link,
		"\033", ansiReset,
		"\033", ansiReset,
		"\033", ansiReset,
		"\033", ansiReset,
		"\033", ansiReset,
		"\033", ansiReset,
		"\033", ansiReset,
		"\033", ansiReset,
		"\033", ansiReset,
		"\033", ansiReset,
		"\033", ansiReset,
		"\033", ansiReset,
		"\033", ansiReset,
		"\033", ansiReset,
		"\033", ansiReset,
		"\033", ansiReset,
		"\033", ansiReset,
		"\033", ansiReset,
		"\033", ansiReset,
		"\033", ansiReset,
	)
}

// 检查输出是否重定向
func isOutputRedirected() bool {
	return !term.IsTerminal(int(os.Stdout.Fd()))
}

// 计算字符串显示宽度（CJK字符计为2宽度）
func getStringDisplayWidth(s string) int {
	width := 0
	for _, r := range s {
		if isCJK(r) {
			width += 2
		} else {
			width += 1
		}
	}
	return width
}

// 检查是否为CJK字符
func isCJK(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || 
		(r >= 0x3400 && r <= 0x4DBF) || 
		(r >= 0x20000 && r <= 0x2A6DF) || 
		(r >= 0x2A700 && r <= 0x2B73F)
}

// 按显示宽度填充字符串
func padByWidth(s string, totalWidth int) string {
	currentWidth := getStringDisplayWidth(s)
	padding := totalWidth - currentWidth
	if padding <= 0 {
		return s
	}
	return s + strings.Repeat(" ", padding)
}

// 获取文件类型（关键修复）
func getFileType(info fs.FileInfo, path string) FileType {
	// 优先检测符号链接
	if info.Mode()&fs.ModeSymlink != 0 {
		return FileTypeSymbolicLink
	}

	// 然后检测目录
	if info.IsDir() {
		return FileTypeDirectory
	}

	ext := strings.ToLower(filepath.Ext(info.Name()))

	// 检测备份文件
	for _, backupExt := range backupExtensions {
		if ext == backupExt {
			return FileTypeBackup
		}
	}

	// 检测媒体文件
	for _, mediaExt := range mediaExtensions {
		if ext == mediaExt {
			return FileTypeMedia
		}
	}

	// 检测归档文件
	for _, archiveExt := range archiveExtensions {
		if ext == archiveExt {
			return FileTypeArchive
		}
	}

	// 检测可执行文件
	for _, execExt := range executableExtensions {
		if ext == execExt {
			return FileTypeExecutable
		}
	}

	return FileTypeOther
}

// 解析命令行参数
func parseArgs(args []string) (*LSArgs, error) {
	lsArgs := &LSArgs{
		Path: ".",
	}

	validOptions := map[rune]bool{
		'f': true, 'c': true, 'l': true, 's': true, 'S': true, 'h': true,
	}

	i := 0
	for i < len(args) {
		arg := args[i]

		if arg == "-h" {
			lsArgs.ShowHelp = true
			return lsArgs, nil
		}

		if strings.HasPrefix(arg, "-") {
			for _, r := range arg[1:] {
				if !validOptions[r] {
					lsArgs.ShowHelp = true
					return lsArgs, nil
				}
			}

			if strings.Contains(arg, "S") {
				i++
				if i < len(args) {
					lsArgs.SearchTerm = args[i]
					lsArgs.StrictCase = true
				}
			} else if strings.Contains(arg, "s") {
				i++
				if i < len(args) {
					lsArgs.SearchTerm = args[i]
					lsArgs.IgnoreCase = true
				}
			} else {
				for _, r := range strings.ToLower(arg[1:]) {
					switch r {
					case 'l':
						lsArgs.LongFormat = true
					case 'f':
						lsArgs.ShowFileType = true
						if i < len(args)-1 {
							next := args[i+1]
							if matched, _ := regexp.MatchString(`^[/*@#~%]$`, next); matched {
								lsArgs.FilterType = next
								i++
							}
						}
					case 'c':
						lsArgs.SetColor = true
					}
				}
			}
		} else {
			lsArgs.Path = arg
		}
		i++
	}

	return lsArgs, nil
}

// 获取终端宽度
func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80
	}
	return width
}

// 计算显示布局
func calculateLayout(displayWidths []int, windowWidth int) (rows, cols int, colWidths []int) {
	if len(displayWidths) == 0 {
		return 0, 0, nil
	}

	padding := spaceLength
	rows = len(displayWidths)
	cols = 1
	colWidths = []int{maxIntSlice(displayWidths)}

	calcWidth := func(displayWidths []int, padding, cols int) (int, []int) {
		maxWidths := make([]int, cols)
		perLines := int(math.Ceil(float64(len(displayWidths)) / float64(cols)))

		for i := 0; i < cols; i++ {
			startIdx := i * perLines
			endIdx := minInt(startIdx+perLines, len(displayWidths))
			if startIdx < len(displayWidths) {
				for j := startIdx; j < endIdx; j++ {
					if displayWidths[j] > maxWidths[i] {
						maxWidths[i] = displayWidths[j]
					}
				}
			}
		}

		sum := 0
		for _, w := range maxWidths {
			sum += w
		}
		return sum + (cols-1)*padding, maxWidths
	}

	for {
		nextCols := cols + 1
		if nextCols > len(displayWidths) {
			break
		}

		tmpWidth, tmpColWidths := calcWidth(displayWidths, padding, nextCols)
		if tmpWidth > windowWidth {
			break
		}

		colWidths = tmpColWidths
		cols = nextCols
	}

	rows = int(math.Ceil(float64(len(displayWidths)) / float64(cols)))
	return rows, cols, colWidths
}

// 辅助函数：切片最大值
func maxIntSlice(nums []int) int {
	if len(nums) == 0 {
		return 0
	}
	max := nums[0]
	for _, num := range nums[1:] {
		if num > max {
			max = num
		}
	}
	return max
}

// 辅助函数：两数最大值
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// 辅助函数：两数最小值
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 过滤项目
func filterItems(items []fs.FileInfo, paths []string, args *LSArgs) ([]fs.FileInfo, []string) {
	var filteredItems []fs.FileInfo
	var filteredPaths []string

	for i, item := range items {
		if args.SearchTerm != "" {
			name := item.Name()
			if args.IgnoreCase {
				if !strings.Contains(strings.ToLower(name), strings.ToLower(args.SearchTerm)) {
					continue
				}
			} else if args.StrictCase {
				if !strings.Contains(name, args.SearchTerm) {
					continue
				}
			}
		}

		if args.FilterType != "" {
			fileType := getFileType(item, paths[i])
			typeId := typeIndicators[fileType]
			if typeId != args.FilterType {
				continue
			}
		}

		filteredItems = append(filteredItems, item)
		filteredPaths = append(filteredPaths, paths[i])
	}

	return filteredItems, filteredPaths
}

// 长格式显示
func displayLongFormat(items []fs.FileInfo, paths []string, args *LSArgs) {
	nameDisplayWidth := 10
	if len(items) > 0 {
		maxWidth := 0
		for i, item := range items {
			baseName := item.Name()
			if args.ShowFileType {
				fileType := getFileType(item, paths[i])
				baseName += typeIndicators[fileType]
			}
			width := getStringDisplayWidth(baseName)
			if width > maxWidth {
				maxWidth = width
			}
		}
		nameDisplayWidth = maxInt(maxWidth, 10)
	}

	modeWidth := 10
	timeWidth := 16
	nameWidth := nameDisplayWidth

	topLine := "┌" + strings.Repeat("─", modeWidth) + "┬" + strings.Repeat("─", timeWidth) + "┬" + strings.Repeat("─", nameWidth) + "┐"
	header := "│" + padByWidth("Mode", modeWidth) + "│" + padByWidth("LastWriteTime", timeWidth) + "│" + padByWidth("Name", nameWidth) + "│"
	divider := "├" + strings.Repeat("─", modeWidth) + "┼" + strings.Repeat("─", timeWidth) + "┼" + strings.Repeat("─", nameWidth) + "┤"
	bottomLine := "└" + strings.Repeat("─", modeWidth) + "┴" + strings.Repeat("─", timeWidth) + "┴" + strings.Repeat("─", nameWidth) + "┘"

	fmt.Println(topLine)
	fmt.Println(header)
	fmt.Println(divider)

	for i, item := range items {
		mode := padByWidth(item.Mode().String(), modeWidth)
		timeStr := padByWidth(item.ModTime().Format("2006/01/02 15:04"), timeWidth)

		fileType := getFileType(item, paths[i])
		baseName := item.Name()
		if args.ShowFileType {
			baseName += typeIndicators[fileType]
		}

		currentWidth := getStringDisplayWidth(baseName)
		paddingSpaces := maxInt(0, nameWidth-currentWidth)

		var name string
		if !isOutputRedirected() && args.SetColor {
			color := colorMap[fileType]
			name = color + baseName + ansiReset + strings.Repeat(" ", paddingSpaces)
		} else {
			name = baseName + strings.Repeat(" ", paddingSpaces)
		}

		fmt.Printf("│%s│%s│%s│\n", mode, timeStr, name)
	}

	fmt.Println(bottomLine)
}

// 显示项目
func displayItems(items []fs.FileInfo, paths []string, args *LSArgs) {
	if len(items) == 0 {
		fmt.Println("No matching files found")
		return
	}

	var displayNames []string
	var displayWidths []int

	for i, item := range items {
		fileType := getFileType(item, paths[i])
		baseName := item.Name()

		if args.ShowFileType {
			baseName += typeIndicators[fileType]
		}

		var displayName string
		if !isOutputRedirected() && args.SetColor {
			color := colorMap[fileType]
			displayName = color + baseName + ansiReset
		} else {
			displayName = baseName
		}

		displayNames = append(displayNames, displayName)
		displayWidths = append(displayWidths, getStringDisplayWidth(baseName))
	}

	windowWidth := getTerminalWidth()
	rows, _, colWidths := calculateLayout(displayWidths, windowWidth)

	lines := make([][]string, rows)
	for i := range lines {
		lines[i] = make([]string, 0)
	}

	for idx := 0; idx < len(displayNames); idx++ {
		x := idx / rows
		y := idx % rows

		name := displayNames[idx]
		if x < len(colWidths) {
			padding := colWidths[x] - displayWidths[idx]
			if padding > 0 {
				name += strings.Repeat(" ", padding)
			}
		}

		lines[y] = append(lines[y], name)
	}

	space := strings.Repeat(" ", spaceLength)
	for _, line := range lines {
		fmt.Println(strings.Join(line, space))
	}
}

// 主函数
func main() {
	args, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing arguments: %v\n", err)
		os.Exit(1)
	}

	if args.ShowHelp {
		fmt.Print(getHelpText())
		return
	}

	// 关键修复：使用Lstat获取符号链接本身属性
	var items []fs.FileInfo
	var paths []string
	entries, err := os.ReadDir(args.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading directory: %v\n", err)
		os.Exit(1)
	}

	for _, entry := range entries {
		fullPath := filepath.Join(args.Path, entry.Name())
		info, err := os.Lstat(fullPath) // 使用Lstat而非entry.Info()
		if err != nil {
			continue
		}
		items = append(items, info)
		paths = append(paths, fullPath)
	}

	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name()) < strings.ToLower(items[j].Name())
	})

	items, paths = filterItems(items, paths, args)

	if args.LongFormat {
		displayLongFormat(items, paths, args)
	} else {
		displayItems(items, paths, args)
	}
}
