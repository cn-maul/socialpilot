# SocialPilot

基于 `design.md` 实现的本地社交辅助 CLI（Go + SQLite + OpenAI 兼容 API）。

## 构建

```bash
go build -o socialpilot .
```

## 快速开始

1. 配置 LLM 与数据库路径：

```bash
./socialpilot config set \
  --baseurl http://127.0.0.1:11434 \
  --apikey sk-xxx \
  --model your-model \
  --timeout 60 \
  --db /tmp/socialpilot.db
```

2. 添加联系人：

```bash
./socialpilot contact add --name "林月" --gender female --tags "客户"
```

3. 录入非结构化社交日志：

```bash
./socialpilot log --name "林月" --message "今天她发火了，说我们的方案没看懂。"
```

4. 获取回复建议：

```bash
./socialpilot chat --name "林月" --message "你们最快什么时候能给新方案？"
```

5. 记录采纳结果：

```bash
./socialpilot commit --name "林月" --message "我选了第一种发过去了"
```

6. 更新人物画像：

```bash
./socialpilot analyze --name "林月"
```

7. 压缩历史会话：

```bash
./socialpilot compress --all
```

## JSON 模式

所有命令支持 `-j/--json`，结果输出到 `stdout`，错误输出到 `stderr`。

## 代理说明

- 默认不使用系统代理，避免本地代理端口未启动导致请求失败。
- 如果你确实需要走 `HTTP_PROXY/HTTPS_PROXY`，启动前设置：

```bash
export SOCIALPILOT_USE_PROXY=1
```

## Web 界面

启动 Web 应用界面：

```bash
./socialpilot web --host 127.0.0.1 --port 8080
```

然后浏览器打开 `http://127.0.0.1:8080`：
- `设置`页管理 AI 配置和数据库参数
- `联系人与详情`页支持姓名搜索、查看人物详情与最近消息
- 在人物详情页完成 `log/chat/commit/analyze/compress` 全流程

## 退出码

- `0` 成功
- `1` 参数/本地验证失败
- `2` LLM 网络或 API 失败
- `3` 数据库失败
- `4` LLM JSON 解析失败
