package llm

import "fmt"

// Default prompt templates
const DefaultPromptExtract = `你是一个严格的对话分析 API。将用户非结构化描述转为 JSON。
联系人名称：%s；对方性别：%s。
只输出 JSON，不要附加解释。
输出格式：{"messages":[{"speaker":"user或contact","content":"...","emotion":"...","intent":"..."}]}`

const DefaultPromptCopilot = `你是高情商社交参谋。
联系人档案：%s
历史背景：%s
近期对话：%s
对方刚发来：%s
请给出 3 个回复建议，并严格输出 JSON：{"advice":[{"tone":"...","content":"..."}]}`

const DefaultPromptAnalyze = `你是专业的心理分析师，基于对话内容生成精准的人物画像。

请严格按以下结构输出中文 Markdown（禁止添加或删除标题）：

## MBTI人格
- 类型：[ENFP/INTJ等]

## 核心特征
- [最多2条，每条15-30字，精炼概括性格特质]

## 沟通偏好/雷区
- 偏好：[1-2条，每条10-20字]
- 雷区：[1-2条，每条10-20字]

## 关系温度
- 分数：[0-100]
- 阶段：[如：初识期/熟悉期/信任期]
- 建议：[1条可执行建议，20字内]

硬性约束：
1) 禁止表格、代码块、JSON、英文标题
2) 内容必须精炼，拒绝空泛描述
3) 禁止引用对话原文，仅输出分析结论
4) 不确定处标注"信息不足"
5) 全文控制在300字内`

const DefaultPromptCompress = `你是对话压缩器。请把下面会话压缩为 80-120 字中文总结，只输出纯文本。`

// Current prompt templates (can be customized)
var PromptExtract = DefaultPromptExtract
var PromptCopilot = DefaultPromptCopilot
var PromptAnalyze = DefaultPromptAnalyze
var PromptCompress = DefaultPromptCompress

// SetPrompts updates the prompt templates. Empty strings are ignored.
func SetPrompts(extract, copilot, analyze, compress string) {
	if extract != "" {
		PromptExtract = extract
	}
	if copilot != "" {
		PromptCopilot = copilot
	}
	if analyze != "" {
		PromptAnalyze = analyze
	}
	if compress != "" {
		PromptCompress = compress
	}
}

// ResetPrompts resets all prompts to their default values.
func ResetPrompts() {
	PromptExtract = DefaultPromptExtract
	PromptCopilot = DefaultPromptCopilot
	PromptAnalyze = DefaultPromptAnalyze
	PromptCompress = DefaultPromptCompress
}

// GetDefaultPrompts returns all default prompt templates.
func GetDefaultPrompts() (extract, copilot, analyze, compress string) {
	return DefaultPromptExtract, DefaultPromptCopilot, DefaultPromptAnalyze, DefaultPromptCompress
}

// GetCurrentPrompts returns all current prompt templates.
func GetCurrentPrompts() (extract, copilot, analyze, compress string) {
	return PromptExtract, PromptCopilot, PromptAnalyze, PromptCompress
}

func BuildExtractSystem(name, gender string) string {
	return fmt.Sprintf(PromptExtract, name, gender)
}

func BuildCopilotSystem(profile, summaries, recent, incoming string) string {
	return fmt.Sprintf(PromptCopilot, profile, summaries, recent, incoming)
}
