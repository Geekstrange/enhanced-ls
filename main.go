package main

import (
	"fmt"
	"io/fs"
	"math"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// ─────────────────────────────────────────────
// Types & constants
// ─────────────────────────────────────────────

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

// ValidTypeIndicators is the canonical set of filter characters, derived
// from the typeIndicators map in init().
const ValidTypeIndicators = "/*@#~%"

var (
	executableExtensions = []string{
		".appx", ".exe", ".com", ".bat", ".cmd", ".ps1", ".vbs",
		".msi", ".msix", ".msixbundle", ".msm", ".msp", ".mst",
		".cpl", ".wsf", ".psm1", ".elf", ".bash", ".zsh", ".php",
		".scr", ".app", ".command", ".workflow", ".ts", ".wasm",
		".sh", ".out", ".bin", ".run", ".desktop", ".reg", ".ipa",
		".py", ".rb", ".pl", ".js", ".jar", ".lua", ".ahk",
		".msc", ".jse", ".vbe", ".scpt", ".pif", ".gadget",
	}
	archiveExtensions = []string{
		".7z", ".zip", ".rar", ".tar", ".gz", ".xz", ".bz2",
		".cab", ".img", ".iso", ".pea", ".rpm", ".tgz", ".qcow2",
		".z", ".deb", ".arj", ".lzh", ".lzma", ".lzma2", ".zipx",
		".war", ".zst", ".part", ".s7z", ".split", ".aar", ".br",
		".wim", ".esd", ".apk", ".dmg", ".pkg", ".ear", ".lz",
		".crx", ".xpi", ".cpio", ".lha", ".sitx", ".vmdk",
	}
	mediaExtensions = []string{
		".aac", ".amr", ".caf", ".m3u", ".midi", ".mod",
		".mp1", ".mp2", ".mp3", ".ogg", ".opus", ".ra", ".wma",
		".wav", ".wv", ".m4a", ".flac", ".alac", ".aiff", ".ape",
		".3gp", ".3g2", ".asf", ".avi", ".flv", ".m4v", ".mkv",
		".mov", ".mp4", ".mpeg", ".mpg", ".mpe", ".mts", ".rm",
		".rmvb", ".swf", ".vob", ".webm", ".wmv", ".ogv", ".m2ts",
		".ai", ".art", ".blend", ".cgm", ".cin", ".cur", ".cut",
		".dcx", ".dng", ".dpx", ".emf", ".fit", ".fits", ".fpx",
		".g3", ".hdr", ".ief", ".jbig", ".jfif", ".jls", ".jp2",
		".jpc", ".jpx", ".jpg", ".jpeg", ".jxl", ".raw", ".cr2",
		".pbm", ".pcd", ".pcx", ".pgm", ".pict", ".png", ".pnm",
		".ppm", ".psd", ".ras", ".rgb", ".svg", ".tga", ".tif",
		".tiff", ".wbmp", ".xpm", ".bmp", ".webp", ".avif", ".ico",
		".heic", ".heif", ".nef", ".arw", ".psb", ".glb", ".gltf",
	}
	backupExtensions = []string{
		".bak", ".backup", ".orig", ".old", ".tmp", ".temp",
		".swap", ".chklist", ".chk", ".ms", ".diz", ".wbk",
		".xlk", ".cdr_", ".nch", ".ftg", ".gid", ".syd",
		".bkp", ".gho", ".vhd", ".vhdx", ".tib", ".log",
		".sql", ".dump", ".sav", ".dbk", ".rdb",
	}

	// extTypeMap is built in init() for O(1) extension lookups.
	extTypeMap map[string]FileType

	ansiReset = "\033[0m"
	colorMap  = map[FileType]string{
		FileTypeDirectory:    "\033[94m",
		FileTypeExecutable:   "\033[32m",
		FileTypeSymbolicLink: "\033[96m",
		FileTypeArchive:      "\033[91m",
		FileTypeMedia:        "\033[95m",
		FileTypeBackup:       "\033[90m",
		FileTypeOther:        "",
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

// ─────────────────────────────────────────────
// init
// ─────────────────────────────────────────────

func init() {
	// Resolve current OS user.
	if u, err := user.Current(); err == nil {
		currentUser = u.Username
	}

	// Build extension → FileType map for O(1) lookups.
	extTypeMap = make(map[string]FileType)
	for _, e := range backupExtensions {
		extTypeMap[e] = FileTypeBackup
	}
	for _, e := range mediaExtensions {
		extTypeMap[e] = FileTypeMedia
	}
	for _, e := range archiveExtensions {
		extTypeMap[e] = FileTypeArchive
	}
	// Executable extensions are only used on Windows but we store them
	// unconditionally and gate the lookup at call-time.
	for _, e := range executableExtensions {
		if _, exists := extTypeMap[e]; !exists {
			extTypeMap[e] = FileTypeExecutable
		}
	}
}

// ─────────────────────────────────────────────
// Argument types
// ─────────────────────────────────────────────

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
	ShowAll      bool // -a: show hidden (dot) files
}

type FileInfoEx struct {
	fs.FileInfo
	Path      string
	Links     uint64
	OwnerName string
}

// ─────────────────────────────────────────────
// Terminal / display utilities
// ─────────────────────────────────────────────

func isOutputRedirected() bool {
	stat, err := os.Stdout.Stat()
	if err != nil {
		return true
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

func getTerminalWidth() int {
	// If output is redirected there is no terminal; use a very large value so
	// nothing gets truncated by column arithmetic.
	if isOutputRedirected() {
		return math.MaxInt
	}
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		return 80
	}
	return width
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
	padding := totalWidth - getStringDisplayWidth(s)
	if padding <= 0 {
		return s
	}
	return s + strings.Repeat(" ", padding)
}

func padLeftByWidth(s string, totalWidth int) string {
	padding := totalWidth - getStringDisplayWidth(s)
	if padding <= 0 {
		return s
	}
	return strings.Repeat(" ", padding) + s
}

// ─────────────────────────────────────────────
// Hyperlink / gradient helpers (cosmetic)
// ─────────────────────────────────────────────

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

// ─────────────────────────────────────────────
// Help text
// ─────────────────────────────────────────────

func getHelpText() string {
	startRGB := [3]int{0, 150, 255}
	endRGB := [3]int{50, 255, 50}
	gradientTitle := addGradient("Enhanced-ls v0.0.9 (Cross-Platform)", startRGB, endRGB)
	link := createHyperlink(gradientTitle, "https://github.com/Geekstrange/enhanced-ls")

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
    %s-a%s        show hidden files (entries starting with '.').
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
		blue, reset,
		cyan, reset,
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

// ─────────────────────────────────────────────
// Argument parsing
// ─────────────────────────────────────────────

func parseArgs(args []string) (*LSArgs, error) {
	lsArgs := &LSArgs{Path: "."}

	validOptions := "faclrSsSh"

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

			// Enforce mutual exclusivity of -s and -S.
			hasS := strings.ContainsRune(options, 'S')
			hass := strings.ContainsRune(options, 's')
			if hasS && hass {
				return nil, fmt.Errorf("-s (case-insensitive) and -S (case-sensitive) are mutually exclusive")
			}

			if hasS {
				lsArgs.StrictCase = true
				if i < len(args)-1 && !strings.HasPrefix(args[i+1], "-") {
					i++
					lsArgs.SearchTerm = args[i]
				}
			} else if hass {
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
							if len(next) == 1 && strings.ContainsAny(next, ValidTypeIndicators) {
								lsArgs.FilterType = next
								i++
							}
						}
					case 'c':
						lsArgs.SetColor = true
					case 'r':
						lsArgs.Recursive = true
					case 'a':
						lsArgs.ShowAll = true
					}
				}
			}
		} else {
			// Positional path argument.
			if lsArgs.Path == "." {
				lsArgs.Path = arg
			} else {
				fmt.Fprintf(os.Stderr, "Warning: multiple paths not supported, using first path: %s\n", lsArgs.Path)
			}
		}
		i++
	}

	return lsArgs, nil
}

