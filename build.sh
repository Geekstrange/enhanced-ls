#!/bin/bash
# Copyright (c) 2025 Shuaibo Zhang
# Created Time: 2025-07-07 13:16:34

NAME="enls"
DEFAULT_VERSION="v0.1.0"
OUTPUT_DIR="release_bin"

# 解析命令行参数
while getopts "v:" opt; do
  case $opt in
    v)
      # 如果版本号不以'v'开头,则添加'v'前缀
      if [[ "$OPTARG" != v* ]]; then
        VERSION="v$OPTARG"
      else
        VERSION="$OPTARG"
      fi
      ;;
    \?)
      echo "无效选项: -$OPTARG" >&2
      exit 1
      ;;
  esac
done

# 如果没有指定版本号,则使用默认值
if [ -z "$VERSION" ]; then
  VERSION="$DEFAULT_VERSION"
  # 默认版本号也统一加上 v 前缀
  if [[ "$VERSION" != v* ]]; then
    VERSION="v$VERSION"
  fi
fi

# 创建输出目录
mkdir -p "$OUTPUT_DIR"

# 目标平台列表
PLATFORMS=(
  "windows/amd64"
  "windows/arm64"
  "linux/amd64"
  "linux/arm64"
  "linux/loong64"
  "darwin/amd64"
  "darwin/arm64"
)

# 遍历所有平台进行编译并打包
for PLATFORM in "${PLATFORMS[@]}"; do
  # 分割平台配置
  GOOS=${PLATFORM%/*}
  GOARCH=${PLATFORM#*/}

  # 根据操作系统确定二进制文件名
  BINARY_NAME="$NAME"
  if [ "$GOOS" = "windows" ]; then
    BINARY_NAME="$NAME.exe"
  fi

  # 创建临时目录用于本次构建（避免多平台互相覆盖）
  TMP_DIR="$OUTPUT_DIR/.tmp_${GOOS}_${GOARCH}"
  mkdir -p "$TMP_DIR"

  # 编译二进制文件到临时目录
  printf "\033[32mCompiling\033[0m $GOOS/$GOARCH -> $TMP_DIR/$BINARY_NAME\n"
  env GOOS=$GOOS GOARCH=$GOARCH go build -o "$TMP_DIR/$BINARY_NAME" .

  # 确保非 Windows 平台的二进制文件具有可执行权限
  if [ "$GOOS" != "windows" ]; then
    chmod +x "$TMP_DIR/$BINARY_NAME"
  fi

  # 打包为 tar.gz，压缩包命名包含版本和平台信息
  TAR_NAME="$NAME-$VERSION-${GOOS}_${GOARCH}.tar.gz"
  printf "\033[32mPackaging\033[0m $TAR_NAME\n"
  # 进入临时目录，打包时仅包含二进制文件，不包含目录结构
  (cd "$TMP_DIR" && tar czf "../$TAR_NAME" "$BINARY_NAME")

  # 清理临时目录
  rm -rf "$TMP_DIR"
done

printf "\033[32mSuccess\033[0m\n"
printf "\033[32mVersion\033[0m $VERSION\n"
printf "\033[32mOutput Directory\033[0m: $OUTPUT_DIR\n"
tree "$OUTPUT_DIR"
