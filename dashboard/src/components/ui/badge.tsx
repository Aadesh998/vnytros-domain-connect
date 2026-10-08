import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"

import { cn } from "@/lib/utils"

const badgeVariants = cva(
  "inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-vn-accent/35",
  {
    variants: {
      variant: {
        default:
          "border-vn-accent/30 bg-vn-accent-soft text-vn-accent",
        secondary:
          "border-vn-hairline-2 bg-vn-surface-2 text-vn-text-2",
        destructive:
          "border-vn-danger/30 bg-vn-danger-soft text-vn-danger",
        outline: "border-vn-hairline-2 text-vn-text-2",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
)

interface BadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof badgeVariants> {}

function Badge({ className, variant, ...props }: BadgeProps) {
  return (
    <div className={cn(badgeVariants({ variant }), className)} {...props} />
  )
}

export { Badge }
