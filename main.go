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
	Recursive    bool
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
		ratio := float64(i) / float64(len(chars)-1) // 修复括号问题
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
	gradientTitle := addGradient("Enhanced-ls v0.0.8 (Cross-Platform)", startRGB, endRGB)
	link := createHyperlink(gradientTitle, "https://github.com/Geekstrange/enhanced-ls")

	// 定义颜色变量
	reset := ansiReset
	cyan := "\033[96m"
	green := "\033[32m"
	blue := "\033[94m"
	yellow := "\033[93m"

	return fmt.Sprintf(`
        %s

%sOptions:%s
    %s-f%s        append indicator (one of */@/#~/%%) to entries.
    %s-f id%s     only show entries of specified type (id: one of */@/#~/%%)
    %s-c%s        color the output.
    %s-l%s        display items in a formatted table with borders.
    %s-r%s        recursively list subdirectories (tree view).
    %s-s%s        search files (case-insensitive).
    %s-S%s        search files (case-sensitive).
    %s-h%s        display this help message.

%sFile Type Indicators:%s
    %s/%s         Directory
    %s*%s         Executable
    %s@%s         Symbolic Link
    %s#%s         Archive (compressed file)
    %s~%s         Media file (audio/video/image)
    %s%%%s         Backup/Temporary file

%sExamples:%s
    %s-f%s        Show all files with type indicators
    %s-f #%s      Show only archive files
    %s-f *%s      Show only executables
    %s-fc @%s     Show symbolic links with color
    %s-r%s        Recursive directory listing (tree view)
    %s-r -s go%s  Recursive search for "go" (case-insensitive)
    %s-r -S Go%s  Recursive search for "Go" (case-sensitive)
    %s-r -f #%s   Recursive listing of archive files

%sSupported Platforms:%s
    %s- Windows%s x86_64/ARM64
    %s- Linux%s   x86_64/ARM64/LoongArch
    %s- macOS%s   x86_64/ARM64
`,
		link,
		cyan, reset,
		green, reset,
		green, reset,
		green, reset,
		green, reset,
		green, reset,
		green, reset,
		green, reset,
		green, reset,
		cyan, reset,
		blue, reset,
		blue, reset,
		blue, reset,
		blue, reset,
		blue, reset,
		blue, reset,
		cyan, reset,
		yellow, reset,
		yellow, reset,
		yellow, reset,
		yellow, reset,
		yellow, reset,
		yellow, reset,
		yellow, reset,
		yellow, reset,
		cyan, reset,
		yellow, reset,
		yellow, reset,
		yellow, reset,
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
	// 优先检查符号链接
	if isSymbolicLink(path) {
		return FileTypeSymbolicLink
	}

	// 然后检查目录
	if info.IsDir() {
		return FileTypeDirectory
	}

	// 在非Windows系统检查可执行权限
	if runtime.GOOS != "windows" {
		if info.Mode()&0111 != 0 {
			return FileTypeExecutable
		}
	}

	// 最后检查文件扩展名
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

	// 在Windows系统检查可执行扩展名
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

	validOptions := "fclrSsSh"

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
				lsArgs.StrictCase = true
				if i < len(args)-1 && !strings.HasPrefix(args[i+1], "-") {
					i++
					lsArgs.SearchTerm = args[i]
				}
			} else if strings.Contains(options, "s") {
				lsArgs.IgnoreCase = true
				if i < len(args)-1 && !strings.HasPrefix(args[i+1], "-") {
					i++
					lsArgs.SearchTerm = args[i]
				}
			} else {
				for _, r := range options {
					switch r {
					case 'l':
						lsArgs.LongFormat = true
					case 'f':
						lsArgs.ShowFileType = true
						if i < len(args)-1 && !strings.HasPrefix(args[i+1], "-") {
							next := args[i+1]
							if len(next) == 1 && strings.ContainsAny(next, "/*@#~%") {
								lsArgs.FilterType = next
								i++
							}
						}
					case 'c':
						lsArgs.SetColor = true
					case 'r':
						lsArgs.Recursive = true
					}
				}
			}
		} else {
			// 处理路径参数
			if lsArgs.Path == "." {
				lsArgs.Path = arg
			} else {
				// 如果有多个路径参数,只取第一个
				fmt.Fprintf(os.Stderr, "Warning: multiple paths not supported, using first path: %s\n", lsArgs.Path)
			}
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

func passesFilter(name string, fileType FileType, args *LSArgs) bool {
	if args.SearchTerm != "" {
		if args.IgnoreCase {
			if !strings.Contains(strings.ToLower(name), strings.ToLower(args.SearchTerm)) {
				return false
			}
		} else if args.StrictCase {
			if !strings.Contains(name, args.SearchTerm) {
				return false
			}
		}
	}

	if args.FilterType != "" {
		typeId := typeIndicators[fileType]
		if typeId != args.FilterType {
			return false
		}
	}

	return true
}

func getOwner() string {
	return currentUser
}

func getLinkCount(info fs.FileInfo) uint64 {
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

func displayTree(path string, args *LSArgs, level int, prefix string) {
	info, err := os.Lstat(path)
	if err != nil {
		return
	}

	name := info.Name()
	fileType := getFileType(info, path)

	// 应用过滤条件
	if !passesFilter(name, fileType, args) {
		return
	}

	// 添加类型指示器
	displayName := name
	if args.ShowFileType {
		displayName += typeIndicators[fileType]
	}

	// 应用颜色
	if !isOutputRedirected() && args.SetColor {
		color := colorMap[fileType]
		displayName = color + displayName + ansiReset
	}

	// 打印当前条目
	if level == 0 {
		// 根目录
		fmt.Println(displayName)
	} else {
		fmt.Printf("%s%s\n", prefix, displayName)
	}

	// 如果是目录,递归处理
	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return
		}

		// 过滤掉隐藏文件 (以.开头)
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
			entryInfo, err := os.Lstat(fullPath)
			if err != nil {
				continue
			}

			childName := entryInfo.Name()
			childType := getFileType(entryInfo, fullPath)

			// 对子项应用过滤条件
			if !passesFilter(childName, childType, args) {
				continue
			}

			// 确定连接线
			connector := "├── "
			newPrefix := prefix + "│   "
			if i == len(entries)-1 {
				connector = "└── "
				newPrefix = prefix + "    "
			}

			// 添加类型指示器
			childDisplayName := childName
			if args.ShowFileType {
				childDisplayName += typeIndicators[childType]
			}

			// 应用颜色
			if !isOutputRedirected() && args.SetColor {
				color := colorMap[childType]
				childDisplayName = color + childDisplayName + ansiReset
			}

			fmt.Printf("%s%s%s\n", prefix, connector, childDisplayName)

			// 如果子项是目录,递归处理
			if entry.IsDir() {
				displayTree(fullPath, args, level+1, newPrefix)
			}
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

	// 递归模式处理
	if args.Recursive {
		displayTree(args.Path, args, 0, "")
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

		fileType := getFileType(info, fullPath)
		if !passesFilter(entry.Name(), fileType, args) {
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

	if args.LongFormat {
		displayLongFormat(items, args)
	} else {
		displayItems(items, args)
	}
}
