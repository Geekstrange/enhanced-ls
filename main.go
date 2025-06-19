func getHelpText() string {
	startRGB := [3]int{0, 150, 255}
	endRGB := [3]int{50, 255, 50}
	gradientTitle := addGradient("Enhanced-ls v0.01", startRGB, endRGB)
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
		"\033", ansiReset,
		"\033", ansiReset,
		"\033", ansiReset,
		"\033", ansiReset,
		"\033", ansiReset,
	)
}
