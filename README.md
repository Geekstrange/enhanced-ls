# enhanced-ls

This is a cross-platform enhanced "ls" command-line tool designed to provide users with a richer and more convenient experience for viewing file and folder listings.

## Features

- 🎨 **Colorized output**: Directories, executables, and symbolic links are displayed in different colors
- 📝 **File type indicators**: Append `/` (directory), `*` (executable), or `@` (symbolic link) after file names
- 📊 **Multi-column layout**: Automatically adapts to terminal width for multi-column display
- 🖥️ **Detailed mode**: Use the `-l` option to display a table layout
- 📏 **CJK character support**: Correctly handles width calculation for Chinese, Japanese, and Korean characters
- 🚀 **Lightweight and efficient**: Implemented in Golang with no external dependencies

## Requirements

1. [PowerShell 7.2+](https://github.com/PowerShell/PowerShell/releases) is recommended
2. Modern terminals such as [Windows Terminal](https://github.com/microsoft/terminal/releases) / [Tabby](https://tabby.sh/) / [Fluent Terminal](https://github.com/felixse/FluentTerminal/releases) are recommended

## Installation

1. Clone or download the project locally:
   ```bash
   git clone https://github.com/Geekstrange/enhanced-ls.git
   ```

2. Add the following content to your PowerShell profile (`$PROFILE`):
   ```bash
   # Remove the existing ls alias
   Remove-Item Alias:ls -ErrorAction SilentlyContinue

   # Set the ls alias to enls.exe
   function Invoke-Ls {
       \path\to\enls.exe -c @args  # @args passes through all arguments
   }
   Set-Alias ls Invoke-Ls
   ```

3. Reload the profile:
   ```bash
   .$PROFILE
   ```

## Usage

*Demonstrated using PowerShell 7.5+ on Windows*

### Basic command

```bash
ls [path] [options]
```

### Options

| Option       | Description                         |
| ---------- | ---------------------------- |
| `-f` or `-F` | Show file type indicators (`*/@#~%`) **or** filter files of a specified type (e.g., `-f "#"` shows only compressed files) |
| `-c` or `-C` | Enable colorized output                 |
| `-l` or `-L` | Detailed list mode |
| `-s` | Case-insensitive query |
| `-S` | Strict case-sensitive query |
| `-r` | Recursive display |
| `-1` | Single-column display (one entry per line) |
| `-t` | Sort by modification time (newest first) |
| `-d` | List only the directories themselves (do not expand contents) |
| `--json` | Output machine-readable JSON (for scripts/agents) |
| `--reverse` | Reverse sorting |
| `--version` | Show version number |
| `--help`   | Show help information                 |

> Supports multiple path arguments and wildcards: `enls *.go`, `enls src bin` (globs not expanded by the shell will be expanded by enls itself)

### Examples

1. **Basic usage** (multi-column layout, automatically adapts to terminal width):

   ```bash
   ls
   ```

   ![ls](https://github.com/Geekstrange/enhanced-ls-for-powershell/blob/main/image/ls.png)

2. **Colorized output**:

   ```bash
   ls -c
   ```

   ![ls-c](https://github.com/Geekstrange/enhanced-ls-for-powershell/blob/main/image/lsc.png)

3. **Show file type indicators**:

   ```bash
   ls -f
   ```

   ![ls-f](https://github.com/Geekstrange/enhanced-ls-for-powershell/blob/main/image/lsf.png)

5. **Combined options** (color + file type + table indicators):

   ```bash
   ls -c -f -l or ls -cfl
   ```

   ![ls-cfl](https://github.com/Geekstrange/enhanced-ls-for-powershell/blob/main/image/lslcf.png)

6. **Specify a path**:

   ```bash
   ls C:\Users
   ls -l D:\Projects
   ```

7. **Recursive display**

   ```bash
   ls -s "r"
   ```

   ![ls-r](https://github.com/Geekstrange/enhanced-ls-for-powershell/blob/main/image/lsr.png)

8. **Strict case-sensitive query**

   ```bash
   ls -S "R" -l
   ```

   ![ls-S-l](https://github.com/Geekstrange/enhanced-ls-for-powershell/blob/main/image/lssl.png)

## Version Selection Guide

Choose the corresponding installation file based on your operating system and architecture. The following table shows the version mapping for different platforms:

| **Operating System** | **Architecture**              | **File Name**                      |
| :----------- | :-------------------- | :------------------------------ |
| **Windows**  | x86_64/AMD64          | `enls-vx.x.x-windows_amd64.exe` |
|              | ARM64/AArch64         | `enls-vx.x.x-windows_arm64.exe` |
| **Linux**    | x86_64/AMD64          | `enls-vx.x.x-linux_amd64`       |
|              | ARM64/AArch64         | `enls-vx.x.x-linux_arm64`       |
|              | LoongArch         | `enls-vx.x.x-linux_loong64`       |
| **macOS**    | Intel (x86_64/AMD64)        | `enls-vx.x.x-darwin_amd64`      |
|              | Apple Silicon (ARM64/AArch64) | `enls-vx.x.x-darwin_arm64`      |

## How to Determine My System Architecture

### Windows

1. **Open Command Prompt**: You can open it by searching for "cmd" or "Command Prompt" in the Start menu

2. **Enter the command**: Type the following command in Command Prompt and press Enter:

   ```cmd
   wmic os get osarchitecture
   ```

3. **Check the output**:

   - If it shows "64-bit", your system is **64-bit (x86_64)**
   - If it shows "ARM64", your system is **ARM64**

### Linux

1. **Open Terminal**: You can open it by searching for "Terminal" in the application menu or using the shortcut (usually `Ctrl+Alt+T`)

2. **Enter the command**: Type the following command in the terminal and press Enter:

   ```bash
   uname -m
   ```

3. **Check the output**:

   - If it shows `x86_64`, your system is **64-bit (x86_64)**
   - If it shows `aarch64`, your system is **ARM64**

### macOS

1. **Open Terminal**: You can open it by searching for "Terminal"

2. **Enter the command**: Type the following command in the terminal and press Enter:

   ```zsh
   uname -m
   ```

3. **Check the output**:

   - If it shows `x86_64`, your system is **Intel (x86_64)**
   - If it shows `arm64`, your system is **Apple Silicon (ARM64)**


## License

This project is licensed under the [MIT License](LICENSE).

---