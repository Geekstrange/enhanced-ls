package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"golang.org/x/term"
)

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

var (
	executableExtensions = []string{".exe", ".bat", ".cmd", ".ps1", ".sh", ".js", ".py", ".rb", ".pl", ".cs", ".vbs"}
	archiveExtensions    = []string{".7z", ".zip", ".rar", ".tar", ".gz", ".xz", ".bz2", ".cab", ".img", ".iso", ".jar", ".pea", ".rpm", ".tgz", ".z", ".deb", ".arj", ".lzh", ".lzma", ".lzma2", ".war", ".zst", ".part", ".s7z", ".split"}
	mediaExtensions      = []string{".aac", ".amr", ".caf", ".m3u", ".midi", ".mod", ".mp1", ".mp2", ".mp3", ".ogg", ".opus", ".ra", ".wma", ".wav", ".wv", ".3gp", ".3g2", ".asf", ".avi", ".flv", ".m4v", ".mkv", ".mov", ".mp4", ".mpeg", ".mpg", ".mpe", ".mts", ".rm", ".rmvb", ".swf", ".vob", ".webm", ".wmv", ".ai", ".avage", ".art", ".blend", ".cgm", ".cin", ".cur", ".cut", ".dcx", ".dng", ".dpx", ".emf", ".fit", ".fits", ".fpx", ".g3", ".hdr", ".ief", ".jbig", ".jfif", ".jls", ".jp2", ".jpc", ".jpx", ".jpg", ".jpeg", ".jxl", ".pbm", ".pcd", ".pcx", ".pgm", ".pict", ".png", ".pnm", ".ppm", ".psd", ".ras", ".rgb", ".svg", ".tga", ".tif", ".tiff", ".wbmp", ".xpm"}
	backupExtensions     = []string{".bak", ".backup", ".orig", ".old", ".tmp", ".temp", ".swap", ".chklist", ".chk", ".ms", ".diz", ".wbk", ".xlk", ".cdr_", ".nch", ".ftg", ".gid", ".syd"}

	ansiReset = "\033[0m"
	colorMap  = map[FileType]string{
		FileTypeDirectory:    "\033[94m", // 亮蓝色
		FileTypeExecutable:   "\033[32m", // 绿色
		FileTypeSymbolicLink: "\033[96m", // 亮青色
		FileTypeArchive:      "\033[91m", // 红色
		FileTypeMedia:        "\033[95m", // 紫色
		FileTypeBackup:       "\033[90m", // 灰色
		FileTypeOther:        ansiReset,
	}

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

func createHyperlink(text, url string) string {
	if isOutputRedirected() {
		return text
	}
	return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, text)
}

