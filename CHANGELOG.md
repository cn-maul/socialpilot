# Changelog

## [1.5.0] - 2026-03-27

### Added
- **AI 提示词自定义**: 支持在设置页面自定义 4 个 AI 提示词。
  - `Analyze`: 人物画像生成提示词
  - `Copilot`: 回复建议生成提示词
  - `Extract`: 对话提取提示词
  - `Compress`: 历史压缩提示词
- **提示词重置**: 一键重置所有提示词为默认值。
- **设置页面分页**: 设置页面拆分为两个子页面。
  - 基础设置: API URL、Key、Model、数据库路径、超时
  - 提示词设置: 4 个可自定义提示词编辑器
- **MBTI人格分析**: 人物画像新增MBTI人格类型分析。

### Changed
- 配置文件新增 `prompt_extract`、`prompt_copilot`、`prompt_analyze`、`prompt_compress` 字段。
- 提示词支持运行时热更新，无需重启服务。
- **优化人物画像提示词**:
  - 内容更精简（控制在300字内）
  - 新增MBTI人格分析模块
  - 结构优化为：MBTI人格 → 核心特征 → 沟通偏好/雷区 → 关系温度
  - 前端显示改为4列布局
- **简化联系人详情页面**:
  - 移除消息结构图和活跃曲线图表
  - 移除统计信息（最近互动、压缩会话、对方/我最近一句）
  - 人物画像卡片直接显示在速览下方

### Fixed
- 设置页面提示词区域支持滚动显示。

## [1.4.0] - 2026-03-24

### Added
- **JSON 聊天记录导入**: `log` 命令现在支持直接粘贴 JSON 数组格式的聊天记录。
  - 支持多种字段名：`sender/speaker/from/name/user/author` + `message/content/text/msg/body`
  - 自动识别"我"、"self"、"user"等为用户发言
  - 格式示例：`[{"sender": "李依娴", "message": "你好"}, {"sender": "我", "message": "嗨"}]`
  - 如果 JSON 解析失败，自动回退到 LLM 解析非结构化文本

### Changed
- **`normalizeSpeaker` 增强**: 支持更多用户别名（我、self、me、自己、本人、主）。

## [1.3.0] - 2026-03-24

### Added
- **新 shadcn 组件**: 添加 `sonner`、`skeleton`、`avatar`、`tooltip`、`alert` 组件，提升 UI 体验。
- **Toast 通知系统**: 使用 `sonner` 替代状态文本，提供更优雅的操作反馈（成功/错误/警告提示）。
- **联系人头像**: 联系人列表和详情页现在显示基于姓名首字的头像，男/女不同颜色。
- **加载骨架屏**: 搜索、聊天建议等区域添加加载状态骨架屏，提升用户体验。
- **图标按钮**: 操作按钮使用图标增强可读性（搜索、添加、删除、发送等）。
- **主题切换图标**: 暗色/浅色模式切换按钮改为太阳/月亮图标。
- **工具提示**: 关键按钮添加 Tooltip 说明（如"删除联系人"、"更新画像"、"压缩历史"）。

### Changed
- **状态管理重构**: 使用统一的 `loading` 对象替代多个独立的 `*Status` 状态。
- **API Key 输入框**: 设置页面的 API Key 输入框改为密码类型，增强安全性。
- **按钮禁用状态**: 所有操作按钮在请求期间自动禁用，防止重复提交。
- **输入框占位符**: 为常用输入框添加占位符提示文字。

## [1.2.0] - 2026-03-24

### Security
- **API Key 脱敏**: `/api/config/get` 端点现在返回脱敏后的 API Key（如 `sk-****abcd`），防止敏感凭证在前端泄露。

### Changed
- **数据库连接池**: 在 service 层实现全局单例数据库连接管理，避免每次请求都打开/关闭连接的性能开销。
- **代码去重**: 将重复的 `openService`/`mustService` 函数提取到共享的 `service.OpenService()`，减少 CLI 和 Web 处理器之间的代码重复。