// ─────────────────────────────────────────────
// File-type detection
// ─────────────────────────────────────────────

// isSymbolicLink uses Lstat so it works correctly on all platforms,
// including when the link target does not exist.
func isSymbolicLink(path string) bool {
	fi, err := os.Lstat(path)
	return err == nil && fi.Mode()&os.ModeSymlink != 0
}

func getFileType(info fs.FileInfo, path string) FileType {
	// Symbolic link check MUST come before directory check because a symlink
	// to a directory would otherwise report as a directory.
	if isSymbolicLink(path) {
		return FileTypeSymbolicLink
	}

	if info.IsDir() {
		return FileTypeDirectory
	}

	// On non-Windows systems honour the executable permission bits.
	if runtime.GOOS != "windows" {
		if info.Mode()&0111 != 0 {
			return FileTypeExecutable
		}
	}

	ext := strings.ToLower(filepath.Ext(info.Name()))

	// O(1) map lookup instead of iterating three slices.
	if ft, ok := extTypeMap[ext]; ok {
		// On non-Windows systems we only return FileTypeExecutable from the
		// extension map when running on Windows.
		if ft == FileTypeExecutable && runtime.GOOS != "windows" {
			return FileTypeOther
		}
		return ft
	}

	return FileTypeOther
}

// ─────────────────────────────────────────────
// Filtering
// ─────────────────────────────────────────────

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
		if typeIndicators[fileType] != args.FilterType {
			return false
		}
	}

	return true
}

