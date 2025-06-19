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

// FileType represents the type of file
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

// String returns the string representation of FileType
func (ft FileType) String() string {
	switch ft {
	case FileTypeDirectory:
		return "Directory"
	case FileTypeExecutable:
		return "Executable"
	case FileTypeSymbolicLink:
		return "SymbolicLink"
	case FileTypeArchive:
		return "Archive"
	case FileTypeMedia:
		return "Media"
	case FileTypeBackup:
		return "Backup"
	default:
		return "Other"
	}
}

// Configuration and constants
var (
	// Executable file extensions
	executableExtensions = []string{
		".exe", ".bat", ".cmd", ".ps1", ".sh",
		".js", ".py", ".rb", ".pl", ".cs", ".vbs",
	}

	// Archive file extensions
	archiveExtensions = []string{
		".7z", ".zip", ".rar", ".tar", ".gz", ".xz", ".bz2",
		".cab", ".img", ".iso", ".jar", ".pea", ".rpm", ".tgz", ".z", ".deb", ".arj", ".lzh",
		".lzma", ".lzma2", ".war", ".zst", ".part", ".s7z", ".split",
	}

	// Media file extensions
	mediaExtensions = []string{
		// Audio formats
		".aac", ".amr", ".caf", ".m3u", ".midi", ".mod", ".mp1", ".mp2", ".mp3", ".ogg", ".opus", ".ra", ".wma", ".wav", ".wv",
		// Video formats
		".3gp", ".3g2", ".asf", ".avi", ".flv", ".m4v", ".mkv", ".mov", ".mp4", ".mpeg", ".mpg", ".mpe", ".mts", ".rm", ".rmvb", ".swf", ".vob", ".webm", ".wmv",
		// Image formats
		".ai", ".avage", ".art", ".blend", ".cgm", ".cin", ".cur", ".cut", ".dcx", ".dng", ".dpx", ".emf", ".fit", ".fits", ".fpx", ".g3", ".hdr", ".ief", ".jbig", ".jfif", ".jls", ".jp2", ".jpc", ".jpx", ".jpg", ".jpeg", ".jxl", ".pbm", ".pcd", ".pcx", ".pgm", ".pict", ".png", ".pnm", ".ppm", ".psd", ".ras", ".rgb", ".svg", ".tga", ".tif", ".tiff", ".wbmp", ".xpm",
	}

	// Backup file extensions
	backupExtensions = []string{
		".bak", ".backup", ".orig", ".old", ".tmp", ".temp", ".swap",
		".chklist", ".chk", ".ms", ".diz", ".wbk", ".xlk", ".cdr_",
		".nch", ".ftg", ".gid", ".syd",
	}

	// ANSI color codes
	ansiReset = "\033[0m"
	colorMap  = map[FileType]string{
		FileTypeDirectory:    "\033[94m", // bright blue
		FileTypeExecutable:   "\033[32m", // green
		FileTypeSymbolicLink: "\033[96m", // bright cyan
		FileTypeArchive:      "\033[91m", // red
		FileTypeMedia:        "\033[95m", // purple
		FileTypeBackup:       "\033[90m", // gray
		FileTypeOther:        ansiReset,  // reset
	}

	// File type indicators
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

// LSArgs holds command line arguments
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

// addGradient creates a gradient colored text
func addGradient(text string, startRGB, endRGB [3]int) string {
	if isOutputRedirected() {
		return text
	}

	result := ""
	chars := []rune(text)
	for i, char := range chars {
		// Calculate color interpolation
		ratio := float64(i) / float64(len(chars)-1)
		if len(chars) == 1 {
			ratio = 0
		}
		
		r := int(float64(startRGB[0]) + (float64(endRGB[0])-float64(startRGB[0]))*ratio)
		g := int(float64(startRGB[1]) + (float64(endRGB[1])-float64(startRGB[1]))*ratio)
		b := int(float64(startRGB[2]) + (float64(endRGB[2])-float64(startRGB[2]))*ratio)

		// Generate ANSI true color sequence
		result += fmt.Sprintf("\033[38;2;%d;%d;%dm%c", r, g, b, char)
	}
	return result + ansiReset
}

// createHyperlink creates a hyperlink with ANSI escape sequences
func createHyperlink(text, url string) string {
	if isOutputRedirected() {
		return text
	}
	return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, text)
}

