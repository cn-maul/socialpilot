import { useMemo } from "react"
import { Bar, BarChart, CartesianGrid, Line, LineChart, XAxis } from "recharts"

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from "@/components/ui/chart"

type DetailMessage = {
  speaker: string
  content: string
}

const speakerChartConfig = {
  me: { label: "我", color: "var(--chart-1)" },
  contact: { label: "对方", color: "var(--chart-2)" },
} satisfies ChartConfig

const activityChartConfig = {
  length: { label: "消息长度", color: "var(--chart-3)" },
} satisfies ChartConfig

export default function ContactCharts({ messages }: { messages: DetailMessage[] }) {
  const speakerData = useMemo(() => {
    const me = messages.filter((m) => m.speaker === "user").length
    const contact = messages.filter((m) => m.speaker !== "user").length
    return [
      { role: "我", me, contact: 0 },
      { role: "对方", me: 0, contact },
    ]
  }, [messages])

  const activityData = useMemo(() => {
    const rows = [...messages].reverse().slice(-12)
    return rows.map((m, i) => ({
      seq: `${i + 1}`,
      length: (m.content || "").length,
    }))
  }, [messages])

  return (
    <div className="grid grid-cols-1 gap-4 2xl:grid-cols-2">
      <Card>
        <CardHeader>
          <CardTitle>消息结构图</CardTitle>
          <CardDescription>我与对方在最近消息中的占比。</CardDescription>
        </CardHeader>
        <CardContent>
          <ChartContainer config={speakerChartConfig} className="min-h-[220px] w-full">
            <BarChart accessibilityLayer data={speakerData} margin={{ left: 8, right: 8 }}>
              <CartesianGrid vertical={false} />
              <XAxis dataKey="role" tickLine={false} axisLine={false} tickMargin={8} />
              <ChartTooltip content={<ChartTooltipContent />} />
              <Bar dataKey="me" fill="var(--color-me)" radius={8} />
              <Bar dataKey="contact" fill="var(--color-contact)" radius={8} />
            </BarChart>
          </ChartContainer>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>最近活跃曲线</CardTitle>
          <CardDescription>最近 12 条消息的长度变化趋势。</CardDescription>
        </CardHeader>
        <CardContent>
          <ChartContainer config={activityChartConfig} className="min-h-[220px] w-full">
            <LineChart accessibilityLayer data={activityData} margin={{ left: 8, right: 8 }}>
              <CartesianGrid vertical={false} />
              <XAxis dataKey="seq" tickLine={false} axisLine={false} tickMargin={8} />
              <ChartTooltip content={<ChartTooltipContent />} />
              <Line
                dataKey="length"
                type="monotone"
                stroke="var(--color-length)"
                strokeWidth={2.5}
                dot={false}
              />
            </LineChart>
          </ChartContainer>
        </CardContent>
      </Card>
    </div>
  )
}
