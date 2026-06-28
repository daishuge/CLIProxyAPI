import * as React from "react";
import { cn } from "@/lib/utils";

/**
 * Shimmer placeholder for loading states. Pairs with EmptyState and the error
 * state to form the unified three-state pattern used across data surfaces.
 */
function Skeleton({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        "relative overflow-hidden rounded-md bg-muted",
        "before:absolute before:inset-0 before:-translate-x-full",
        "before:bg-gradient-to-r before:from-transparent before:via-foreground/[0.06] before:to-transparent",
        "before:animate-[ppap-shimmer_1.5s_infinite]",
        className,
      )}
      {...props}
    />
  );
}

export { Skeleton };