### Added
- **JSON 提取器测试**: 为 `pkg/jsonx/extract.go` 添加完整的单元测试，覆盖 Markdown 代码块、嵌套对象、混合内容等边界情况。

### Fixed
- 重构后清理了 `cmd/root.go` 和 `cmd/web.go` 中未使用的导入。

## [1.1.0] - 2026-03-23

### Added
- 新增独立 `webui/` 前端工程（Vite + React + TypeScript + shadcn/ui）。
- 新增统一设置页，集中管理 `baseurl/apikey/model/timeout/db_path`。
- 新增联系人搜索与列表视图，支持姓名检索与点击进入详情。
- 新增联系人创建表单，性别与标签改为固定下拉选项（单选）：
  - 性别：男/女
  - 标签：同事/同学/朋友/亲戚
- 新增暗色模式切换并持久化到 `localStorage`。

### Changed
- 人物详情页重构为工作台式布局，整合并串联流程：
  - `log` 结构化录入
  - `chat` 智能建议
  - 建议采纳写入 `commit`（闭环）
  - `analyze` 画像更新
  - `compress` 历史压缩
- 人物画像展示改为结构化摘要（核心特征/沟通偏好与雷区/关系温度），不直接原样展示完整 Markdown。
- 联系人卡片补充性别徽标与标签徽标，提升列表可读性。

### Fixed
- 修复联系人名称展示异常（此前可能显示为 `"/ - 暂无画像"`）导致无法正确识别的问题。
- 修复前端交互中“单选字段被当作多选类型”引发的 TypeScript 构建错误。
- 调整 ESLint 配置，消除 shadcn 生成组件在 `react-refresh/only-export-components` 规则下的误报。

### Verified
- `webui` 构建通过：`npm run build`
- `webui` Lint 通过：`npm run lint`

## [1.0.0] - 2026-03-23

### Added
- 初始化 Go CLI 项目结构：`cmd/`, `internal/`, `pkg/`, `main.go`。
- 集成 `cobra` 命令框架，提供子命令：
  - `config set`
  - `contact add`
  - `log`
  - `chat`
  - `commit`
  - `analyze`
  - `compress`
- 实现全局 `--json/-j` 输出模式。
- 实现退出码规范：
  - `0` 成功
  - `1` 参数或本地验证失败
  - `2` LLM 网络/API 失败
  - `3` 数据库错误
  - `4` LLM JSON 解析失败

### Database
- 使用 `modernc.org/sqlite` + `sqlx` 实现 SQLite 存储层。
- 自动迁移并创建核心表：
  - `contacts`
  - `sessions`
  - `messages`
  - `raw_logs`
- 新增必要索引以支持查询性能。

### LLM
- 实现 OpenAI 兼容 API 客户端（`/v1/chat/completions`）。
- 支持 `baseurl/apikey/model` 配置化。
- 实现 Prompt 模板：
  - 结构化提取（extract）
  - 回复建议（copilot）
  - 人物分析（analyze）
  - 会话压缩（compress）
- 实现 JSON 提取容错（从 Markdown code fence 或文本中提取 JSON）。

### Service Flows
- Ingest Flow：原始日志入 `raw_logs`，LLM 结构化后入 `messages`。
- Copilot Flow：记录来信 + 装配短期/长期记忆 + 返回建议。
- Commit Flow：记录用户最终发送内容闭环。
- Analyze Flow：聚合对话生成画像并写回 `contacts.profile_summary`。
- Compress Flow：压缩旧会话并更新 `sessions.summary/status`。

### Docs
- 新增 `README.md`，包含构建与快速开始示例。

### Verified
- `go mod tidy` 通过。
- `go build ./...` 通过。
- 基础命令路径与主要错误码路径已验证。

### Known Gaps
- 尚未实现更细粒度数据库错误分类（锁冲突等）到 `exit code 3` 的专项测试。
- 尚未提供自动化测试（单元测试/集成测试）。
- `compress` 当前采用固定“7天未更新”策略，尚未参数化。
- 未实现联系人列表、会话查看等运维型命令。
