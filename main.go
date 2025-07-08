package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
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
	executableExtensions = []string{".appx", ".exe", ".com", ".bat", ".cmd", ".ps1", ".vbs", ".msi", ".msix", ".msm", ".msp", ".mst", ".scr", ".app", ".command", ".workflow", ".sh", ".out", ".bin", ".run", ".py", ".rb", ".pl", ".js", ".jar", ".lua", ".ahk", ".ipa", ".apk"}
	archiveExtensions    = []string{".7z", ".zip", ".rar", ".tar", ".gz", ".xz", ".bz2", ".cab", ".img", ".iso", ".jar", ".pea", ".rpm", ".tgz", ".z", ".deb", ".arj", ".lzh", ".lzma", ".lzma2", ".war", ".zst", ".part", ".s7z", ".split"}
	mediaExtensions      = []string{".aac", ".amr", ".caf", ".m3u", ".midi", ".mod", ".mp1", ".mp2", ".mp3", ".ogg", ".opus", ".ra", ".wma", ".wav", ".wv", ".3gp", ".3g2", ".asf", ".avi", ".flv", ".m4v", ".mkv", ".mov", ".mp4", ".mpeg", ".mpg", ".mpe", ".mts", ".rm", ".rmvb", ".swf", ".vob", ".webm", ".wmv", ".ai", ".avage", ".art", ".blend", ".cgm", ".cin", ".cur", ".cut", ".dcx", ".dng", ".dpx", ".emf", ".fit", ".fits", ".fpx", ".g3", ".hdr", ".ief", ".jbig", ".jfif", ".jls", ".jp2", ".jpc", ".jpx", ".jpg", ".jpeg", ".jxl", ".pbm", ".pcd", ".pcx", ".pgm", ".pict", ".png", ".pnm", ".ppm", ".psd", ".ras", ".rgb", ".svg", ".tga", ".tif", ".tiff", ".wbmp", ".xpm"}
	backupExtensions     = []string{".bak", ".backup", ".orig", ".old", ".tmp", ".temp", ".swap", ".chklist", ".chk", ".ms", ".diz", ".wbk", ".xlk", ".cdr_", ".nch", ".ftg", ".gid", ".syd"}

	ansiReset    = "\033[0m"
	colorMap     = map[FileType]string{
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
	currentUser = "user"
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
	Recursive    bool // 新增递归选项
}