// getHelpText returns the help text
func getHelpText() string {
	startRGB := [3]int{0, 150, 255}
	endRGB := [3]int{50, 255, 50}
	gradientTitle := addGradient("Enhanced-ls for PowerShell v0.01", startRGB, endRGB)
	link := createHyperlink(gradientTitle, "https://github.com/Geekstrange/enhanced-ls-for-powershell")

	return fmt.Sprintf(`
        %s

%s[96mOptions:%s
    %s[32m-1%s      list one file per line.
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
		"\033", ansiReset,
	)
}

// isOutputRedirected checks if output is redirected
func isOutputRedirected() bool {
	return !term.IsTerminal(int(os.Stdout.Fd()))
}

// getStringDisplayWidth calculates display width (CJK characters count as 2)
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

// isCJK checks if a rune is a CJK character
func isCJK(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
		(r >= 0x3400 && r <= 0x4DBF) || // CJK Extension A
		(r >= 0x20000 && r <= 0x2A6DF) || // CJK Extension B
		(r >= 0x2A700 && r <= 0x2B73F) // CJK Extension C
}

// padByWidth pads string to specified display width
func padByWidth(s string, totalWidth int) string {
	currentWidth := getStringDisplayWidth(s)
	padding := totalWidth - currentWidth
	if padding <= 0 {
		return s
	}
	return s + strings.Repeat(" ", padding)
}

// getFileType determines the file type based on extension and attributes
func getFileType(info fs.FileInfo, path string) FileType {
	// Check if it's a symbolic link
	if info.Mode()&fs.ModeSymlink != 0 {
		return FileTypeSymbolicLink
	}

	// Check if it's a directory
	if info.IsDir() {
		return FileTypeDirectory
	}

	ext := strings.ToLower(filepath.Ext(info.Name()))

	// Check backup files first
	for _, backupExt := range backupExtensions {
		if ext == backupExt {
			return FileTypeBackup
		}
	}

	// Check media files
	for _, mediaExt := range mediaExtensions {
		if ext == mediaExt {
			return FileTypeMedia
		}
	}

	// Check archive files
	for _, archiveExt := range archiveExtensions {
		if ext == archiveExt {
			return FileTypeArchive
		}
	}

	// Check executable files
	for _, execExt := range executableExtensions {
		if ext == execExt {
			return FileTypeExecutable
		}
	}

	return FileTypeOther
}

// parseArgs parses command line arguments
func parseArgs(args []string) (*LSArgs, error) {
	lsArgs := &LSArgs{
		Path: ".",
	}

	validOptions := map[rune]bool{
		'f': true, 'c': true, 'l': true, 's': true, 'S': true, 'h': true, '1': true,
	}

	i := 0
	for i < len(args) {
		arg := args[i]

		if arg == "-h" {
			lsArgs.ShowHelp = true
			return lsArgs, nil
		}

		if strings.HasPrefix(arg, "-") {
			// Check for invalid options
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
						// Check if next argument is a filter type
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

// getTerminalWidth gets the terminal width
func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80 // default width
	}
	return width
}

// calculateLayout calculates the optimal layout for displaying items
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

// maxIntSlice returns the maximum value in a slice of integers
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

// maxInt returns the maximum of two integers
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// filterItems filters items based on search term and filter type
func filterItems(items []fs.FileInfo, paths []string, args *LSArgs) ([]fs.FileInfo, []string) {
	var filteredItems []fs.FileInfo
	var filteredPaths []string

	for i, item := range items {
		// Apply search filter
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

		// Apply type filter
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

// displayLongFormat displays items in long table format
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

	// Build table borders
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
		if !isOutputRedirected() && args.SetColor && fileType != FileTypeOther {
			color := colorMap[fileType]
			name = color + baseName + ansiReset + strings.Repeat(" ", paddingSpaces)
		} else {
			name = baseName + strings.Repeat(" ", paddingSpaces)
		}

		fmt.Printf("│%s│%s│%s│\n", mode, timeStr, name)
	}

	fmt.Println(bottomLine)
}

// displayItems displays items in column format
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
		if !isOutputRedirected() && args.SetColor && fileType != FileTypeOther {
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

	// Create lines array
	lines := make([][]string, rows)
	for i := range lines {
		lines[i] = make([]string, 0)
	}

	// Fill lines with items (column-wise)
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

	// Print lines
	space := strings.Repeat(" ", spaceLength)
	for _, line := range lines {
		fmt.Println(strings.Join(line, space))
	}
}

// main function
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

	// Read directory
	entries, err := os.ReadDir(args.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading directory: %v\n", err)
		os.Exit(1)
	}

	// Convert to FileInfo and collect paths
	var items []fs.FileInfo
	var paths []string
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		items = append(items, info)
		paths = append(paths, filepath.Join(args.Path, entry.Name()))
	}

	// Sort items by name
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name()) < strings.ToLower(items[j].Name())
	})

	// Apply filters
	items, paths = filterItems(items, paths, args)

	// Display results
	if args.LongFormat {
		displayLongFormat(items, paths, args)
	} else {
		displayItems(items, paths, args)
	}
}