// ─────────────────────────────────────────────
// Tree-entry formatting helper (DRY)
// ─────────────────────────────────────────────

// formatTreeEntry computes the display name for a single directory entry in
// tree mode and reports whether the entry should be skipped.
func formatTreeEntry(entry fs.DirEntry, fullPath string, args *LSArgs) (displayName string, isDir bool, skip bool) {
	// Hidden-file filtering.
	if !args.ShowAll && strings.HasPrefix(entry.Name(), ".") {
		return "", false, true
	}

	// Prefer DirEntry.Info() to avoid a redundant Lstat call; fall back to
	// Lstat only for symbolic links so we get accurate link information.
	var info fs.FileInfo
	var err error
	if entry.Type()&os.ModeSymlink != 0 {
		info, err = os.Lstat(fullPath)
	} else {
		info, err = entry.Info()
	}
	if err != nil {
		return "", false, true
	}

	isDir = entry.IsDir()
	name := entry.Name()
	fileType := getFileType(info, fullPath)

	if !passesFilter(name, fileType, args) {
		return "", isDir, true
	}

	if args.ShowFileType {
		name += typeIndicators[fileType]
	}

	if !isOutputRedirected() && args.SetColor {
		displayName = colorMap[fileType] + name + ansiReset
	} else {
		displayName = name
	}
	return displayName, isDir, false
}

// ─────────────────────────────────────────────
// Tree display (unified – no duplicate printing)
// ─────────────────────────────────────────────

// displayTree prints the directory tree rooted at path in the style of the
// standard `tree` command.  The root is always printed by the caller (main).
// Each node is printed exactly once: by its parent when iterating children.
func displayTree(path string, args *LSArgs, prefix string) {
	entries, err := os.ReadDir(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: cannot read directory %s: %v\n", path, err)
		return
	}

	// Collect visible, filtered entries.
	var visible []fs.DirEntry
	for _, entry := range entries {
		if !args.ShowAll && strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		visible = append(visible, entry)
	}

	sort.Slice(visible, func(i, j int) bool {
		return strings.ToLower(visible[i].Name()) < strings.ToLower(visible[j].Name())
	})

	for i, entry := range visible {
		fullPath := filepath.Join(path, entry.Name())

		isLast := i == len(visible)-1
		connector := "├── "
		newPrefix := prefix + "│   "
		if isLast {
			connector = "└── "
			newPrefix = prefix + "    "
		}

		displayName, isDir, skip := formatTreeEntry(entry, fullPath, args)
		if skip {
			continue
		}

		fmt.Printf("%s%s%s\n", prefix, connector, displayName)

		if isDir {
			displayTree(fullPath, args, newPrefix)
		}
	}
}

// ─────────────────────────────────────────────
// Layout / column calculation
// ─────────────────────────────────────────────

