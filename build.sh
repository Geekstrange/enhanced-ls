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

if [ -z "$VERSION" ]; then
  VERSION="$DEFAULT_VERSION"
  if [[ "$VERSION" != v* ]]; then
    VERSION="v$VERSION"
  fi
fi

mkdir -p "$OUTPUT_DIR"

PLATFORMS=(
  "windows/amd64"
  "windows/arm64"
  "linux/amd64"
  "linux/arm64"
  "linux/loong64"
  "darwin/amd64"
  "darwin/arm64"
)

for PLATFORM in "${PLATFORMS[@]}"; do
  GOOS=${PLATFORM%/*}
  GOARCH=${PLATFORM#*/}

  BINARY_NAME="$NAME"
  if [ "$GOOS" = "windows" ]; then
    BINARY_NAME="$NAME.exe"
  fi

  TMP_DIR="$OUTPUT_DIR/.tmp_${GOOS}_${GOARCH}"
  mkdir -p "$TMP_DIR"

  printf "\033[32mCompiling\033[0m $GOOS/$GOARCH -> $TMP_DIR/$BINARY_NAME\n"
  env GOOS=$GOOS GOARCH=$GOARCH go build -o "$TMP_DIR/$BINARY_NAME" .

  if [ "$GOOS" != "windows" ]; then
    chmod +x "$TMP_DIR/$BINARY_NAME"
  fi

  TAR_NAME="$NAME-$VERSION-${GOOS}_${GOARCH}.tar.gz"
  printf "\033[32mPackaging\033[0m $TAR_NAME\n"
  # 关键修复：强制设置归档内文件的权限为 0755
  (cd "$TMP_DIR" && tar --mode=0755 -czf "../$TAR_NAME" "$BINARY_NAME")

  rm -rf "$TMP_DIR"
done

printf "\033[32mSuccess\033[0m\n"
printf "\033[32mVersion\033[0m $VERSION\n"
printf "\033[32mOutput Directory\033[0m: $OUTPUT_DIR\n"
tree "$OUTPUT_DIR"