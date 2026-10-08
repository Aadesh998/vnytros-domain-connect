import * as React from "react"

import { cn } from "@/lib/utils"

const Switch = React.forwardRef<
  HTMLInputElement,
  React.InputHTMLAttributes<HTMLInputElement>
>(({ className, ...props }, ref) => (
  <label
    className={cn(
      "relative inline-flex h-5 w-9 shrink-0 cursor-pointer items-center",
      props.disabled && "cursor-not-allowed opacity-50",
      className,
    )}
  >
    <input type="checkbox" className="peer sr-only" ref={ref} {...props} />
    <span className="absolute inset-0 rounded-full bg-vn-surface-3 transition-colors peer-checked:bg-vn-accent peer-focus-visible:ring-2 peer-focus-visible:ring-vn-accent/35 peer-focus-visible:ring-offset-2 peer-focus-visible:ring-offset-vn-bg" />
    <span className="pointer-events-none absolute left-0.5 size-4 rounded-full bg-vn-text shadow transition-transform peer-checked:translate-x-4" />
  </label>
))
Switch.displayName = "Switch"

export { Switch }
