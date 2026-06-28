/**
 * Build version, injected by Vite from `package.json` at compile time.
 * `__APP_VERSION__` is declared in `vite.config.ts`'s `define` and typed in
 * `src/vite-env.d.ts`.
 */
export const APP_VERSION: string =
  typeof __APP_VERSION__ === "string" ? __APP_VERSION__ : "0.0.0";
