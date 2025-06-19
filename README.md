# enhanced-ls-for-powershell

一个为 PowerShell 提供类 Linux `ls` 命令功能的模块，支持彩色输出、文件类型指示符和多列布局。

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
   ```powershell
   git clone https://github.com/Geekstrange/enhanced-ls-for-powershell.git 
   ```

2. 在 PowerShell 配置文件 (`$PROFILE`) 中添加以下内容：
   ```powershell
   # 移除现有的 ls 别名
   Remove-Item Alias:ls -ErrorAction SilentlyContinue
   
   # 设置 ls 别名指向enls.exe
   function Invoke-Ls {
       \path\to\enls.exe -c @args  # @args 表示透传所有参数
   }
   Set-Alias ls Invoke-Ls
   ```

3. 重新加载配置文件：
   ```powershell
   .$PROFILE
   ```

## 使用说明

### 基本命令

```powershell
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
| `--help`   | 显示帮助信息                 |

### 示例

1. **基本使用**（多列布局，自动适应终端宽度）：

   ```powershell
   ls
   ```

   ![ls](https://github.com/Geekstrange/enhanced-ls-for-powershell/blob/main/image/ls.png)

2. **彩色输出**：

   ```powershell
   ls -c
   ```

   ![ls-c](https://github.com/Geekstrange/enhanced-ls-for-powershell/blob/main/image/lsc.png)

3. **显示文件类型指示符**：

   ```powershell
   ls -f
   ```

   ![ls-f](https://github.com/Geekstrange/enhanced-ls-for-powershell/blob/main/image/lsf.png)

5. **组合选项**（彩色+文件类型+表格指示符）：

   ```powershell
   ls -c -f -l或ls -cfl
   ```

   ![ls-cfl](https://github.com/Geekstrange/enhanced-ls-for-powershell/blob/main/image/lslcf.png)

6. **指定路径**：

   ```powershell
   ls C:\Users
   ls -l D:\Projects
   ```

7. **忽略大小写查询**

   ```powershell
   ls -s "r"
   ```

   ![ls-s](https://github.com/Geekstrange/enhanced-ls-for-powershell/blob/main/image/lss.png)

8. **严格匹配大小写查询**

   ```powershell
   ls -S "R" -l
   ```

   ![ls-S-l](https://github.com/Geekstrange/enhanced-ls-for-powershell/blob/main/image/lssl.png)

## 许可证

本项目采用 [MIT 许可证](LICENSE)

---

**让 PowerShell 拥有 Linux 终端的体验！**  
现在就开始使用 `ls` 命令，享受更直观、更丰富的文件列表体验！
