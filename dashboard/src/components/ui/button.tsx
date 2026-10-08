import * as React from "react"
import { Slot } from "@radix-ui/react-slot"
import { cva, type VariantProps } from "class-variance-authority"

import { cn } from "@/lib/utils"

const buttonVariants = cva(
  "inline-flex shrink-0 items-center justify-center gap-2 whitespace-nowrap rounded-md border border-transparent bg-clip-padding text-sm font-medium tracking-[-0.005em] outline-none transition-all select-none focus-visible:ring-2 focus-visible:ring-vn-accent/35 active:translate-y-px disabled:pointer-events-none disabled:translate-y-0 disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0",
  {
    variants: {
      variant: {
        default:
          "bg-vn-accent text-white shadow-vn-glow hover:bg-vn-accent-2",
        destructive:
          "border-vn-danger/30 bg-vn-danger-soft text-vn-danger hover:bg-vn-danger/20",
        outline:
          "border-vn-hairline-2 bg-vn-surface text-vn-text shadow-sm hover:border-vn-hairline-3 hover:bg-vn-surface-2",
        secondary:
          "border-vn-hairline bg-vn-surface-2 text-vn-text hover:bg-vn-surface-3",
        ghost: "text-vn-text-2 hover:bg-vn-surface-2 hover:text-vn-text",
        link: "h-auto rounded-none border-none p-0 text-vn-accent-2 underline-offset-4 hover:text-vn-text hover:underline",
      },
      size: {
        default: "h-[42px] px-[18px]",
        sm: "h-9 px-3.5 text-[13px]",
        lg: "h-[46px] px-[22px] text-[14.5px]",
        icon: "h-10 w-10 p-0",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)

interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : "button"
    return (
      <Comp
        className={cn(buttonVariants({ variant, size, className }))}
        ref={ref}
        {...props}
      />
    )
  }
)
Button.displayName = "Button"

export { Button }