func getHelpText() string {
	startRGB := [3]int{0, 150, 255}
	endRGB := [3]int{50, 255, 50}
	gradientTitle := addGradient("Enhanced-ls v0.05 (Cross-Platform)", startRGB, endRGB)
	link := createHyperlink(gradientTitle, "https://github.com/Geekstrange/linux-like-ls-for-powershell")

	return fmt.Sprintf(`
        %s

%s[96mOptions:%s
    %s[32m-f%s      append indicator (one of */@/#/~/%%) to entries.
    %s[32m-f id%s   only show entries of specified type (id: one of */@/#/~/%%)
    %s[32m-c%s      color the output.
    %s[32m-l%s      display items in a formatted table with borders.
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
    %s[93m-fc @%s   Show symbolic links with color

%s[96mSupported Platforms:%s
    %s[93m- Windows (PowerShell 7.5+)%s
    %s[93m- Linux%s
    %s[93m- macOS%s
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
		"\033", ansiReset,
		"\033", ansiReset,
		"\033", ansiReset,
		"\033", ansiReset,
	)
}

func isOutputRedirected() bool {
	file := os.Stdout
	stat, err := file.Stat()
	if err != nil {
		return true
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

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

func isCJK(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) ||
		(r >= 0x3400 && r <= 0x4DBF) ||
		(r >= 0x20000 && r <= 0x2A6DF) ||
		(r >= 0x2A700 && r <= 0x2B73F)
}

func padByWidth(s string, totalWidth int) string {
	currentWidth := getStringDisplayWidth(s)
	padding := totalWidth - currentWidth
	if padding <= 0 {
		return s
	}
	return s + strings.Repeat(" ", padding)
}

func getFileType(info fs.FileInfo, path string) FileType {
	// 优先检测符号链接
	if isSymbolicLink(info) {
		return FileTypeSymbolicLink
	}

	if info.IsDir() {
		return FileTypeDirectory
	}

	// 在Linux/macOS上检测可执行文件
	if runtime.GOOS != "windows" {
		if info.Mode()&0111 != 0 {
			return FileTypeExecutable
		}
	}

	ext := strings.ToLower(filepath.Ext(info.Name()))

	for _, backupExt := range backupExtensions {
		if ext == backupExt {
			return FileTypeBackup
		}
	}

	for _, mediaExt := range mediaExtensions {
		if ext == mediaExt {
			return FileTypeMedia
		}
	}

	for _, archiveExt := range archiveExtensions {
		if ext == archiveExt {
			return FileTypeArchive
		}
	}

	// 在Windows上检测可执行文件
	if runtime.GOOS == "windows" {
		for _, execExt := range executableExtensions {
			if ext == execExt {
				return FileTypeExecutable
			}
		}
	}

	return FileTypeOther
}

func isSymbolicLink(info fs.FileInfo) bool {
	// 所有平台通用的符号链接检测方法
	return info.Mode()&os.ModeSymlink != 0
}

func parseArgs(args []string) (*LSArgs, error) {
	lsArgs := &LSArgs{
		Path: ".",
	}

	validOptions := "fclSsSh"

	i := 0
	for i < len(args) {
		arg := args[i]

		if arg == "-h" {
			lsArgs.ShowHelp = true
			return lsArgs, nil
		}

		if strings.HasPrefix(arg, "-") {
			options := arg[1:]
			if options == "" {
				return lsArgs, fmt.Errorf("invalid option: %s", arg)
			}

			for _, r := range options {
				if !strings.ContainsRune(validOptions, r) {
					lsArgs.ShowHelp = true
					return lsArgs, nil
				}
			}

			if strings.Contains(options, "S") {
				i++
				if i < len(args) {
					lsArgs.SearchTerm = args[i]
					lsArgs.StrictCase = true
				}
			} else if strings.Contains(options, "s") {
				i++
				if i < len(args) {
					lsArgs.SearchTerm = args[i]
					lsArgs.IgnoreCase = true
				}
			} else {
				for _, r := range options {
					switch r {
					case 'l':
						lsArgs.LongFormat = true
					case 'f':
						lsArgs.ShowFileType = true
						if i < len(args)-1 {
							next := args[i+1]
							if len(next) == 1 && strings.ContainsAny(next, "/*@#~%") {
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

func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		return 80
	}
	return width
}

func calculateLayout(displayWidths []int, windowWidth int) (rows, cols int, colWidths []int) {
	if len(displayWidths) == 0 {
		return 0, 0, nil
	}

	padding := spaceLength
	cols = 1
	colWidths = []int{maxIntSlice(displayWidths)}

	calcWidth := func(displayWidths []int, padding, cols int) (int, []int) {
		maxWidths := make([]int, cols)
		perLines := (len(displayWidths) / cols)
		if len(displayWidths)%cols != 0 {
			perLines++
		}

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

	rows = (len(displayWidths) / cols)
	if len(displayWidths)%cols != 0 {
		rows++
	}
	return rows, cols, colWidths
}

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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func filterItems(items []fs.FileInfo, paths []string, args *LSArgs) ([]fs.FileInfo, []string) {
	var filteredItems []fs.FileInfo
	var filteredPaths []string

	for i, item := range items {
		name := item.Name()

		if args.SearchTerm != "" {
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

func displayLongFormat(items []fs.FileInfo, paths []string, args *LSArgs) {
	// 动态计算列的最大宽度
	modeWidth := 4
	timeWidth := 16
	nameWidth := 4

	for i, item := range items {
		// Mode 列
		modeStr := item.Mode().String()
		if w := len(modeStr); w > modeWidth {
			modeWidth = w
		}

		// Name 列
		baseName := item.Name()
		if args.ShowFileType {
			fileType := getFileType(item, paths[i])
			baseName += typeIndicators[fileType]
		}
		if w := getStringDisplayWidth(baseName); w > nameWidth {
			nameWidth = w
		}
	}

	// 设置最小宽度
	modeWidth = maxInt(modeWidth, 4)
	timeWidth = maxInt(timeWidth, 16)
	nameWidth = maxInt(nameWidth, 4)

	// 创建表格边框
	topLine := "┌" + strings.Repeat("─", modeWidth) + "┬" +
		strings.Repeat("─", timeWidth) + "┬" +
		strings.Repeat("─", nameWidth) + "┐"

	header := "│" + padByWidth("Mode", modeWidth) + "│" +
		padByWidth("LastWriteTime", timeWidth) + "│" +
		padByWidth("Name", nameWidth) + "│"

	divider := "├" + strings.Repeat("─", modeWidth) + "┼" +
		strings.Repeat("─", timeWidth) + "┼" +
		strings.Repeat("─", nameWidth) + "┤"

	bottomLine := "└" + strings.Repeat("─", modeWidth) + "┴" +
		strings.Repeat("─", timeWidth) + "┴" +
		strings.Repeat("─", nameWidth) + "┘"

	fmt.Println(topLine)
	fmt.Println(header)
	fmt.Println(divider)

	for i, item := range items {
		mode := padByWidth(item.Mode().String(), modeWidth)
		timeStr := item.ModTime().Format("2006/01/02 15:04")
		if len(timeStr) > timeWidth {
			timeStr = timeStr[:timeWidth]
		} else {
			timeStr = padByWidth(timeStr, timeWidth)
		}

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
		row := idx % rows
		col := idx / rows

		if col >= len(colWidths) {
			continue
		}

		name := displayNames[idx]
		padding := colWidths[col] - displayWidths[idx]
		if padding > 0 {
			name += strings.Repeat(" ", padding)
		}

		lines[row] = append(lines[row], name)
	}

	space := strings.Repeat(" ", spaceLength)
	for _, line := range lines {
		fmt.Println(strings.Join(line, space))
	}
}

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

	// 确保路径格式正确
	args.Path = filepath.Clean(args.Path)
	if runtime.GOOS == "windows" {
		args.Path = strings.ReplaceAll(args.Path, "/", "\\")
	}

	var items []fs.FileInfo
	var paths []string
	entries, err := os.ReadDir(args.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading directory: %v\n", err)
		os.Exit(1)
	}

	for _, entry := range entries {
		fullPath := filepath.Join(args.Path, entry.Name())
		info, err := os.Lstat(fullPath)
		if err != nil {
			continue
		}
		items = append(items, info)
		paths = append(paths, fullPath)
	}

	// 按文件名排序
	sort.Slice(items, func(i, j int) bool {
		if runtime.GOOS == "windows" {
			return strings.ToLower(items[i].Name()) < strings.ToLower(items[j].Name())
		}
		return items[i].Name() < items[j].Name()
	})

	items, paths = filterItems(items, paths, args)

	if args.LongFormat {
		displayLongFormat(items, paths, args)
	} else {
		displayItems(items, paths, args)
	}
}
