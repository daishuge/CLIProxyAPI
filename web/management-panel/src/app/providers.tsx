import * as React from "react";
import { QueryClientProvider } from "@tanstack/react-query";
import { queryClient } from "@/lib/query";
import { TooltipProvider, Toaster } from "@/components/ui";
import { HashRouterProvider } from "./router";

/** Composes the app-wide context providers in one place. */
export function AppProviders({ children }: { children: React.ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      <TooltipProvider delayDuration={200}>
        <HashRouterProvider>{children}</HashRouterProvider>
        <Toaster />
      </TooltipProvider>
    </QueryClientProvider>
  );
}
