#!/bin/bash

# =================================================================
# ES-SPECTRE 麒麟 V10 图形版自动部署/打包脚本
# =================================================================

# 1. 检查权限
if [ "$EUID" -ne 0 ]; then 
  echo "请使用 sudo 运行此脚本以安装依赖库"
  exit
fi

echo "🚀 [1/4] 正在安装系统级编译依赖 (GTK3 & WebKitGtk)..."
# 麒麟系统通常基于 Ubuntu/Debian 架构
apt-get update
apt-get install -y build-essential libgtk-3-dev libwebkit2gtk-4.0-dev \
                   pkg-config curl git nodejs npm

echo "🚀 [2/4] 检查 Go 语言环境..."
if ! command -v go &> /dev/null; then
    echo "未检测到 Go 环境，建议手动安装 Go 1.21+。退出打包。"
    exit
fi

echo "🚀 [3/4] 安装 Wails CLI 工具..."
go install github.com/wailsapp/wails/v2/cmd/wails@latest
export PATH=$PATH:$(go env GOPATH)/bin

echo "🚀 [4/4] 开始执行本地打包..."
cd gui
wails build -o es-spectre-gui-kylin -clean

if [ $? -eq 0 ]; then
    echo "✅ 打包成功！"
    echo "输出目录: $(pwd)/build/bin/es-spectre-gui-kylin"
    echo "使用命令运行: ./build/bin/es-spectre-gui-kylin"
else
    echo "❌ 打包失败，请检查上方日志。"
fi
