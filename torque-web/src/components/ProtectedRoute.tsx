import { Navigate, Outlet } from "react-router-dom";
import { useAuth } from "@/providers/AuthProvider";

/**
 * Route guard that redirects unauthenticated users to /login.
 * While the session is loading, renders a fullscreen shimmer skeleton.
 */
export function ProtectedRoute() {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="flex h-screen w-screen items-center justify-center bg-bg">
        <div
          className="h-8 w-48 rounded bg-surface bg-[length:200%_100%] bg-gradient-to-r from-surface via-elevated to-surface animate-shimmer"
          aria-label="Carregando..."
        />
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <Outlet />;
}
