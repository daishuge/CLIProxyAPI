import { useAuthStore } from "@/lib/auth";
import { MainLayout } from "@/components/layout/main-layout";
import { LoginPage } from "@/pages/login-page";
import { AppRouter } from "./app-router";

/**
 * Top-level gate: unauthenticated users see the login screen; authenticated
 * users get the full shell. Acts as the `ProtectedRoute` for the whole app
 * since every feature route lives behind the same management key.
 */
export function App() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);

  if (!isAuthenticated) {
    return <LoginPage />;
  }

  return (
    <MainLayout>
      <AppRouter />
    </MainLayout>
  );
}
