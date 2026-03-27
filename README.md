# SocialPilot

<div align="center">

**本地社交关系管理助手 - AI 驱动的智能 CRM**

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[English](#english) | [中文文档](#中文文档)

</div>

---

## 中文文档

### 🎯 项目简介

SocialPilot 是一个基于 AI 的本地社交关系管理工具，帮助你：
- 记录和分析社交互动
- 生成联系人画像（含 MBTI 人格分析）
- 智能回复建议
- 管理长期关系

**核心特性：**
- 🏠 **完全本地化** - 数据存储在本地 SQLite，隐私安全
- 🤖 **AI 驱动** - 支持 OpenAI 兼容 API（GPT、Claude、文心一言等）
- 🎨 **现代化 Web UI** - React + TypeScript + shadcn/ui
- 📊 **MBTI 人格分析** - 基于对话自动分析人格类型
- ⚙️ **提示词可自定义** - 完全控制 AI 行为
- 📱 **响应式设计** - 支持桌面和移动端

### 📸 功能演示

#### Web UI 界面

**联系人管理**
- 搜索、创建、删除联系人
- 查看人物详情和画像
- 标签分类（同事、同学、朋友、亲戚）

**人物画像分析**
- MBTI 人格类型 + 描述
- 核心性格特征
- 沟通偏好与雷区
- 关系温度评估

**智能交互**
- 结构化录入（Log）
- 智能回复建议（Chat）
- 采纳回写闭环（Commit）
- 历史压缩（Compress）

### 🚀 快速开始

#### 方式一：下载预编译版本

前往 [Releases](https://github.com/cn-maul/socialpilot/releases) 下载对应平台的可执行文件。

#### 方式二：从源码构建

**前置要求：**
- Go 1.21+
- Node.js 18+ (构建 Web UI)

```bash
# 克隆仓库
git clone https://github.com/cn-maul/socialpilot.git
cd socialpilot

# 构建 Web UI
cd webui
npm install
npm run build
cd ..

# 编译后端
go build -o socialpilot .

# 运行
./socialpilot web
```

浏览器打开 http://127.0.0.1:8080

### 📖 使用指南

#### 1. 初始配置

首次使用需要配置 AI API：

**Web UI 配置：**
1. 打开 http://127.0.0.1:8080
2. 进入「设置」→「基础设置」
3. 填写：
   - Base URL（如：`https://api.openai.com/v1`）
   - API Key
   - Model（如：`gpt-4`）
   - 超时时间（默认 60 秒）

**支持的 API 提供商：**
- OpenAI (GPT-4, GPT-3.5)
- Anthropic (Claude)
- 百度千帆
- 智谱 AI
- 月之暗面
- Ollama (本地模型)
- 其他 OpenAI 兼容 API

#### 2. 创建联系人

**Web UI：**
- 左侧「新增联系人」填写姓名、性别、标签

**CLI：**
```bash
./socialpilot contact add --name "张三" --gender male --tags "同事"
```

#### 3. 录入互动记录

**Web UI：**
- 选择联系人 → 「结构化录入」
- 输入非结构化描述，如："今天他说方案需要改，语气有点急"
- 点击「执行录入」

**CLI：**
```bash
./socialpilot log --name "张三" --message "今天他说方案需要改"
```

**支持 JSON 导入：**
```bash
./socialpilot log --name "张三" --message '[{"sender":"张三","message":"你好"},{"sender":"我","message":"嗨"}]'
```

#### 4. 智能回复建议

**Web UI：**
- 「对方新消息」输入框填写对方刚发的消息
- 点击「生成建议」
- 查看多条建议，选择合适的采纳

**CLI：**
```bash
./socialpilot chat --name "张三" --message "你们什么时候给新版方案？"
```

#### 5. 更新人物画像

**Web UI：**
- 点击联系人标题栏的 ✨ 图标
- 等待 AI 分析完成
- 查看最新的 MBTI 人格、核心特征、沟通偏好、关系温度

**CLI：**
```bash
./socialpilot analyze --name "张三"
```

#### 6. 历史压缩

长时间运行后，可压缩旧会话节省空间：

```bash
# 压缩所有联系人的旧会话
./socialpilot compress --all

# 压缩指定联系人
./socialpilot compress --name "张三"
```

### ⚙️ 高级功能

#### 自定义 AI 提示词

进入「设置」→「提示词设置」可自定义：

1. **人物画像提示词 (Analyze)**
   - 控制画像输出格式和内容
   - 默认包含 MBTI 分析

2. **回复建议提示词 (Copilot)**
   - 调整回复风格
   - 可添加特定场景建议

3. **对话提取提示词 (Extract)**
   - 非结构化文本解析
   - JSON 格式转换

4. **历史压缩提示词 (Compress)**
   - 会话摘要生成
   - 重点信息提取

点击「重置为默认」可随时恢复。

#### MBTI 人格分析

基于对话行为模式分析：
- **E/I（外向/内向）**：主动发起对话倾向
- **S/N（感觉/直觉）**：细节描述 vs 概念思维
- **T/F（思考/情感）**：逻辑分析 vs 情感表达
- **J/P（判断/知觉）**：计划性 vs 灵活性

输出包含：
- MBTI 类型（如：INTJ）
- 10 字内人格描述
- 核心特征
- 沟通偏好与雷区
- 关系温度评分

### 📁 数据存储

- **数据库位置**：默认 `~/.local/share/socialpilot/socialpilot.db`
- **配置文件**：`~/.config/socialpilot/config.json`
- **完全本地**：数据不上传，隐私安全

### 🛠️ 命令行工具

```bash
# 查看帮助
./socialpilot --help

# JSON 输出模式
./socialpilot contact add --name "李四" --json

# Web 服务
./socialpilot web --host 0.0.0.0 --port 8080

# 使用代理
export SOCIALPILOT_USE_PROXY=1
./socialpilot web
```

### 🔧 开发相关

#### 项目结构

```
socialpilot/
├── cmd/                 # CLI 命令实现
├── internal/
│   ├── config/         # 配置管理
│   ├── db/             # 数据库模型
│   ├── llm/            # LLM 客户端
│   └── service/        # 业务逻辑
├── pkg/
│   └── jsonx/          # JSON 提取工具
├── webui/              # React 前端
│   ├── src/
│   │   ├── components/ # UI 组件
│   │   └── App.tsx     # 主应用
│   └── dist/           # 构建产物
├── main.go
└── README.md
```

#### 本地开发

```bash
# 启动前端开发服务器
cd webui
npm run dev

# 启动后端（另一个终端）
go run main.go web
```

### 📝 更新日志

查看 [CHANGELOG.md](CHANGELOG.md) 了解版本历史。

### 🤝 贡献

欢迎提交 Issue 和 Pull Request！

### 📄 License

MIT License

---

## English

### 🎯 Overview

SocialPilot is an AI-powered local social relationship management tool that helps you:
- Record and analyze social interactions
- Generate contact profiles with MBTI personality analysis
- Get intelligent reply suggestions
- Manage long-term relationships

### ✨ Key Features

- 🏠 **Fully Local** - Data stored in local SQLite, privacy-first
- 🤖 **AI-Powered** - Supports OpenAI-compatible APIs
- 🎨 **Modern Web UI** - React + TypeScript + shadcn/ui
- 📊 **MBTI Analysis** - Automatic personality typing from conversations
- ⚙️ **Customizable Prompts** - Full control over AI behavior
- 📱 **Responsive Design** - Desktop and mobile friendly

### 🚀 Quick Start

Download from [Releases](https://github.com/cn-maul/socialpilot/releases) or build from source:

```bash
git clone https://github.com/cn-mail/socialpilot.git
cd socialpilot
go build -o socialpilot .
./socialpilot web
```

Open http://127.0.0.1:8080

### 📖 Documentation

See [中文文档](#中文文档) for detailed usage guide.

### 📄 License

MIT License
