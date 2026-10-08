import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Activity,
  AlertCircle,
  ArrowUpRight,
  CheckCircle2,
  Clock,
  Globe,
  Key,
} from "lucide-react"

const metrics = [
  {
    label: "Total domains",
    value: "12",
    hint: "+2 from last month",
    icon: Globe,
    tone: "text-vn-accent bg-vn-accent-soft",
  },
  {
    label: "Active API keys",
    value: "3",
    hint: "No keys revoked recently",
    icon: Key,
    tone: "text-vn-cyan bg-vn-cyan-soft",
  },
  {
    label: "API requests",
    value: "14,203",
    hint: "+14% from last week",
    icon: Activity,
    tone: "text-vn-success bg-vn-success-soft",
  },
]

const usage = [42, 58, 49, 64, 72, 54, 80, 68, 92, 78, 88, 96]

const events = [
  {
    message: "Domain myapp.dev verification failed.",
    time: "2 hours ago",
    icon: AlertCircle,
    tone: "text-vn-danger bg-vn-danger-soft",
  },
  {
    message: "API key prod-key was created.",
    time: "Yesterday",
    icon: Key,
    tone: "text-vn-accent bg-vn-accent-soft",
  },
  {
    message: "Domain example.com verified successfully.",
    time: "Oct 01, 2023",
    icon: CheckCircle2,
    tone: "text-vn-success bg-vn-success-soft",
  },
]

export function DashboardFeature() {
  return (
    <div>
      <section className="mb-6">
        <span className="vn-eyebrow mb-2">dashboard</span>
        <h1 className="mt-2 text-3xl font-[540] tracking-[-0.035em] text-vn-text md:text-4xl">
          Overview
        </h1>
        <p className="mt-2 max-w-2xl text-vn-text-2">
          Connect domains, monitor verification status, and manage your Vnytros
          integration from one workspace.
        </p>
      </section>

      <section className="mb-5 grid grid-cols-1 gap-4 md:grid-cols-3">
        {metrics.map((metric) => {
          const Icon = metric.icon
          return (
            <Card key={metric.label}>
              <CardHeader className="flex flex-row items-start justify-between space-y-0 pb-3">
                <div>
                  <CardTitle className="font-mono text-[11px] uppercase tracking-widest text-vn-text-3">
                    {metric.label}
                  </CardTitle>
                  <div className="mt-3 text-3xl font-[540] leading-none tracking-[-0.03em] text-vn-text">
                    {metric.value}
                  </div>
                </div>
                <div
                  className={`flex size-10 items-center justify-center rounded-md ${metric.tone}`}
                >
                  <Icon className="size-5" />
                </div>
              </CardHeader>
              <CardContent>
                <p className="flex items-center text-xs text-vn-text-3">
                  {metric.label === "API requests" && (
                    <ArrowUpRight className="mr-1 size-3 text-vn-success" />
                  )}
                  {metric.hint}
                </p>
              </CardContent>
            </Card>
          )
        })}
      </section>

      <section className="grid grid-cols-1 gap-4 lg:grid-cols-7">
        <Card className="lg:col-span-4">
          <CardHeader>
            <div className="flex items-start justify-between gap-4">
              <div>
                <CardTitle className="text-base tracking-[-0.02em]">
                  Usage activity
                </CardTitle>
                <p className="mt-1 text-sm text-vn-text-3">
                  Mock API request volume over the last 12 periods.
                </p>
              </div>
              <span className="rounded-full border border-vn-success/30 bg-vn-success-soft px-2.5 py-1 font-mono text-[11px] text-vn-success">
                Live
              </span>
            </div>
          </CardHeader>
          <CardContent>
            <div className="flex h-[220px] items-end gap-2 rounded-xl border border-vn-hairline bg-vn-bg-2 p-4">
              {usage.map((height, index) => (
                <div
                  key={index}
                  className="flex flex-1 items-end rounded-full bg-vn-surface-3"
                >
                  <div
                    className="w-full rounded-full bg-gradient-to-t from-vn-accent to-vn-cyan shadow-vn-glow"
                    style={{ height: `${height}%` }}
                  />
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        <Card className="lg:col-span-3">
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle className="text-base tracking-[-0.02em]">
                  Recent events
                </CardTitle>
                <p className="mt-1 text-sm text-vn-text-3">
                  Latest actions in your workspace.
                </p>
              </div>
              <Clock className="size-4 text-vn-text-4" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {events.map((event) => {
                const Icon = event.icon
                return (
                  <div
                    key={`${event.message}-${event.time}`}
                    className="flex gap-3 rounded-xl border border-vn-hairline bg-vn-bg-2 p-3"
                  >
                    <div
                      className={`mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-lg ${event.tone}`}
                    >
                      <Icon className="size-4" />
                    </div>
                    <div className="min-w-0">
                      <p className="text-sm font-medium leading-snug text-vn-text">
                        {event.message}
                      </p>
                      <p className="mt-1 text-xs text-vn-text-3">
                        {event.time}
                      </p>
                    </div>
                  </div>
                )
              })}
            </div>
          </CardContent>
        </Card>
      </section>
    </div>
  )
}
