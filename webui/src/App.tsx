import { useEffect, useMemo, useState } from "react"
import { Toaster, toast } from "sonner"
import { Moon, Sun, Search, Plus, Trash2, Send, Sparkles, Database, User } from "lucide-react"

import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Separator } from "@/components/ui/separator"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Textarea } from "@/components/ui/textarea"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"

type Contact = {
  ID?: string
  Name?: string
  Gender?: string
  Tags?: string
  ProfileSummary?: string
  name?: string
  gender?: string
  tags?: string
  profile_summary?: string
}

type DetailMessage = {
  speaker: string
  content: string
  emotion: string
  intent: string
  timestamp: string
}

type DetailPayload = {
  contact: Contact
  stats: {
    session_count?: number
    message_count?: number
    compressed_count?: number
    last_interaction?: string
    latest_contact_msg?: string
    latest_user_msg?: string
  }
  messages: DetailMessage[]
}

type LlmConfig = {
  baseurl: string
  apikey: string
  model: string
  db_path: string
  timeout_seconds: number
}

type PromptsConfig = {
  extract: string
  copilot: string
  analyze: string
  compress: string
  default_extract: string
  default_copilot: string
  default_analyze: string
  default_compress: string
}

function normName(c: Contact): string {
  return c.name || c.Name || ""
}

function normGender(gender: string | undefined) {
  if ((gender || "").toLowerCase() === "female") {
    return { label: "女", variant: "destructive" as const }
  }
  return { label: "男", variant: "secondary" as const }
}

function stripMd(md: string): string {
  return String(md || "")
    .replace(/^#+\s*/gm, "")
    .replace(/^\|.*\|$/gm, " ")
    .replace(/^:?-{2,}:?$/gm, " ")
    .replace(/\*\*(.*?)\*\*/g, "$1")
    .replace(/\|/g, " ")
    .replace(/[-*]\s+/g, "")
    .replace(/\d+\.\s+/g, "")
    .replace(/\s+/g, " ")
    .trim()
}

function clip(input: string, n: number): string {
  const s = String(input || "").trim()
  if (s.length <= n) return s
  return `${s.slice(0, n)}...`
}

function clipRunes(input: string, n: number): string {
  const arr = Array.from(String(input || "").trim())
  if (arr.length <= n) return arr.join("")
  return `${arr.slice(0, n).join("")}...`
}

function shortContactSummary(md: string): string {
  const raw = stripMd(md)
    .replace(/行为心理学分析报告/g, "")
    .replace(/人物画像/g, "")
    .replace(/核心特征/g, "")
    .replace(/沟通偏好\/雷区/g, "")
    .replace(/关系温度评估/g, "")
    .trim()

  if (!raw) return "暂无"

  const first = raw
    .split(/[。；，\n]/)
    .map((x) => x.trim())
    .find((x) => x.length >= 2)

  if (!first) return "暂无"
  return clipRunes(first, 10)
}

function extractProfileSections(md: string) {
  const src = String(md || "")
  if (!src.trim()) {
    return { intro: "暂无画像", mbti: "暂无", core: "暂无", pref: "暂无", relation: "暂无" }
  }

  const lines = src.split("\n")
  let key = ""
  const sections: Record<string, string[]> = { mbti: [], core: [], pref: [], relation: [], other: [] }

  for (const raw of lines) {
    const line = raw.trim()
    if (/^##+\s*/.test(line)) {
      if (line.includes("MBTI") || line.includes("人格")) key = "mbti"
      else if (line.includes("核心") || line.includes("特征") || line.includes("性格")) key = "core"
      else if (line.includes("雷区") || line.includes("偏好") || line.includes("沟通")) key = "pref"
      else if (line.includes("关系温度") || line.includes("关系评估") || line.includes("亲密度")) key = "relation"
      else key = "other"
      continue
    }
    if (!line) continue
    sections[key || "other"].push(line)
  }

  const pickKeyLines = (text: string, limit: number) => {
    const parts = String(text || "")
      .split(/[。；\n]/)
      .map((x) => stripMd(x))
      .filter(Boolean)
    return clip(parts.slice(0, limit).join("；"), 180)
  }

  let mbti = pickKeyLines(sections.mbti.join("\n"), 2)
  let core = pickKeyLines(sections.core.join("\n"), 2)
  let pref = pickKeyLines(sections.pref.join("\n"), 3)
  let relation = pickKeyLines(sections.relation.join("\n"), 3)

  const fallbackParts = sections.other
    .join("\n")
    .split(/[。；\n]/)
    .map((x) => stripMd(x))
    .filter((x) => x.length >= 2)

  if (!mbti && fallbackParts[0]) mbti = clip(fallbackParts[0], 80)
  if (!core && fallbackParts[1]) core = clip(fallbackParts[1], 80)
  if (!pref && fallbackParts[2]) pref = clip(fallbackParts[2], 80)
  if (!relation && fallbackParts[3]) relation = clip(fallbackParts[3], 80)

  mbti = mbti || "-"
  core = core || "-"
  pref = pref || "-"
  relation = relation || "-"

  const introBase = [mbti, core, pref, relation].filter(x => x && x !== "-").join("；")
  const intro = clip(introBase || stripMd(src), 140) || "暂无画像"

  return { intro, mbti, core, pref, relation }
}

async function apiGet<T>(url: string): Promise<T> {
  const res = await fetch(url)
  const payload = await res.json()
  if (!res.ok || payload?.status !== "success") {
    throw new Error(payload?.error || `Request failed: ${res.status}`)
  }
  return payload as T
}

async function apiPost<T>(url: string, body: unknown): Promise<T> {
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  })
  const payload = await res.json()
  if (!res.ok || payload?.status !== "success") {
    throw new Error(payload?.error || `Request failed: ${res.status}`)
  }
  return payload as T
}