type FileInfoEx struct {
	fs.FileInfo
	Path      string
	Links     uint64
	OwnerName string
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
	gradientTitle := addGradient("Enhanced-ls v0.07 (Cross-Platform)", startRGB, endRGB)
	link := createHyperlink(gradientTitle, "https://github.com/Geekstrange/enhanced-ls")

	return fmt.Sprintf(`
        %s

%s[96mOptions:%s
    %s[32m-f%s        append indicator (one of */@/#~/%%) to entries.
    %s[32m-f id%s     only show entries of specified type (id: one of */@/#~/%%)
    %s[32m-c%s        color the output.
    %s[32m-l%s        display items in a formatted table with borders.
    %s[32m-r%s        recursively list subdirectories (tree view).
    %s[32m-s%s        search files (case-insensitive).
    %s[32m-S%s        search files (case-sensitive).
    %s[32m-h%s        display this help message.

%s[96mFile Type Indicators:%s
    %s[94m/%s         Directory
    %s[94m*%s         Executable
    %s[94m@%s         Symbolic Link
    %s[94m#%s         Archive (compressed file)
    %s[94m~%s         Media file (audio/video/image)
    %s[94m%%%s         Backup/Temporary file

%s[96mExamples:%s
    %s[93m-f%s        Show all files with type indicators
    %s[93m-f #%s      Show only archive files
    %s[93m-f *%s      Show only executables
    %s[93m-fc @%s     Show symbolic links with color
    %s[93m-r%s        Recursive directory listing (tree view)

%s[96mSupported Platforms:%s
    %s[93m- Windows%s x86_64/ARM64
    %s[93m- Linux%s   x86_64/ARM64/LoongArch
    %s[93m- macOS%s   x86_64/ARM64
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

func padLeftByWidth(s string, totalWidth int) string {
	currentWidth := getStringDisplayWidth(s)
	padding := totalWidth - currentWidth
	if padding <= 0 {
		return s
	}
	return strings.Repeat(" ", padding) + s
}

func isSymbolicLink(path string) bool {
	_, err := os.Readlink(path)
	return err == nil
}

func getFileType(info fs.FileInfo, path string) FileType {
	if isSymbolicLink(path) {
		return FileTypeSymbolicLink
	}

	if info.IsDir() {
		return FileTypeDirectory
	}

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

	if runtime.GOOS == "windows" {
		for _, execExt := range executableExtensions {
			if ext == execExt {
				return FileTypeExecutable
			}
		}
	}

	return FileTypeOther
}

func parseArgs(args []string) (*LSArgs, error) {
	lsArgs := &LSArgs{
		Path: ".",
	}

	validOptions := "fclrSsSh" // 添加r选项

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
					case 'r': // 处理递归选项
						lsArgs.Recursive = true
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

func filterItems(items []FileInfoEx, args *LSArgs) ([]FileInfoEx, []string) {
	var filteredItems []FileInfoEx
	var filteredPaths []string

	for _, item := range items {
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
			fileType := getFileType(item.FileInfo, item.Path)
			typeId := typeIndicators[fileType]
			if typeId != args.FilterType {
				continue
			}
		}

		filteredItems = append(filteredItems, item)
		filteredPaths = append(filteredPaths, item.Path)
	}

	return filteredItems, filteredPaths
}

func getOwner() string {
	return currentUser
}

func getLinkCount(info fs.FileInfo) uint64 {
	// 目录返回2，文件返回1作为占位值
	if info.IsDir() {
		return 2
	}
	return 1
}

func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%c",
		float64(size)/float64(div), "KMGTPE"[exp])
}

func displayLongFormat(items []FileInfoEx, args *LSArgs) {
	if len(items) == 0 {
		fmt.Println("No matching files found")
		return
	}

	modeWidth := 10
	linksWidth := 6
	ownerWidth := 9
	sizeWidth := 8
	timeWidth := 16
	nameWidth := 12

	// 计算每列的最大宽度
	for _, item := range items {
		baseName := item.Name()
		if args.ShowFileType {
			fileType := getFileType(item.FileInfo, item.Path)
			baseName += typeIndicators[fileType]
		}

		if w := len(item.Mode().String()); w > modeWidth {
			modeWidth = w
		}
		if w := len(strconv.FormatUint(item.Links, 10)); w > linksWidth {
			linksWidth = w
		}
		if w := getStringDisplayWidth(item.OwnerName); w > ownerWidth {
			ownerWidth = w
		}
		if w := len(formatSize(item.Size())); w > sizeWidth {
			sizeWidth = w
		}
		if w := getStringDisplayWidth(baseName); w > nameWidth {
			nameWidth = w
		}
	}

	// 确保最小宽度
	modeWidth = maxInt(modeWidth, 4)
	linksWidth = maxInt(linksWidth, 5)
	ownerWidth = maxInt(ownerWidth, 5)
	sizeWidth = maxInt(sizeWidth, 4)
	timeWidth = maxInt(timeWidth, 15)
	nameWidth = maxInt(nameWidth, 4)

	topLine := "┌" + strings.Repeat("─", modeWidth) + "┬" +
		strings.Repeat("─", linksWidth) + "┬" +
		strings.Repeat("─", ownerWidth) + "┬" +
		strings.Repeat("─", sizeWidth) + "┬" +
		strings.Repeat("─", timeWidth) + "┬" +
		strings.Repeat("─", nameWidth) + "┐"

	header := "│" + padByWidth("Mode", modeWidth) + "│" +
		padByWidth("Links", linksWidth) + "│" +
		padByWidth("Owner", ownerWidth) + "│" +
		padByWidth("Size", sizeWidth) + "│" +
		padByWidth("Change Time", timeWidth) + "│" +
		padByWidth("Name", nameWidth) + "│"

	divider := "├" + strings.Repeat("─", modeWidth) + "┼" +
		strings.Repeat("─", linksWidth) + "┼" +
		strings.Repeat("─", ownerWidth) + "┼" +
		strings.Repeat("─", sizeWidth) + "┼" +
		strings.Repeat("─", timeWidth) + "┼" +
		strings.Repeat("─", nameWidth) + "┤"

	bottomLine := "└" + strings.Repeat("─", modeWidth) + "┴" +
		strings.Repeat("─", linksWidth) + "┴" +
		strings.Repeat("─", ownerWidth) + "┴" +
		strings.Repeat("─", sizeWidth) + "┴" +
		strings.Repeat("─", timeWidth) + "┴" +
		strings.Repeat("─", nameWidth) + "┘"

	fmt.Println(topLine)
	fmt.Println(header)
	fmt.Println(divider)

	for _, item := range items {
		mode := padByWidth(item.Mode().String(), modeWidth)
		links := padLeftByWidth(strconv.FormatUint(item.Links, 10), linksWidth)
		owner := padByWidth(item.OwnerName, ownerWidth)
		size := padLeftByWidth(formatSize(item.Size()), sizeWidth)

		timeStr := item.ModTime().Format("2006/01/02 15:04")
		if len(timeStr) > timeWidth {
			timeStr = timeStr[:timeWidth]
		} else {
			timeStr = padByWidth(timeStr, timeWidth)
		}

		fileType := getFileType(item.FileInfo, item.Path)
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

		fmt.Printf("│%s│%s│%s│%s│%s│%s│\n", mode, links, owner, size, timeStr, name)
	}

	fmt.Println(bottomLine)
}

func displayItems(items []FileInfoEx, args *LSArgs) {
	if len(items) == 0 {
		fmt.Println("No matching files found")
		return
	}

	var displayNames []string
	var displayWidths []int

	for _, item := range items {
		fileType := getFileType(item.FileInfo, item.Path)
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

// 递归显示目录树
func displayTree(root string, args *LSArgs) {
	fmt.Println(root)
	displayTreeRecursive(root, "", args)
}

func displayTreeRecursive(path string, prefix string, args *LSArgs) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return
	}

	// 过滤掉隐藏文件（以.开头）
	var visibleEntries []fs.DirEntry
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), ".") {
			visibleEntries = append(visibleEntries, entry)
		}
	}
	entries = visibleEntries

	// 排序条目
	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})

	for i, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())
		info, err := os.Lstat(fullPath)
		if err != nil {
			continue
		}

		// 确定连接线
		connector := "├── "
		if i == len(entries)-1 {
			connector = "└── "
		}

		// 获取文件类型
		fileType := getFileType(info, fullPath)
		typeIndicator := ""
		if args.ShowFileType {
			typeIndicator = typeIndicators[fileType]
		}

		// 设置颜色
		var displayName string
		if !isOutputRedirected() && args.SetColor {
			color := colorMap[fileType]
			displayName = color + entry.Name() + typeIndicator + ansiReset
		} else {
			displayName = entry.Name() + typeIndicator
		}

		// 打印当前条目
		fmt.Printf("%s%s%s\n", prefix, connector, displayName)

		// 如果是目录，递归处理
		if entry.IsDir() {
			newPrefix := prefix
			if i == len(entries)-1 {
				newPrefix += "    "
			} else {
				newPrefix += "│   "
			}
			displayTreeRecursive(fullPath, newPrefix, args)
		}
	}
}

func init() {
	// 初始化当前用户信息
	u, err := user.Current()
	if err == nil {
		currentUser = u.Username
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

	args.Path = filepath.Clean(args.Path)
	if runtime.GOOS == "windows" {
		args.Path = strings.ReplaceAll(args.Path, "/", "\\")
	}

	// 检查路径是否存在
	fileInfo, err := os.Stat(args.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error accessing path: %v\n", err)
		os.Exit(1)
	}

	// 如果是递归模式，直接显示目录树
	if args.Recursive {
		if fileInfo.IsDir() {
			displayTree(args.Path, args)
		} else {
			fmt.Println(args.Path)
		}
		return
	}

	// 非递归模式
	var items []FileInfoEx
	var entries []fs.DirEntry
	var fullPath string

	if fileInfo.IsDir() {
		entries, err = os.ReadDir(args.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading directory: %v\n", err)
			os.Exit(1)
		}
	} else {
		// 处理单个文件
		fullPath = args.Path
		info, err := os.Lstat(fullPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error accessing file: %v\n", err)
			os.Exit(1)
		}
		owner := currentUser
		links := getLinkCount(info)

		items = append(items, FileInfoEx{
			FileInfo:  info,
			Path:      fullPath,
			Links:     links,
			OwnerName: owner,
		})
	}

	// 处理目录中的多个文件
	for _, entry := range entries {
		fullPath = filepath.Join(args.Path, entry.Name())
		info, err := os.Lstat(fullPath)
		if err != nil {
			continue
		}

		owner := currentUser
		links := getLinkCount(info)

		items = append(items, FileInfoEx{
			FileInfo:  info,
			Path:      fullPath,
			Links:     links,
			OwnerName: owner,
		})
	}

	// 排序条目
	sort.Slice(items, func(i, j int) bool {
		if runtime.GOOS == "windows" {
			return strings.ToLower(items[i].Name()) < strings.ToLower(items[j].Name())
		}
		return items[i].Name() < items[j].Name()
	})

	filteredItems, _ := filterItems(items, args)

	if args.LongFormat {
		displayLongFormat(filteredItems, args)
	} else {
		displayItems(filteredItems, args)
	}
}
