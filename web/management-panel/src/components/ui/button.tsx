import * as React from "react";
import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import { Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";

const buttonVariants = cva(
  [
    "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md",
    "text-sm font-medium select-none",
    "transition-[background-color,box-shadow,color,border-color] duration-[var(--duration-fast)]",
    "focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring",
    "disabled:pointer-events-none disabled:opacity-50",
    "[&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
  ],
  {
    variants: {
      variant: {
        primary:
          "bg-primary text-primary-foreground shadow-sm hover:bg-brand-600 active:bg-brand-700",
        secondary:
          "bg-secondary text-secondary-foreground border border-border hover:bg-muted active:bg-surface-sunken",
        outline:
          "border border-border-strong bg-transparent text-foreground hover:bg-muted active:bg-surface-sunken",
        ghost: "bg-transparent text-foreground hover:bg-muted active:bg-surface-sunken",
        danger:
          "bg-danger text-danger-foreground shadow-sm hover:opacity-90 active:opacity-80",
        link: "text-primary underline-offset-4 hover:underline",
      },
      size: {
        sm: "h-8 px-3 text-xs rounded-sm",
        md: "h-9 px-4",
        lg: "h-10 px-6 text-base rounded-lg",
        icon: "size-9 p-0",
        "icon-sm": "size-8 p-0 rounded-sm",
      },
    },
    defaultVariants: {
      variant: "primary",
      size: "md",
    },
  },
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
  loading?: boolean;
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, loading = false, children, disabled, ...props }, ref) => {
    const Comp = asChild ? Slot : "button";
    return (
      <Comp
        ref={ref}
        className={cn(buttonVariants({ variant, size }), className)}
        disabled={disabled ?? loading}
        {...props}
      >
        {loading && !asChild ? (
          <Loader2 className="animate-[ppap-spin_0.7s_linear_infinite]" />
        ) : null}
        {children}
      </Comp>
    );
  },
);
Button.displayName = "Button";

export { Button, buttonVariants };
