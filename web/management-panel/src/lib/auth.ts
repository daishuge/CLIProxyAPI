import { create } from "zustand";
import {
  ApiError,
  resolveBaseUrl,
  MANAGEMENT_PREFIX,
  setTokenProvider,
  setUnauthorizedHandler,
} from "./api";

/** localStorage key holding the management key. */
export const MGMT_KEY_STORAGE = "ppap.mgmtKey";

function readStoredKey(): string | null {
  try {
    return window.localStorage.getItem(MGMT_KEY_STORAGE);
  } catch {
    return null;
  }
}

function persistKey(key: string | null): void {
  try {
    if (key) window.localStorage.setItem(MGMT_KEY_STORAGE, key);
    else window.localStorage.removeItem(MGMT_KEY_STORAGE);
  } catch {
    // Storage may be unavailable (private mode); auth still works in-memory.
  }
}

/**
 * Probe an authenticated endpoint with a candidate key directly, bypassing the
 * global token provider. This prevents background queries from picking up an
 * unverified key and triggering spurious 401 logouts.
 */
async function probeWithKey(candidateKey: string): Promise<void> {
  const url = `${resolveBaseUrl()}${MANAGEMENT_PREFIX}/config`;
  let response: Response;
  try {
    response = await fetch(url, {
      headers: { Authorization: `Bearer ${candidateKey}` },
    });
  } catch {
    throw new ApiError({
      message: "Network request failed",
      status: 0,
      kind: "network",
    });
  }
  if (!response.ok) {
    let parsed: unknown;
    try {
      parsed = await response.json();
    } catch {
      // ignore
    }
    const msg: string =
      parsed && typeof parsed === "object" && "message" in parsed && typeof (parsed as Record<string, unknown>).message === "string"
        ? ((parsed as Record<string, string>).message as string)
        : response.status === 401
          ? "Invalid management key"
          : `Probe failed with status ${response.status}`;
    throw new ApiError({
      message: msg,
      status: response.status,
      kind: response.status === 401 ? "unauthorized" : response.status >= 500 ? "server" : "unknown",
    });
  }
}

interface AuthState {
  /** The active management key, or null when logged out. */
  managementKey: string | null;
  /** True once a key is present (optimistic; verified via probe on login). */
  isAuthenticated: boolean;
  /**
   * Validate a candidate key by probing an authenticated endpoint, then persist
   * it on success. Throws `ApiError` on failure so the caller can show messages.
   */
  login: (key: string) => Promise<void>;
  /** Clear the key from memory + storage. */
  logout: () => void;
}

export const useAuthStore = create<AuthState>((set, get) => {
  const initialKey = readStoredKey();
  return {
    managementKey: initialKey,
    isAuthenticated: !!initialKey,
    login: async (key: string) => {
      const trimmed = key.trim();
      // Probe with the candidate key directly — the global token provider and
      // unauthorized handler remain untouched, so background queries keep using
      // the current (valid) key and won't trigger spurious 401 logouts.
      await probeWithKey(trimmed);
      persistKey(trimmed);
      set({ managementKey: trimmed, isAuthenticated: true });
    },
    logout: () => {
      persistKey(null);
      set({ managementKey: null, isAuthenticated: false });
      void get; // keep `get` referenced for future extension without lint noise
    },
  };
});

/**
 * Bind the API client to the auth store: the client reads the current key for
 * the Authorization header and forces logout when any request returns 401.
 * Called once during app bootstrap.
 */
export function bindAuthToClient(): void {
  setTokenProvider(() => useAuthStore.getState().managementKey);
  setUnauthorizedHandler(() => {
    const state = useAuthStore.getState();
    if (state.isAuthenticated) {
      state.logout();
    }
  });
}
