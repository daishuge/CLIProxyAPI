import * as React from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useTranslation } from "react-i18next";
import { Eye, EyeOff, KeyRound, Globe } from "lucide-react";
import { useAuthStore } from "@/lib/auth";
import { ApiError, resolveBaseUrl } from "@/lib/api";
import { LogoMark } from "@/components/brand/logo";
import { ThemeToggle } from "@/components/layout/theme-toggle";
import { LanguageMenu } from "@/components/layout/language-menu";
import {
  Button,
  Card,
  CardContent,
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  Input,
} from "@/components/ui";

const loginSchema = z.object({
  managementKey: z.string().min(1, { message: "login.error_required" }),
});

type LoginValues = z.infer<typeof loginSchema>;

/** Maps an ApiError to a localized login error key. */
function errorKeyFor(error: unknown): string {
  if (error instanceof ApiError) {
    switch (error.kind) {
      case "unauthorized":
        return "login.error_unauthorized";
      case "forbidden":
        return "login.error_forbidden";
      case "not_found":
        return "login.error_not_found";
      case "network":
        return "login.error_network";
      case "timeout":
        return "login.error_timeout";
      case "server":
        return "login.error_server";
      default:
        return "login.error_generic";
    }
  }
  return "login.error_generic";
}

export function LoginPage() {
  const { t } = useTranslation();
  const login = useAuthStore((s) => s.login);
  const [showKey, setShowKey] = React.useState(false);
  const [submitError, setSubmitError] = React.useState<string | null>(null);

  const form = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: { managementKey: "" },
  });

  const onSubmit = async (values: LoginValues) => {
    setSubmitError(null);
    try {
      await login(values.managementKey);
    } catch (err) {
      setSubmitError(errorKeyFor(err));
    }
  };

  const origin = resolveBaseUrl() || window.location.origin;

  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden bg-background px-4">
      <div className="pointer-events-none absolute inset-0 bg-grid opacity-[0.4]" aria-hidden />
      <div
        className="pointer-events-none absolute -top-32 left-1/2 h-96 w-[40rem] -translate-x-1/2 rounded-full bg-brand-500/20 blur-3xl"
        aria-hidden
      />

      <div className="absolute right-4 top-4 flex items-center gap-1">
        <LanguageMenu />
        <ThemeToggle />
      </div>

      <Card className="relative z-10 w-full max-w-md border-border/80 shadow-lg">
        <CardContent className="p-8">
          <div className="mb-7 flex flex-col items-center text-center">
            <LogoMark className="size-12" />
            <h1 className="mt-4 text-lg font-semibold tracking-tight text-foreground">
              {t("app.name")}
            </h1>
            <p className="mt-1.5 max-w-xs text-sm text-muted-foreground">{t("login.subtitle")}</p>
          </div>

          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-5" noValidate>
              <FormField
                control={form.control}
                name="managementKey"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel required>{t("login.key_label")}</FormLabel>
                    <FormControl>
                      <div className="relative">
                        <KeyRound className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                        <Input
                          {...field}
                          type={showKey ? "text" : "password"}
                          autoComplete="off"
                          autoFocus
                          placeholder={t("login.key_placeholder")}
                          className="pl-9 pr-10 font-mono"
                          invalid={!!form.formState.errors.managementKey}
                        />
                        <button
                          type="button"
                          onClick={() => setShowKey((v) => !v)}
                          aria-label={showKey ? t("login.hide_key") : t("login.show_key")}
                          className="absolute right-2 top-1/2 -translate-y-1/2 rounded-sm p-1 text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40"
                        >
                          {showKey ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                        </button>
                      </div>
                    </FormControl>
                    <FormMessage>
                      {form.formState.errors.managementKey?.message
                        ? t(form.formState.errors.managementKey.message)
                        : null}
                    </FormMessage>
                  </FormItem>
                )}
              />

              <div className="flex items-center gap-2 rounded-md border border-border bg-surface-sunken/50 px-3 py-2 text-xs text-muted-foreground">
                <Globe className="size-3.5 shrink-0" />
                <span className="truncate font-mono">{origin}</span>
              </div>

              {submitError ? (
                <p
                  role="alert"
                  className="rounded-md border border-danger/40 bg-danger/[0.06] px-3 py-2 text-sm font-medium text-danger"
                >
                  {t(submitError)}
                </p>
              ) : null}

              <Button type="submit" className="w-full" loading={form.formState.isSubmitting}>
                {form.formState.isSubmitting ? t("login.submitting") : t("login.submit")}
              </Button>
            </form>
          </Form>
        </CardContent>
      </Card>
    </div>
  );
}
