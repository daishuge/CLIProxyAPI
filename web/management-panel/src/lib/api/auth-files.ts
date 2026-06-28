/**
 * Auth Files API — stored provider credentials (OAuth token bundles and key
 * files) managed on the proxy node.
 *
 * Backend contract (internal/api/handlers/management/auth_files.go):
 *   GET    /auth-files                 -> { files: AuthFileInfo[] }
 *   POST   /auth-files                  upload: multipart (field "file") or raw
 *                                       JSON body with ?name=<filename>
 *   DELETE /auth-files?name=<name>
 *   GET    /auth-files/download?name=<name>  -> raw file (attachment)
 *   GET    /auth-files/models?name=<name>    -> { models: AuthFileModel[] }
 *   PATCH  /auth-files/status  body: { name, disabled }
 */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { managementApi, resolveBaseUrl, MANAGEMENT_PREFIX } from "./client";
import { queryKeys } from "../query";

/** A stored auth credential as reported by the backend. */
export interface AuthFileInfo {
  id: string;
  auth_index?: string;
  name: string;
  type?: string;
  provider?: string;
  label?: string;
  status?: string;
  status_message?: string;
  disabled?: boolean;
  unavailable?: boolean;
  runtime_only?: boolean;
  source?: string;
  size?: number;
  success?: number;
  failed?: number;
  email?: string;
  project_id?: string;
  account_type?: string;
  account?: string;
  created_at?: string;
  updated_at?: string;
  modtime?: string;
  last_refresh?: string;
  next_retry_after?: string;
  path?: string;
  priority?: number;
  note?: string;
  websockets?: boolean;
  [key: string]: unknown;
}

/** A model exposed by a specific auth file. */
export interface AuthFileModel {
  id: string;
  display_name?: string;
  type?: string;
  owned_by?: string;
}

interface AuthFilesResponse {
  files?: AuthFileInfo[];
}
interface AuthFileModelsResponse {
  models?: AuthFileModel[];
}

/** List all stored auth files. */
export function fetchAuthFiles(signal?: AbortSignal): Promise<AuthFileInfo[]> {
  return managementApi
    .get<AuthFilesResponse>("/auth-files", signal ? { signal } : undefined)
    .then((res) => res.files ?? []);
}

/** Fetch the models a given auth file can serve. */
export function fetchAuthFileModels(
  name: string,
  signal?: AbortSignal,
): Promise<AuthFileModel[]> {
  return managementApi
    .get<AuthFileModelsResponse>("/auth-files/models", {
      query: { name },
      ...(signal ? { signal } : {}),
    })
    .then((res) => res.models ?? []);
}

/** Upload one or more auth files via multipart form. */
export function uploadAuthFiles(files: File[]): Promise<unknown> {
  const form = new FormData();
  for (const file of files) {
    form.append("file", file, file.name);
  }
  return managementApi.post<unknown>("/auth-files", { body: form });
}

/** Delete an auth file by name. */
export function deleteAuthFile(name: string): Promise<void> {
  return managementApi.delete<void>("/auth-files", { query: { name } });
}

/** Toggle an auth file's disabled state. */
export function patchAuthFileStatus(name: string, disabled: boolean): Promise<unknown> {
  return managementApi.patch<unknown>("/auth-files/status", { body: { name, disabled } });
}

/** Build the authenticated download URL for an auth file. Used to open a
 * browser download; the management key is appended via the existing fetch path
 * is not possible for a plain link, so callers stream the blob instead. */
export function authFileDownloadPath(name: string): string {
  const base = resolveBaseUrl();
  const url = new URL(`${base}${MANAGEMENT_PREFIX}/auth-files/download`, base || window.location.href);
  url.searchParams.set("name", name);
  return url.toString();
}

/** Download an auth file as a blob (carries the Bearer token through the client). */
export function downloadAuthFile(name: string, signal?: AbortSignal): Promise<Response> {
  return managementApi.get<Response>("/auth-files/download", {
    query: { name },
    raw: true,
    ...(signal ? { signal } : {}),
  });
}

/** Query hook for the auth files list. */
export function useAuthFilesQuery(enabled = true) {
  return useQuery({
    queryKey: queryKeys.authFiles,
    queryFn: ({ signal }) => fetchAuthFiles(signal),
    enabled,
  });
}

/** Query hook for a single auth file's models (lazy: enable when selected). */
export function useAuthFileModelsQuery(name: string | null) {
  return useQuery({
    queryKey: queryKeys.authFileModels(name ?? ""),
    queryFn: ({ signal }) => fetchAuthFileModels(name!, signal),
    enabled: !!name,
    staleTime: 60_000,
  });
}

/** Mutation hook to upload auth files. */
export function useUploadAuthFilesMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (files: File[]) => uploadAuthFiles(files),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.authFiles });
    },
  });
}

/** Mutation hook to delete an auth file. */
export function useDeleteAuthFileMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (name: string) => deleteAuthFile(name),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.authFiles });
    },
  });
}

/** Mutation hook to toggle an auth file's disabled state. */
export function useToggleAuthFileStatusMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ name, disabled }: { name: string; disabled: boolean }) =>
      patchAuthFileStatus(name, disabled),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.authFiles });
    },
  });
}
