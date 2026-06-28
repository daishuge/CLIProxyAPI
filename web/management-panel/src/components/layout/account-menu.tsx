import { LogOut, ShieldCheck, UserCircle2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useAuthStore } from "@/lib/auth";
import {
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui";

/** Account dropdown with the masked key fingerprint and sign-out action. */
export function AccountMenu() {
  const { t } = useTranslation();
  const managementKey = useAuthStore((s) => s.managementKey);
  const logout = useAuthStore((s) => s.logout);

  const fingerprint = managementKey
    ? `${managementKey.slice(0, 3)}…${managementKey.slice(-3)}`
    : "—";

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" aria-label={t("header.account")}>
          <UserCircle2 className="size-5" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="min-w-52">
        <DropdownMenuLabel className="flex flex-col gap-1 normal-case">
          <span className="text-xs font-medium text-muted-foreground">{t("account.signed_in")}</span>
          <span className="flex items-center gap-1.5 text-sm font-medium text-foreground">
            <ShieldCheck className="size-3.5 text-success" />
            <span className="font-mono">{fingerprint}</span>
          </span>
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuItem destructive onSelect={() => logout()}>
          <LogOut className="size-4" />
          {t("account.logout")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