function App() {
  const [tab, setTab] = useState("contacts")
  const [dark, setDark] = useState(false)

  const [searchName, setSearchName] = useState("")
  const [contacts, setContacts] = useState<Contact[]>([])
  const [selectedName, setSelectedName] = useState("")

  const [addName, setAddName] = useState("")
  const [addGender, setAddGender] = useState("男")
  const [addTag, setAddTag] = useState("朋友")

  const [detail, setDetail] = useState<DetailPayload | null>(null)
  const [profile, setProfile] = useState({ intro: "暂无画像", mbti: "-", core: "-", pref: "-", relation: "-" })

  const [logText, setLogText] = useState("")
  const [chatText, setChatText] = useState("")
  const [commitText, setCommitText] = useState("")
  const [advice, setAdvice] = useState<Array<{ tone: string; content: string }>>([])

  const [config, setConfig] = useState<LlmConfig>({
    baseurl: "",
    apikey: "",
    model: "",
    db_path: "",
    timeout_seconds: 60,
  })

  const [prompts, setPrompts] = useState<PromptsConfig>({
    extract: "",
    copilot: "",
    analyze: "",
    compress: "",
    default_extract: "",
    default_copilot: "",
    default_analyze: "",
    default_compress: "",
  })

  const [loading, setLoading] = useState({
    config: false,
    prompts: false,
    search: false,
    add: false,
    detail: false,
    log: false,
    chat: false,
    commit: false,
    analyze: false,
    compress: false,
  })

  useEffect(() => {
    const savedTheme = localStorage.getItem("sp_theme")
    setDark(savedTheme === "dark")
  }, [])

  useEffect(() => {
    document.documentElement.classList.toggle("dark", dark)
    localStorage.setItem("sp_theme", dark ? "dark" : "light")
  }, [dark])

  async function loadConfig() {
    setLoading((s) => ({ ...s, config: true }))
    try {
      const res = await apiGet<{ config: LlmConfig }>("/api/config/get")
      setConfig({
        baseurl: res.config.baseurl || "",
        apikey: res.config.apikey || "",
        model: res.config.model || "",
        db_path: res.config.db_path || "",
        timeout_seconds: res.config.timeout_seconds || 60,
      })
      toast.success("配置已加载")
    } catch (error) {
      toast.error("加载配置失败", { description: String(error) })
    } finally {
      setLoading((s) => ({ ...s, config: false }))
    }
  }

  async function saveConfig() {
    setLoading((s) => ({ ...s, config: true }))
    try {
      await apiPost("/api/config/set", config)
      toast.success("配置保存成功")
    } catch (error) {
      toast.error("保存配置失败", { description: String(error) })
    } finally {
      setLoading((s) => ({ ...s, config: false }))
    }
  }

  async function loadPrompts() {
    setLoading((s) => ({ ...s, prompts: true }))
    try {
      const res = await apiGet<{ prompts: PromptsConfig }>("/api/prompts/get")
      setPrompts(res.prompts)
      toast.success("提示词已加载")
    } catch (error) {
      toast.error("加载提示词失败", { description: String(error) })
    } finally {
      setLoading((s) => ({ ...s, prompts: false }))
    }
  }

  async function savePrompts() {
    setLoading((s) => ({ ...s, prompts: true }))
    try {
      await apiPost("/api/prompts/set", {
        prompt_extract: prompts.extract,
        prompt_copilot: prompts.copilot,
        prompt_analyze: prompts.analyze,
        prompt_compress: prompts.compress,
      })
      toast.success("提示词保存成功")
    } catch (error) {
      toast.error("保存提示词失败", { description: String(error) })
    } finally {
      setLoading((s) => ({ ...s, prompts: false }))
    }
  }

  async function resetPrompts() {
    if (!window.confirm("确定要重置所有提示词为默认值吗？")) return
    setLoading((s) => ({ ...s, prompts: true }))
    try {
      await apiPost("/api/prompts/reset", {})
      setPrompts((s) => ({
        ...s,
        extract: "",
        copilot: "",
        analyze: "",
        compress: "",
      }))
      toast.success("提示词已重置为默认值")
    } catch (error) {
      toast.error("重置提示词失败", { description: String(error) })
    } finally {
      setLoading((s) => ({ ...s, prompts: false }))
    }
  }

  async function searchContacts() {
    setLoading((s) => ({ ...s, search: true }))
    try {
      const res = await apiGet<{ contacts: Contact[] }>(`/api/contact/search?q=${encodeURIComponent(searchName)}`)
      const rows = res.contacts || []
      setContacts(rows)
      toast.info(`找到 ${rows.length} 个联系人`)
      if (!selectedName && rows.length > 0) {
        const first = normName(rows[0])
        if (first) {
          await openContact(first)
        }
      }
    } catch (error) {
      toast.error("搜索失败", { description: String(error) })
    } finally {
      setLoading((s) => ({ ...s, search: false }))
    }
  }

  async function addContact() {
    if (!addName.trim()) {
      toast.warning("请输入姓名")
      return
    }
    setLoading((s) => ({ ...s, add: true }))
    try {
      const res = await apiPost<{ name: string }>("/api/contact/add", {
        name: addName,
        gender: addGender === "女" ? "female" : "male",
        tags: addTag,
      })
      setAddName("")
      toast.success(`联系人「${res.name}」创建成功`)
      await searchContacts()
      await openContact(res.name)
    } catch (error) {
      toast.error("创建失败", { description: String(error) })
    } finally {
      setLoading((s) => ({ ...s, add: false }))
    }
  }

  async function deleteContact() {
    if (!selectedName) return
    const ok = window.confirm(`确认删除联系人「${selectedName}」及其全部历史记录？该操作不可恢复。`)
    if (!ok) return
    setLoading((s) => ({ ...s, detail: true }))
    try {
      await apiPost("/api/contact/delete", { name: selectedName })
      setDetail(null)
      setSelectedName("")
      setAdvice([])
      setCommitText("")
      toast.success("联系人已删除")
      await searchContacts()
    } catch (error) {
      toast.error("删除失败", { description: String(error) })
    } finally {
      setLoading((s) => ({ ...s, detail: false }))
    }
  }

  async function openContact(name: string) {
    if (!name) return
    setSelectedName(name)
    setAdvice([])
    setCommitText("")
    await loadDetail(name)
  }

  async function loadDetail(name: string) {
    setLoading((s) => ({ ...s, detail: true }))
    try {
      const res = await apiGet<DetailPayload>(`/api/contact/detail?name=${encodeURIComponent(name)}`)
      setDetail(res)
      const summary = res.contact.profile_summary || res.contact.ProfileSummary || ""
      setProfile(extractProfileSections(summary))
    } catch (error) {
      toast.error("加载详情失败", { description: String(error) })
    } finally {
      setLoading((s) => ({ ...s, detail: false }))
    }
  }

  async function runLog() {
    if (!selectedName || !logText.trim()) return
    setLoading((s) => ({ ...s, log: true }))
    try {
      await apiPost("/api/log", { name: selectedName, message: logText })
      setLogText("")
      await loadDetail(selectedName)
      toast.success("录入成功")
    } catch (error) {
      toast.error("录入失败", { description: String(error) })
    } finally {
      setLoading((s) => ({ ...s, log: false }))
    }
  }

  async function runChat() {
    if (!selectedName || !chatText.trim()) return
    setLoading((s) => ({ ...s, chat: true }))
    try {
      const res = await apiPost<{ advice: Array<{ tone: string; content: string }> }>("/api/chat", {
        name: selectedName,
        message: chatText,
      })
      const adv = res.advice || []
      setAdvice(adv)
      setCommitText(adv[0]?.content || "")
      await loadDetail(selectedName)
      toast.success(`已生成 ${adv.length} 条建议`)
    } catch (error) {
      toast.error("生成建议失败", { description: String(error) })
    } finally {
      setLoading((s) => ({ ...s, chat: false }))
    }
  }

  async function runCommit() {
    if (!selectedName || !commitText.trim()) return
    setLoading((s) => ({ ...s, commit: true }))
    try {
      await apiPost("/api/commit", { name: selectedName, message: commitText })
      await loadDetail(selectedName)
      toast.success("回写成功")
    } catch (error) {
      toast.error("回写失败", { description: String(error) })
    } finally {
      setLoading((s) => ({ ...s, commit: false }))
    }
  }

  async function runAnalyze() {
    if (!selectedName) return
    setLoading((s) => ({ ...s, analyze: true }))
    try {
      await apiPost("/api/analyze", { name: selectedName })
      await loadDetail(selectedName)
      toast.success("画像更新成功")
    } catch (error) {
      toast.error("分析失败", { description: String(error) })
    } finally {
      setLoading((s) => ({ ...s, analyze: false }))
    }
  }

  async function runCompress() {
    if (!selectedName) return
    setLoading((s) => ({ ...s, compress: true }))
    try {
      await apiPost("/api/compress", { all: false, name: selectedName })
      toast.success("压缩完成")
    } catch (error) {
      toast.error("压缩失败", { description: String(error) })
    } finally {
      setLoading((s) => ({ ...s, compress: false }))
    }
  }

  useEffect(() => {
    void loadConfig()
    void loadPrompts()
    void searchContacts()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const selectedContact = useMemo(() => {
    if (!detail) return null
    const name = normName(detail.contact)
    return {
      name,
      gender: normGender(detail.contact.gender || detail.contact.Gender),
      tag: detail.contact.tags || detail.contact.Tags || "-",
      stats: detail.stats,
      messages: detail.messages || [],
    }
  }, [detail])

  return (
    <TooltipProvider>
      <Toaster position="top-center" richColors />
      <div className="mx-auto flex h-screen w-full max-w-[1760px] flex-col gap-4 overflow-hidden px-5 py-5 lg:px-8 lg:py-6">
        <Tabs className="h-full min-h-0" value={tab} onValueChange={setTab}>
          <div className="flex items-center justify-between gap-4">
            <TabsList className="h-11 w-fit gap-1.5 rounded-xl px-1.5">
              <TabsTrigger className="px-5 text-base" value="contacts">
                <User className="mr-2 h-4 w-4" />
                联系人与详情
              </TabsTrigger>
              <TabsTrigger className="px-5 text-base" value="settings">
                <Database className="mr-2 h-4 w-4" />
                设置
              </TabsTrigger>
            </TabsList>
            <Tooltip>
              <TooltipTrigger>
                <Button variant="secondary" size="icon" onClick={() => setDark((v) => !v)}>
                  {dark ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
                </Button>
              </TooltipTrigger>
              <TooltipContent>{dark ? "切换到浅色模式" : "切换到暗色模式"}</TooltipContent>
            </Tooltip>
          </div>

        <TabsContent className="mt-2 min-h-0 flex-1" value="contacts">
          <div className="grid h-full min-h-0 grid-cols-1 gap-5 xl:grid-cols-[420px_minmax(0,1fr)]">
            <div className="flex min-h-0 flex-col gap-5">
              <Card className="flex min-h-0 flex-1 flex-col">
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Search className="h-5 w-5" />
                    联系人搜索
                  </CardTitle>
                  <CardDescription>输入姓名关键字搜索联系人，点击后在右侧查看详情。</CardDescription>
                </CardHeader>
                <CardContent className="flex min-h-0 flex-1 flex-col">
                  <Field>
                    <FieldLabel htmlFor="search-name">姓名关键字</FieldLabel>
                    <div className="mt-2 flex items-center gap-2">
                      <Input
                        id="search-name"
                        className="flex-1"
                        placeholder="输入姓名搜索..."
                        value={searchName}
                        onChange={(e) => setSearchName(e.target.value)}
                        onKeyDown={(e) => e.key === "Enter" && void searchContacts()}
                      />
                      <Button onClick={() => void searchContacts()} disabled={loading.search}>
                        {loading.search ? <Skeleton className="h-4 w-12" /> : "搜索"}
                      </Button>
                      <Button variant="secondary" onClick={() => { setSearchName(""); void searchContacts() }}>
                        清空
                      </Button>
                    </div>
                  </Field>

                  <ScrollArea className="mt-3 min-h-0 flex-1 rounded-xl border bg-muted/30 p-3">
                    <div className="flex flex-col gap-2">
                      {loading.search ? (
                        <>
                          <Skeleton className="h-20 w-full rounded-xl" />
                          <Skeleton className="h-20 w-full rounded-xl" />
                          <Skeleton className="h-20 w-full rounded-xl" />
                        </>
                      ) : contacts.length === 0 ? (
                        <Empty>
                          <EmptyHeader>
                            <EmptyTitle>没有匹配联系人</EmptyTitle>
                            <EmptyDescription>可先创建联系人后再进行记录和分析。</EmptyDescription>
                          </EmptyHeader>
                        </Empty>
                      ) : (
                        contacts.map((c) => {
                          const name = normName(c)
                          const gender = normGender(c.gender || c.Gender)
                          const tags = c.tags || c.Tags || "-"
                          const ps = c.profile_summary || c.ProfileSummary || ""
                          return (
                            <Button
                              key={name}
                              variant={selectedName === name ? "secondary" : "ghost"}
                              className="h-auto justify-start rounded-xl px-3 py-3"
                              onClick={() => void openContact(name)}
                            >
                              <div className="flex w-full items-start gap-3">
                                <Avatar className="h-10 w-10">
                                  <AvatarFallback className={gender.variant === "destructive" ? "bg-pink-100 text-pink-700 dark:bg-pink-900 dark:text-pink-300" : "bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300"}>
                                    {name.slice(0, 1)}
                                  </AvatarFallback>
                                </Avatar>
                                <div className="flex flex-1 flex-col gap-1 text-left">
                                  <div className="font-medium">{name}</div>
                                  <div className="flex gap-2">
                                    <Badge variant={gender.variant}>{gender.label}</Badge>
                                    <Badge variant="outline">{tags}</Badge>
                                  </div>
                                  <div className="text-xs text-muted-foreground">{shortContactSummary(ps)}</div>
                                </div>
                              </div>
                            </Button>
                          )
                        })
                      )}
                    </div>
                  </ScrollArea>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Plus className="h-5 w-5" />
                    新增联系人
                  </CardTitle>
                  <CardDescription>快速创建联系人并进入详情页。</CardDescription>
                </CardHeader>
                <CardContent>
                  <FieldGroup className="grid grid-cols-2 gap-3">
                    <Field>
                      <FieldLabel htmlFor="add-name">姓名</FieldLabel>
                      <Input className="w-[170px]" id="add-name" value={addName} onChange={(e) => setAddName(e.target.value)} />
                    </Field>
                    <Field>
                      <FieldLabel>性别</FieldLabel>
                      <Select value={addGender} onValueChange={(v) => setAddGender(v || "男")}>
                        <SelectTrigger className="w-[170px]">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectGroup>
                            <SelectItem value="男">男</SelectItem>
                            <SelectItem value="女">女</SelectItem>
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                    </Field>
                    <Field>
                      <FieldLabel>标签</FieldLabel>
                      <Select value={addTag} onValueChange={(v) => setAddTag(v || "朋友")}>
                        <SelectTrigger className="w-[170px]">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectGroup>
                            <SelectItem value="同事">同事</SelectItem>
                            <SelectItem value="同学">同学</SelectItem>
                            <SelectItem value="朋友">朋友</SelectItem>
                            <SelectItem value="亲戚">亲戚</SelectItem>
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                    </Field>
                    <div className="flex items-end">
                      <Button onClick={() => void addContact()} disabled={loading.add}>
                        {loading.add ? "创建中..." : "创建联系人"}
                      </Button>
                    </div>
                  </FieldGroup>
                </CardContent>
              </Card>
            </div>

            <Card className="flex min-h-0 flex-col">
              {loading.detail && !detail ? (
                <CardContent className="flex min-h-0 flex-1 items-center justify-center">
                  <div className="flex flex-col items-center gap-4">
                    <Skeleton className="h-12 w-12 rounded-full" />
                    <Skeleton className="h-4 w-32" />
                    <Skeleton className="h-4 w-48" />
                  </div>
                </CardContent>
              ) : !selectedContact ? (
                <CardContent className="flex min-h-0 flex-1 items-center justify-center">
                  <Empty>
                    <EmptyHeader>
                      <EmptyTitle>尚未选择联系人</EmptyTitle>
                      <EmptyDescription>从左侧列表选择联系人，即可查看详情并执行 log/chat/commit/analyze/compress。</EmptyDescription>
                    </EmptyHeader>
                  </Empty>
                </CardContent>
              ) : (
                <>
                  <CardHeader>
                    <CardTitle className="flex items-center gap-3">
                      <Avatar className="h-10 w-10">
                        <AvatarFallback className={selectedContact.gender.variant === "destructive" ? "bg-pink-100 text-pink-700 dark:bg-pink-900 dark:text-pink-300" : "bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300"}>
                          {selectedContact.name.slice(0, 1)}
                        </AvatarFallback>
                      </Avatar>
                      <div className="flex flex-wrap items-center gap-2">
                        <Badge variant={selectedContact.gender.variant}>{selectedContact.gender.label}</Badge>
                        <Badge variant="outline">{selectedContact.tag}</Badge>
                        <Badge variant="secondary">
                          会话 {selectedContact.stats.session_count || 0} / 消息 {selectedContact.stats.message_count || 0}
                        </Badge>
                      </div>
                      <span className="text-xl">{selectedContact.name}</span>
                    </CardTitle>
                    <CardDescription>人物详情与交互工作台</CardDescription>
                    <CardAction>
                      <div className="flex gap-2">
                        <Tooltip>
                          <TooltipTrigger>
                            <Button variant="secondary" size="icon" onClick={() => void runAnalyze()} disabled={loading.analyze}>
                              <Sparkles className="h-4 w-4" />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>更新画像</TooltipContent>
                        </Tooltip>
                        <Tooltip>
                          <TooltipTrigger>
                            <Button variant="secondary" size="icon" onClick={() => void runCompress()} disabled={loading.compress}>
                              <Database className="h-4 w-4" />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>压缩历史</TooltipContent>
                        </Tooltip>
                        <Tooltip>
                          <TooltipTrigger>
                            <Button variant="destructive" size="icon" onClick={() => void deleteContact()} disabled={loading.detail}>
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>删除联系人</TooltipContent>
                        </Tooltip>
                      </div>
                    </CardAction>
                  </CardHeader>

                  <CardContent className="min-h-0 flex-1">
                    <ScrollArea className="h-full p-1 pr-2">
                      <div className="flex flex-col gap-5">
                    <div className="rounded-xl bg-muted/25 p-4">
                      <div className="mb-2 text-sm font-semibold">人物速览</div>
                      <div className="text-sm text-muted-foreground">{profile.intro}</div>
                    </div>

                        <div className="grid grid-cols-1 gap-3 xl:grid-cols-4">
                          <div className="rounded-xl bg-muted/30 p-4">
                            <div className="mb-2 text-sm font-semibold">MBTI人格</div>
                            <div className="text-sm text-muted-foreground">{profile.mbti}</div>
                          </div>
                          <div className="rounded-xl bg-muted/30 p-4">
                            <div className="mb-2 text-sm font-semibold">核心特征</div>
                            <div className="text-sm text-muted-foreground">{profile.core}</div>
                          </div>
                          <div className="rounded-xl bg-muted/30 p-4">
                            <div className="mb-2 text-sm font-semibold">沟通偏好/雷区</div>
                            <div className="text-sm text-muted-foreground">{profile.pref}</div>
                          </div>
                          <div className="rounded-xl bg-muted/30 p-4">
                            <div className="mb-2 text-sm font-semibold">关系温度评估</div>
                            <div className="text-sm text-muted-foreground">{profile.relation}</div>
                          </div>
                        </div>

                        <Separator />

                        <Card>
                      <CardHeader>
                        <CardTitle className="flex items-center gap-2">
                          <Send className="h-5 w-5" />
                          结构化录入（Log）
                        </CardTitle>
                      </CardHeader>
                      <CardContent>
                        <FieldGroup>
                          <Field>
                            <FieldLabel htmlFor="log-text">原始描述</FieldLabel>
                            <Textarea id="log-text" placeholder="例如：今天她说方案不够清晰..." value={logText} onChange={(e) => setLogText(e.target.value)} />
                          </Field>
                        </FieldGroup>
                      </CardContent>
                      <CardFooter className="flex gap-2">
                        <Button onClick={() => void runLog()} disabled={loading.log}>
                          {loading.log ? "录入中..." : "执行录入"}
                        </Button>
                      </CardFooter>
                    </Card>

                        <Card>
                      <CardHeader>
                        <CardTitle className="flex items-center gap-2">
                          <Sparkles className="h-5 w-5" />
                          智能回复建议 + 回写闭环
                        </CardTitle>
                      </CardHeader>
                      <CardContent className="flex flex-col gap-3">
                        <FieldGroup>
                          <Field>
                            <FieldLabel htmlFor="chat-text">对方新消息</FieldLabel>
                            <Textarea id="chat-text" placeholder="例如：你们什么时候给新版方案？" value={chatText} onChange={(e) => setChatText(e.target.value)} />
                          </Field>
                        </FieldGroup>
                        <div className="flex gap-2">
                          <Button onClick={() => void runChat()} disabled={loading.chat}>
                            {loading.chat ? "生成中..." : "生成建议"}
                          </Button>
                        </div>

                        <div className="flex flex-col gap-2">
                          {loading.chat ? (
                            <>
                              <Skeleton className="h-32 w-full rounded-xl" />
                              <Skeleton className="h-32 w-full rounded-xl" />
                            </>
                          ) : advice.length === 0 ? (
                            <FieldDescription>暂无建议，点击"生成建议"后在这里显示。</FieldDescription>
                          ) : (
                            advice.map((a, i) => (
                              <Card key={`${a.tone}-${i}`}>
                                <CardHeader>
                                  <CardTitle className="text-base">建议 {i + 1}</CardTitle>
                                  <CardDescription>{a.tone || "未命名"}</CardDescription>
                                </CardHeader>
                                <CardContent>{a.content}</CardContent>
                                <CardFooter>
                                  <Button variant="secondary" onClick={() => setCommitText(a.content)}>采用此建议</Button>
                                </CardFooter>
                              </Card>
                            ))
                          )}
                        </div>

                        <FieldGroup>
                          <Field>
                            <FieldLabel htmlFor="commit-text">最终回写内容</FieldLabel>
                            <Textarea id="commit-text" value={commitText} onChange={(e) => setCommitText(e.target.value)} />
                          </Field>
                        </FieldGroup>
                        <div className="flex gap-2">
                          <Button onClick={() => void runCommit()} disabled={loading.commit}>
                            {loading.commit ? "回写中..." : "采纳并回写"}
                          </Button>
                        </div>
                      </CardContent>
                    </Card>

                        <Card>
                      <CardHeader>
                        <CardTitle>最近消息（最多50条）</CardTitle>
                      </CardHeader>
                      <CardContent>
                        <ScrollArea className="h-[280px] rounded-xl border bg-muted/20 p-3">
                          <div className="flex flex-col gap-2">
                            {selectedContact.messages.length === 0 ? (
                              <FieldDescription>暂无消息</FieldDescription>
                            ) : (
                              selectedContact.messages.map((m, i) => (
                                <div key={`${m.timestamp}-${i}`} className="rounded-xl bg-muted/25 p-3">
                                  <div className="mb-1 flex items-center justify-between gap-2">
                                    <div className="text-sm font-semibold">{m.speaker === "user" ? "我" : "对方"}</div>
                                    <div className="text-xs text-muted-foreground">
                                      {(m.timestamp || "").replace("T", " ").slice(0, 19)}
                                    </div>
                                  </div>
                                  <div className="text-sm">{m.content}</div>
                                  <div className="mt-2 flex gap-2">
                                    {m.emotion ? <Badge variant="outline">情绪：{m.emotion}</Badge> : null}
                                    {m.intent ? <Badge variant="outline">意图：{m.intent}</Badge> : null}
                                  </div>
                                </div>
                              ))
                            )}
                          </div>
                        </ScrollArea>
                      </CardContent>
                    </Card>
                      </div>
                    </ScrollArea>
                  </CardContent>
                </>
              )}
            </Card>
          </div>
        </TabsContent>

        <TabsContent className="mt-2 min-h-0 flex-1" value="settings">
          <Tabs defaultValue="basic" className="flex h-full flex-col">
            <TabsList className="mb-4 h-10 w-fit gap-1 rounded-lg px-1">
              <TabsTrigger className="px-4 text-sm" value="basic">基础设置</TabsTrigger>
              <TabsTrigger className="px-4 text-sm" value="prompts">提示词设置</TabsTrigger>
            </TabsList>

            <div className="min-h-0 flex-1 overflow-auto">
              <TabsContent className="min-h-0" value="basic">
              <div className="flex justify-center">
                <Card className="w-full max-w-2xl">
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <Database className="h-5 w-5" />
                      AI 与系统设置
                    </CardTitle>
                    <CardDescription>这里的配置将用于 log/chat/analyze/compress 调用。</CardDescription>
                  </CardHeader>
                  <CardContent>
                    <FieldGroup className="grid grid-cols-1 gap-4 md:grid-cols-2">
                      <Field className="md:col-span-2">
                        <FieldLabel htmlFor="cfg-baseurl">Base URL</FieldLabel>
                        <Input id="cfg-baseurl" placeholder="https://api.openai.com/v1" value={config.baseurl} onChange={(e) => setConfig((s) => ({ ...s, baseurl: e.target.value }))} />
                      </Field>
                      <Field className="md:col-span-2">
                        <FieldLabel htmlFor="cfg-apikey">API Key</FieldLabel>
                        <Input id="cfg-apikey" type="password" placeholder="sk-..." value={config.apikey} onChange={(e) => setConfig((s) => ({ ...s, apikey: e.target.value }))} />
                      </Field>
                      <Field>
                        <FieldLabel htmlFor="cfg-model">Model</FieldLabel>
                        <Input id="cfg-model" placeholder="gpt-4o" value={config.model} onChange={(e) => setConfig((s) => ({ ...s, model: e.target.value }))} />
                      </Field>
                      <Field>
                        <FieldLabel htmlFor="cfg-timeout">超时（秒）</FieldLabel>
                        <Input
                          id="cfg-timeout"
                          type="number"
                          value={String(config.timeout_seconds)}
                          onChange={(e) => setConfig((s) => ({ ...s, timeout_seconds: Number.parseInt(e.target.value || "60", 10) || 60 }))}
                        />
                      </Field>
                      <Field className="md:col-span-2">
                        <FieldLabel htmlFor="cfg-db">数据库路径</FieldLabel>
                        <Input id="cfg-db" value={config.db_path} onChange={(e) => setConfig((s) => ({ ...s, db_path: e.target.value }))} />
                      </Field>
                    </FieldGroup>
                  </CardContent>
                  <CardFooter className="flex gap-2">
                    <Button onClick={() => void saveConfig()} disabled={loading.config}>
                      {loading.config ? "保存中..." : "保存设置"}
                    </Button>
                  </CardFooter>
                </Card>
              </div>
            </TabsContent>

            <TabsContent className="min-h-0" value="prompts">
              <div className="flex justify-center">
                <Card className="w-full max-w-2xl">
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <Sparkles className="h-5 w-5" />
                      AI 提示词设置
                    </CardTitle>
                    <CardDescription>自定义 AI 提示词。留空则使用默认提示词。修改后点击保存即可生效。</CardDescription>
                  </CardHeader>
                  <CardContent>
                    <FieldGroup className="flex flex-col gap-4">
                      <Field>
                        <FieldLabel htmlFor="prompt-analyze">人物画像提示词 (Analyze)</FieldLabel>
                        <FieldDescription>用于生成联系人画像的提示词</FieldDescription>
                        <Textarea
                          id="prompt-analyze"
                          className="min-h-[200px] font-mono text-sm"
                          placeholder={prompts.default_analyze || "留空使用默认提示词..."}
                          value={prompts.analyze}
                          onChange={(e) => setPrompts((s) => ({ ...s, analyze: e.target.value }))}
                        />
                      </Field>
                      <Field>
                        <FieldLabel htmlFor="prompt-copilot">回复建议提示词 (Copilot)</FieldLabel>
                        <FieldDescription>用于生成回复建议的提示词</FieldDescription>
                        <Textarea
                          id="prompt-copilot"
                          className="min-h-[150px] font-mono text-sm"
                          placeholder={prompts.default_copilot || "留空使用默认提示词..."}
                          value={prompts.copilot}
                          onChange={(e) => setPrompts((s) => ({ ...s, copilot: e.target.value }))}
                        />
                      </Field>
                      <Field>
                        <FieldLabel htmlFor="prompt-extract">对话提取提示词 (Extract)</FieldLabel>
                        <FieldDescription>用于从非结构化文本提取对话的提示词</FieldDescription>
                        <Textarea
                          id="prompt-extract"
                          className="min-h-[120px] font-mono text-sm"
                          placeholder={prompts.default_extract || "留空使用默认提示词..."}
                          value={prompts.extract}
                          onChange={(e) => setPrompts((s) => ({ ...s, extract: e.target.value }))}
                        />
                      </Field>
                      <Field>
                        <FieldLabel htmlFor="prompt-compress">对话压缩提示词 (Compress)</FieldLabel>
                        <FieldDescription>用于压缩历史对话的提示词</FieldDescription>
                        <Textarea
                          id="prompt-compress"
                          className="min-h-[100px] font-mono text-sm"
                          placeholder={prompts.default_compress || "留空使用默认提示词..."}
                          value={prompts.compress}
                          onChange={(e) => setPrompts((s) => ({ ...s, compress: e.target.value }))}
                        />
                      </Field>
                    </FieldGroup>
                  </CardContent>
                  <CardFooter className="flex gap-2">
                    <Button onClick={() => void savePrompts()} disabled={loading.prompts}>
                      {loading.prompts ? "保存中..." : "保存提示词"}
                    </Button>
                    <Button variant="secondary" onClick={() => void resetPrompts()} disabled={loading.prompts}>
                      重置为默认
                    </Button>
                  </CardFooter>
                </Card>
              </div>
            </TabsContent>
            </div>
          </Tabs>
        </TabsContent>
      </Tabs>
    </div>
    </TooltipProvider>
  )
}

export default App
