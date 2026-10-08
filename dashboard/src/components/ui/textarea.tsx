import * as React from "react"

import { cn } from "@/lib/utils"

const Textarea = React.forwardRef<
  HTMLTextAreaElement,
  React.TextareaHTMLAttributes<HTMLTextAreaElement>
>(({ className, ...props }, ref) => (
  <textarea
    ref={ref}
    className={cn(
      "flex min-h-[88px] w-full rounded-md border border-vn-hairline-2 bg-vn-bg-2 px-3.5 py-2 text-sm text-vn-text shadow-sm transition-colors placeholder:text-vn-text-4 focus-visible:border-vn-accent/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-vn-accent/25 disabled:cursor-not-allowed disabled:opacity-50",
      className,
    )}
    {...props}
  />
))
Textarea.displayName = "Textarea"

export { Textarea }
