#!/bin/bash
set -e

echo "🔨 Building frontend..."
cd webui
npm run build
cd ..

echo ""
echo "📦 Building backend with embedded web UI..."
go build -o socialpilot

echo ""
echo "✅ Build complete!"
echo ""
ls -lh socialpilot
echo ""
echo "📝 使用说明："
echo "  - 配置文件：./config.json (首次保存设置时自动创建)"
echo "  - 数据库文件：./socialpilot.db (首次添加联系人时自动创建)"
echo "  - 所有数据文件都在二进制文件同目录下"
echo ""
echo "🚀 启动服务："
echo "  ./socialpilot web"
echo "  ./socialpilot web --port 3000"
echo "  ./socialpilot web --host 0.0.0.0 --port 8080"
