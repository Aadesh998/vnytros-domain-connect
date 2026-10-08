import type { LucideIcon } from "lucide-react"
import { cn } from "@/lib/utils"

/**
 * A single headline figure. Kept deliberately plain — the value is the thing
 * being read, so it carries the visual weight and the label recedes.
 */
export function StatTile({
  label,
  value,
  hint,
  icon: Icon,
  tone = "neutral",
}: {
  label: string
  value: string | number
  hint?: string
  icon?: LucideIcon
  tone?: "neutral" | "positive" | "negative"
}) {
  const toneClass =
    tone === "positive"
      ? "text-vn-success"
      : tone === "negative"
        ? "text-vn-danger"
        : "text-vn-text"

  return (
    <div className="rounded-xl border border-vn-hairline bg-vn-surface p-4">
      <div className="flex items-center justify-between gap-2">
        <p className="font-mono text-[10px] uppercase tracking-widest text-vn-text-4">
          {label}
        </p>
        {Icon && <Icon className="size-4 text-vn-text-4" />}
      </div>
      <p className={cn("mt-2 text-2xl font-semibold tabular-nums", toneClass)}>
        {value}
      </p>
      {hint && <p className="mt-1 text-xs text-vn-text-3">{hint}</p>}
    </div>
  )
}
