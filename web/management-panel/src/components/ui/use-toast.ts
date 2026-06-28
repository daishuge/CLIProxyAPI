import { create } from "zustand";

export type ToastTone = "default" | "success" | "warning" | "danger" | "info";

export interface ToastRecord {
  id: string;
  title?: string;
  description?: string;
  tone: ToastTone;
  duration: number;
}

export type ToastInput = Omit<ToastRecord, "id" | "tone" | "duration"> &
  Partial<Pick<ToastRecord, "tone" | "duration">>;

interface ToastStore {
  toasts: ToastRecord[];
  push: (input: ToastInput) => string;
  dismiss: (id: string) => void;
  clear: () => void;
}

const DEFAULT_DURATION = 5000;

function makeId(): string {
  return `toast-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

/**
 * Headless toast queue. The `Toaster` component subscribes to render entries;
 * feature code calls `toast(...)` to enqueue notifications without prop drilling.
 */
export const useToastStore = create<ToastStore>((set) => ({
  toasts: [],
  push: (input) => {
    const id = makeId();
    const record: ToastRecord = {
      id,
      tone: input.tone ?? "default",
      duration: input.duration ?? DEFAULT_DURATION,
      ...(input.title !== undefined ? { title: input.title } : {}),
      ...(input.description !== undefined ? { description: input.description } : {}),
    };
    set((state) => ({ toasts: [...state.toasts, record] }));
    return id;
  },
  dismiss: (id) => set((state) => ({ toasts: state.toasts.filter((t) => t.id !== id) })),
  clear: () => set({ toasts: [] }),
}));

/** Imperative toast helper usable outside React (e.g. API interceptors). */
export const toast = Object.assign(
  (input: ToastInput) => useToastStore.getState().push(input),
  {
    success: (title: string, description?: string) =>
      useToastStore.getState().push({ tone: "success", title, ...(description ? { description } : {}) }),
    error: (title: string, description?: string) =>
      useToastStore.getState().push({ tone: "danger", title, ...(description ? { description } : {}) }),
    warning: (title: string, description?: string) =>
      useToastStore.getState().push({ tone: "warning", title, ...(description ? { description } : {}) }),
    info: (title: string, description?: string) =>
      useToastStore.getState().push({ tone: "info", title, ...(description ? { description } : {}) }),
  },
);

/** Hook returning the live toast list and queue controls. */
export function useToast() {
  const toasts = useToastStore((s) => s.toasts);
  const dismiss = useToastStore((s) => s.dismiss);
  return { toasts, toast, dismiss };
}
