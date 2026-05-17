# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`enhanced-ls` is a cross-platform enhanced `ls` command (`enls`) written in Go. Single-binary CLI tool with colored output, file type indicators, CJK-aware column layout, long format with table borders, tree view, and search/filter capabilities.

## Build & Run

```bash
# Build for current platform
go build -o enls main.go

# Cross-compile all platforms (outputs to release_bin/)
bash build.sh              # default version "bate"
bash build.sh -v 1.0.0    # with version tag

# Run directly
go run main.go [path] [options]
```

Target platforms: windows/amd64, windows/arm64, linux/amd64, linux/arm64, linux/loong64, darwin/amd64, darwin/arm64.

## Architecture

Everything lives in a single `main.go` (~965 lines). No packages, no tests.

Key structures:
- `LSArgs` — parsed CLI arguments (path, flags, search term, filter type)
- `FileInfoEx` — extends `fs.FileInfo` with path, link count, owner
- `FileType` enum — Directory, Executable, SymbolicLink, Archive, Media, Backup, Other

Flow: `main()` → `parseArgs()` → either `displayTree()` (recursive) or `displayItems()`/`displayLongFormat()` (flat listing).

Display modes:
- Default: multi-column layout auto-fitted to terminal width via `calculateLayout()`
- `-l`: bordered table with Mode/Links/Owner/Size/Time/Name columns
- `-r`: tree view with `├──`/`└──` connectors

File type detection (`getFileType`): checks symlink first, then directory, then executable permission (Unix) or extension (Windows), then extension-based classification for archives/media/backup.

## CLI Flags

`-f` file type indicators, `-c` color, `-l` long format, `-r` recursive tree, `-s` case-insensitive search, `-S` case-sensitive search, `-h` help. Flags can be combined (e.g. `-cfl`).

## Dependencies

Only external dependency: `golang.org/x/term` (terminal size detection). Go 1.24.4+.

## Notes

- Output is ANSI-color aware and disables colors/hyperlinks when stdout is redirected (`isOutputRedirected()`)
- CJK character width handling throughout (`isCJK`, `getStringDisplayWidth`, `padByWidth`)
- Hidden files (dot-prefixed) are excluded in recursive/tree mode
- The module path in go.mod is `main.go` (non-standard but functional)
