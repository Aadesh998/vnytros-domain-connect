import * as React from "react"

import { cn } from "@/lib/utils"

const Input = React.forwardRef<HTMLInputElement, React.ComponentProps<"input">>(
  ({ className, type, ...props }, ref) => {
    return (
      <input
        type={type}
        className={cn(
          "flex h-[42px] w-full rounded-md border border-vn-hairline-2 bg-vn-bg-2 px-3.5 py-2 text-base text-vn-text shadow-sm transition-colors file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-vn-text placeholder:text-vn-text-4 focus-visible:border-vn-accent/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-vn-accent/25 disabled:cursor-not-allowed disabled:opacity-50 md:text-sm",
          className
        )}
        ref={ref}
        {...props}
      />
    )
  }
)
Input.displayName = "Input"

export { Input }
