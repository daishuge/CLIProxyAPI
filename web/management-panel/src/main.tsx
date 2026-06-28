import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import "@/styles/index.css";
import "@/i18n";

import { bindAuthToClient } from "@/lib/auth";
import { initTheme } from "@/lib/theme";
import { AppProviders } from "@/app/providers";
import { ErrorBoundary } from "@/app/error-boundary";
import { App } from "@/app/app";

// Bootstrap: wire the API client to auth state and apply the persisted theme
// before the first paint to avoid a flash of the wrong color scheme.
bindAuthToClient();
initTheme();

const rootElement = document.getElementById("root");
if (!rootElement) {
  throw new Error("Root element #root not found");
}

createRoot(rootElement).render(
  <StrictMode>
    <ErrorBoundary>
      <AppProviders>
        <App />
      </AppProviders>
    </ErrorBoundary>
  </StrictMode>,
);
