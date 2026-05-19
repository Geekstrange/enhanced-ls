# enhanced-ls

这是一个跨平台的增强版“ls” 命令工具，旨在为用户提供更丰富、更便捷的文件和文件夹列表查看体验

## 功能特点

- 🎨 **彩色输出**：目录、可执行文件和符号链接使用不同颜色显示
- 📝 **文件类型指示符**：在文件名后添加 `/`（目录）、`*`（可执行文件）或 `@`（符号链接）
- 📊 **多列布局**：自动适应终端宽度进行多列显示
- 🖥️ **详细模式**：使用 `-l` 选项显示表格布局
- 📏 **CJK字符支持**：正确处理中文、日文、韩文字符的宽度计算
- 🚀 **轻量高效**：Golang实现，无需外部依赖

## 环境要求

1. 建议使用[PowerShell 7.2+](https://github.com/PowerShell/PowerShell/releases)
2. 建议使用[Windows Terminal](https://github.com/microsoft/terminal/releases) / [Tabby](https://tabby.sh/) / [Fluent Terminal](https://github.com/felixse/FluentTerminal/releases) 等现代终端

## 安装

1. 将项目克隆或下载到本地：
   ```bash
   git clone https://github.com/Geekstrange/enhanced-ls.git
   ```

2. 在 PowerShell 配置文件 (`$PROFILE`) 中添加以下内容：
   ```bash
   # 移除现有的 ls 别名
   Remove-Item Alias:ls -ErrorAction SilentlyContinue

   # 设置 ls 别名指向enls.exe
   function Invoke-Ls {
       \path\to\enls.exe -c @args  # @args 表示透传所有参数
   }
   Set-Alias ls Invoke-Ls
   ```

3. 重新加载配置文件：
   ```bash
   .$PROFILE
   ```

## 使用说明

*使用Windows下的PowerShell 7.5+演示*

### 基本命令

```bash
ls [路径] [选项]
```

### 选项

| 选项       | 描述                         |
| ---------- | ---------------------------- |
| `-f`或`-F` | 显示文件类型指示符(`*/@#~%`) **或** 筛选指定类型文件（如`-f "#"`仅显示压缩文件） |
| `-c`或`-C` | 启用彩色输出                 |
| `-l`或`-L` | 详细列表模式 |
| `-s` | 忽略大小写查询 |
| `-S` | 严格匹配大小写查询 |
| `-r` | 递归显示 |
| `--help`   | 显示帮助信息                 |

### 示例

1. **基本使用**（多列布局，自动适应终端宽度）：

   ```bash
   ls
   ```

   ![ls](https://github.com/Geekstrange/enhanced-ls-for-powershell/blob/main/image/ls.png)

2. **彩色输出**：

   ```bash
   ls -c
   ```

   ![ls-c](https://github.com/Geekstrange/enhanced-ls-for-powershell/blob/main/image/lsc.png)

3. **显示文件类型指示符**：

   ```bash
   ls -f
   ```

   ![ls-f](https://github.com/Geekstrange/enhanced-ls-for-powershell/blob/main/image/lsf.png)

5. **组合选项**（彩色+文件类型+表格指示符）：

   ```bash
   ls -c -f -l或ls -cfl
   ```

   ![ls-cfl](https://github.com/Geekstrange/enhanced-ls-for-powershell/blob/main/image/lslcf.png)

6. **指定路径**：

   ```bash
   ls C:\Users
   ls -l D:\Projects
   ```

7. **递归显示**

   ```bash
   ls -s "r"
   ```

   ![ls-r](https://github.com/Geekstrange/enhanced-ls-for-powershell/blob/main/image/lsr.png)

8. **严格匹配大小写查询**

   ```bash
   ls -S "R" -l
   ```

   ![ls-S-l](https://github.com/Geekstrange/enhanced-ls-for-powershell/blob/main/image/lssl.png)

## 版本选择指南

根据您的操作系统和架构，请选择对应的安装文件以下是不同平台的版本对应关系：

| **操作系统** | **架构**              | **文件名**                      |
| :----------- | :-------------------- | :------------------------------ |
| **Windows**  | x86_64/AMD64          | `enls-vx.x.x-windows_amd64.exe` |
|              | ARM64/AArch64         | `enls-vx.x.x-windows_arm64.exe` |
| **Linux**    | x86_64/AMD64          | `enls-vx.x.x-linux_amd64`       |
|              | ARM64/AArch64         | `enls-vx.x.x-linux_arm64`       |
|              | LoongArch         | `enls-vx.x.x-linux_loong64`       |
| **macOS**    | Intel (x86_64/AMD64)        | `enls-vx.x.x-darwin_amd64`      |
|              | Apple Silicon (ARM64/AArch64) | `enls-vx.x.x-darwin_arm64`      |

## 如何确定我的系统架构

### Windows 系统

1. **打开命令提示符**：可以通过在开始菜单中搜索“cmd”或“命令提示符”来打开

2. **输入命令**：在命令提示符中输入以下命令并按回车键：

   ```cmd
   wmic os get osarchitecture
   ```

3. **查看输出结果**：

   - 如果显示“64-bit”，则您的系统是 **64位 (x86_64)**
   - 如果显示“ARM64”，则您的系统是 **ARM64**

### Linux 系统

1. **打开终端**：可以通过在应用程序菜单中搜索“终端”或使用快捷键（通常是`Ctrl+Alt+T`）来打开

2. **输入命令**：在终端中输入以下命令并按回车键：

   ```bash
   uname -m
   ```

3. **查看输出结果**：

   - 如果显示`x86_64`，则您的系统是 **64位 (x86_64)**
   - 如果显示`aarch64`，则您的系统是 **ARM64**

### macOS 系统

1. **打开终端**：可以通过在应用程序菜单中搜索“终端”来打开

2. **输入命令**：在终端中输入以下命令并按回车键：

   ```zsh
   uname -m
   ```

3. **查看输出结果**：

   - 如果显示`x86_64`，则您的系统是 **Intel (x86_64)**
   - 如果显示`arm64`，则您的系统是 **Apple Silicon (ARM64)**


## 许可证

本项目采用 [MIT 许可证](LICENSE)

---
