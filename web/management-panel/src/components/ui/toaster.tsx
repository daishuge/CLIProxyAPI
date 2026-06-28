import {
  Toast,
  ToastClose,
  ToastDescription,
  ToastProvider,
  ToastTitle,
  ToastViewport,
} from "./toast";
import { useToast, type ToastTone } from "./use-toast";

const toneToVariant: Record<ToastTone, "default" | "success" | "warning" | "danger" | "info"> = {
  default: "default",
  success: "success",
  warning: "warning",
  danger: "danger",
  info: "info",
};

/** Mounts the toast viewport once near the app root. */
export function Toaster() {
  const { toasts, dismiss } = useToast();
  return (
    <ToastProvider swipeDirection="right">
      {toasts.map((t) => (
        <Toast
          key={t.id}
          variant={toneToVariant[t.tone]}
          duration={t.duration}
          onOpenChange={(open) => {
            if (!open) dismiss(t.id);
          }}
        >
          {t.title ? <ToastTitle>{t.title}</ToastTitle> : null}
          {t.description ? <ToastDescription>{t.description}</ToastDescription> : null}
          <ToastClose />
        </Toast>
      ))}
      <ToastViewport />
    </ToastProvider>
  );
}
