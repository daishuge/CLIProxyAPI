import { create } from "zustand";
import { fetchConfig } from "./api";
import { setTokenProvider, setUnauthorizedHandler } from "./api";

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
      // Temporarily expose the candidate key so the probe request is authorized.
      set({ managementKey: trimmed });
      try {
        await fetchConfig();
        persistKey(trimmed);
        set({ isAuthenticated: true });
      } catch (err) {
        // Roll back to the previously persisted key on failure.
        const previous = readStoredKey();
        set({ managementKey: previous, isAuthenticated: !!previous });
        throw err;
      }
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
