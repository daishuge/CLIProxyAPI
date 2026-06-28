import * as React from "react";
import { withTranslation, type WithTranslation } from "react-i18next";
import { AlertOctagon } from "lucide-react";
import { Button } from "@/components/ui";

interface ErrorBoundaryState {
  hasError: boolean;
}

class ErrorBoundaryBase extends React.Component<
  WithTranslation & { children: React.ReactNode },
  ErrorBoundaryState
> {
  constructor(props: WithTranslation & { children: React.ReactNode }) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(): ErrorBoundaryState {
    return { hasError: true };
  }

  componentDidCatch(error: Error): void {
    // Surface render failures in the console for diagnostics.
    console.error("Render error:", error);
  }

  override render(): React.ReactNode {
    const { t, children } = this.props;
    if (this.state.hasError) {
      return (
        <div className="flex min-h-screen flex-col items-center justify-center gap-4 bg-background px-4 text-center">
          <span className="flex size-12 items-center justify-center rounded-full bg-danger/10 text-danger">
            <AlertOctagon className="size-6" />
          </span>
          <div className="space-y-1">
            <h1 className="text-lg font-semibold text-foreground">{t("error.boundary_title")}</h1>
            <p className="max-w-sm text-sm text-muted-foreground">{t("error.boundary_desc")}</p>
          </div>
          <Button variant="outline" onClick={() => window.location.reload()}>
            {t("error.reload")}
          </Button>
        </div>
      );
    }
    return children;
  }
}

export const ErrorBoundary = withTranslation()(ErrorBoundaryBase);