func calculateLayout(displayWidths []int, windowWidth int) (rows, cols int, colWidths []int) {
	if len(displayWidths) == 0 {
		return 0, 0, nil
	}

	padding := spaceLength
	cols = 1
	colWidths = []int{maxIntSlice(displayWidths)}

	calcWidth := func(widths []int, pad, ncols int) (int, []int) {
		maxWidths := make([]int, ncols)
		perLine := len(widths) / ncols
		if len(widths)%ncols != 0 {
			perLine++
		}
		for col := 0; col < ncols; col++ {
			start := col * perLine
			end := minInt(start+perLine, len(widths))
			for j := start; j < end; j++ {
				if widths[j] > maxWidths[col] {
					maxWidths[col] = widths[j]
				}
			}
		}
		sum := 0
		for _, w := range maxWidths {
			sum += w
		}
		return sum + (ncols-1)*pad, maxWidths
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

	rows = len(displayWidths) / cols
	if len(displayWidths)%cols != 0 {
		rows++
	}
	return rows, cols, colWidths
}

func maxIntSlice(nums []int) int {
	if len(nums) == 0 {
		return 0
	}
	m := nums[0]
	for _, n := range nums[1:] {
		if n > m {
			m = n
		}
	}
	return m
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

// ─────────────────────────────────────────────
// File metadata helpers
// ─────────────────────────────────────────────

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
	return fmt.Sprintf("%.1f%c", float64(size)/float64(div), "KMGTPE"[exp])
}

// ─────────────────────────────────────────────
// Display: flat / column layout
// ─────────────────────────────────────────────

func displayItems(items []FileInfoEx, args *LSArgs) {
	if len(items) == 0 {
		fmt.Println("No matching files found")
		return
	}

	displayNames := make([]string, len(items))
	displayWidths := make([]int, len(items))

	for i, item := range items {
		fileType := getFileType(item.FileInfo, item.Path)
		baseName := item.Name()

		if args.ShowFileType {
			baseName += typeIndicators[fileType]
		}

		if !isOutputRedirected() && args.SetColor {
			displayNames[i] = colorMap[fileType] + baseName + ansiReset
		} else {
			displayNames[i] = baseName
		}
		displayWidths[i] = getStringDisplayWidth(baseName)
	}

	windowWidth := getTerminalWidth()
	rows, _, colWidths := calculateLayout(displayWidths, windowWidth)

	lines := make([][]string, rows)

	for idx, name := range displayNames {
		row := idx % rows
		col := idx / rows
		if col >= len(colWidths) {
			continue
		}
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

// ─────────────────────────────────────────────
// Display: long / table format
// ─────────────────────────────────────────────

func displayLongFormat(items []FileInfoEx, args *LSArgs) {
	if len(items) == 0 {
		fmt.Println("No matching files found")
		return
	}

	// Whether to show the Links column (meaningless on Windows).
	showLinks := runtime.GOOS != "windows"

	modeWidth := 4
	linksWidth := 5
	ownerWidth := 5
	sizeWidth := 4
	timeWidth := 15
	nameWidth := 4

	// Pre-compute formatted values to avoid duplicate calls.
	type rowData struct {
		mode      string
		links     string
		owner     string
		size      string
		timeStr   string
		baseName  string
		fileType  FileType
	}

	rows := make([]rowData, len(items))
	for i, item := range items {
		ft := getFileType(item.FileInfo, item.Path)
		bn := item.Name()
		if args.ShowFileType {
			bn += typeIndicators[ft]
		}
		ts := item.ModTime().Format("2006/01/02 15:04")

		rows[i] = rowData{
			mode:     item.Mode().String(),
			links:    strconv.FormatUint(item.Links, 10),
			owner:    item.OwnerName,
			size:     formatSize(item.Size()),
			timeStr:  ts,
			baseName: bn,
			fileType: ft,
		}

		if w := len(rows[i].mode); w > modeWidth {
			modeWidth = w
		}
		if showLinks {
			if w := len(rows[i].links); w > linksWidth {
				linksWidth = w
			}
		}
		if w := getStringDisplayWidth(rows[i].owner); w > ownerWidth {
			ownerWidth = w
		}
		if w := len(rows[i].size); w > sizeWidth {
			sizeWidth = w
		}
		if w := getStringDisplayWidth(rows[i].baseName); w > nameWidth {
			nameWidth = w
		}
	}

	modeWidth = maxInt(modeWidth, 4)
	linksWidth = maxInt(linksWidth, 5)
	ownerWidth = maxInt(ownerWidth, 5)
	sizeWidth = maxInt(sizeWidth, 4)
	timeWidth = maxInt(timeWidth, 15)
	nameWidth = maxInt(nameWidth, 4)

	// Helper to build a border row.
	border := func(left, mid, right, h string) string {
		parts := []string{
			strings.Repeat(h, modeWidth),
			strings.Repeat(h, ownerWidth),
			strings.Repeat(h, sizeWidth),
			strings.Repeat(h, timeWidth),
			strings.Repeat(h, nameWidth),
		}
		if showLinks {
			// Insert links column after mode.
			parts = append([]string{parts[0], strings.Repeat(h, linksWidth)}, parts[1:]...)
		}
		return left + strings.Join(parts, mid) + right
	}

	topLine := border("┌", "┬", "┐", "─")
	divider := border("├", "┼", "┤", "─")
	bottomLine := border("└", "┴", "┘", "─")

	// Header row.
	headerFields := []string{
		padByWidth("Mode", modeWidth),
		padByWidth("Owner", ownerWidth),
		padByWidth("Size", sizeWidth),
		padByWidth("Change Time", timeWidth),
		padByWidth("Name", nameWidth),
	}
	if showLinks {
		headerFields = append([]string{headerFields[0], padByWidth("Links", linksWidth)}, headerFields[1:]...)
	}
	header := "│" + strings.Join(headerFields, "│") + "│"

	fmt.Println(topLine)
	fmt.Println(header)
	fmt.Println(divider)

	for _, rd := range rows {
		mode := padByWidth(rd.mode, modeWidth)
		owner := padByWidth(rd.owner, ownerWidth)
		size := padLeftByWidth(rd.size, sizeWidth)

		// Truncate time with ellipsis indicator if needed.
		ts := rd.timeStr
		const ellipsis = "…"
		if len(ts) > timeWidth {
			// Leave room for the ellipsis (1 rune = 3 UTF-8 bytes, but 1 display cell).
			ts = ts[:timeWidth-1] + ellipsis
		} else {
			ts = padByWidth(ts, timeWidth)
		}

		currentWidth := getStringDisplayWidth(rd.baseName)
		paddingSpaces := maxInt(0, nameWidth-currentWidth)

		var nameField string
		if !isOutputRedirected() && args.SetColor {
			nameField = colorMap[rd.fileType] + rd.baseName + ansiReset + strings.Repeat(" ", paddingSpaces)
		} else {
			nameField = rd.baseName + strings.Repeat(" ", paddingSpaces)
		}

		fields := []string{mode, owner, size, ts, nameField}
		if showLinks {
			fields = append([]string{fields[0], padLeftByWidth(rd.links, linksWidth)}, fields[1:]...)
		}
		fmt.Println("│" + strings.Join(fields, "│") + "│")
	}

	fmt.Println(bottomLine)
}

// ─────────────────────────────────────────────
// main
// ─────────────────────────────────────────────

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

	// filepath.Clean already normalises separators on every platform,
	// including Windows — do NOT do an additional ReplaceAll here as it
	// would corrupt UNC paths (\\server\share).
	args.Path = filepath.Clean(args.Path)

	fileInfo, err := os.Stat(args.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error accessing path: %v\n", err)
		os.Exit(1)
	}

	// ── Recursive / tree mode ──────────────────────────────────────────
	if args.Recursive {
		rootName := fileInfo.Name()
		rootType := getFileType(fileInfo, args.Path)

		// Always print the root directory header.
		var rootDisplay string
		if !isOutputRedirected() && args.SetColor {
			rootDisplay = colorMap[rootType] + rootName + ansiReset
		} else {
			rootDisplay = rootName
		}
		if args.ShowFileType {
			rootDisplay += typeIndicators[rootType]
		}
		fmt.Println(rootDisplay)

		// Only recurse into it when it passes the filter (or there is no
		// filter, meaning all roots are valid).
		if passesFilter(rootName, rootType, args) {
			displayTree(args.Path, args, "")
		}
		return
	}

	// ── Non-recursive mode ────────────────────────────────────────────
	var items []FileInfoEx

	if fileInfo.IsDir() {
		entries, err := os.ReadDir(args.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading directory: %v\n", err)
			os.Exit(1)
		}

		for _, entry := range entries {
			// Hidden-file filtering.
			if !args.ShowAll && strings.HasPrefix(entry.Name(), ".") {
				continue
			}

			fullPath := filepath.Join(args.Path, entry.Name())

			// Use DirEntry.Info() to avoid an extra syscall; Lstat only for symlinks.
			var info fs.FileInfo
			if entry.Type()&os.ModeSymlink != 0 {
				info, err = os.Lstat(fullPath)
			} else {
				info, err = entry.Info()
			}
			if err != nil {
				continue
			}

			fileType := getFileType(info, fullPath)
			if !passesFilter(entry.Name(), fileType, args) {
				continue
			}

			items = append(items, FileInfoEx{
				FileInfo:  info,
				Path:      fullPath,
				Links:     getLinkCount(info),
				OwnerName: currentUser,
			})
		}
	} else {
		// Single-file argument.
		info, err := os.Lstat(args.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error accessing file: %v\n", err)
			os.Exit(1)
		}
		items = append(items, FileInfoEx{
			FileInfo:  info,
			Path:      args.Path,
			Links:     getLinkCount(info),
			OwnerName: currentUser,
		})
	}

	// Sort entries.
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
