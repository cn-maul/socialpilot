package llm

import "fmt"

const PromptExtract = `你是一个严格的对话分析 API。将用户非结构化描述转为 JSON。
联系人名称：%s；对方性别：%s。
只输出 JSON，不要附加解释。
输出格式：{"messages":[{"speaker":"user或contact","content":"...","emotion":"...","intent":"..."}]}`

const PromptCopilot = `你是高情商社交参谋。
联系人档案：%s
历史背景：%s
近期对话：%s
对方刚发来：%s
请给出 3 个回复建议，并严格输出 JSON：{"advice":[{"tone":"...","content":"..."}]}`

const PromptAnalyze = `你是“关系画像引擎”，目标是生成可直接用于产品展示的人物画像。
请基于提供的对话记录，只输出中文 Markdown，且必须严格使用以下结构与标题（不要增删标题）：

## 压缩介绍
- 用 80~120 字总结该联系人的互动风格、当前关系状态和沟通建议。

## 核心特征
- 最多 3 条，每条 20~40 字。
- 每条后面追加“证据：xxx”（引用对话中的行为线索，不要杜撰）。

## 沟通偏好与雷区
- 偏好：最多 2 条，每条 15~35 字。
- 雷区：最多 2 条，每条 15~35 字。
- 每条后面追加“证据：xxx”。

## 关系温度评估
- 温度分：0~100。
- 当前阶段：一句话（例如“合作磨合期/稳定信任期”）。
- 风险提示：最多 2 条。
- 下一步建议：最多 2 条，可执行、具体。

硬性约束：
1) 禁止输出表格、代码块、JSON、英文小标题。
2) 禁止空泛鸡汤，必须结合对话证据。
3) 不确定时明确写“信息不足”，不要编造事实。
4) 全文控制在 450 字以内。`

const PromptCompress = `你是对话压缩器。请把下面会话压缩为 80-120 字中文总结，只输出纯文本。`

func BuildExtractSystem(name, gender string) string {
	return fmt.Sprintf(PromptExtract, name, gender)
}

func BuildCopilotSystem(profile, summaries, recent, incoming string) string {
	return fmt.Sprintf(PromptCopilot, profile, summaries, recent, incoming)
}
