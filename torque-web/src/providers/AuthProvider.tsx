import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";
import { get, post } from "@/api/client";
import { queryClient } from "@/providers/QueryProvider";
import { router } from "@/routes";
import type { SessionBundle } from "@/contracts/manual";

// ---------------------------------------------------------------------------
// Context shape
// ---------------------------------------------------------------------------

interface AuthContextValue {
  session: SessionBundle | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

// ---------------------------------------------------------------------------
// Mock session — used when backend is not available
// ---------------------------------------------------------------------------

const MOCK_SESSION: SessionBundle = {
  user: {
    id: "usr_mock_001",
    email: "fabio@siderurgicaaurora.com.br",
    displayName: "Fabio Milennials",
  },
  org: {
    id: "org_mock_001",
    name: "Siderurgica Aurora",
    slug: "siderurgica-aurora",
    planId: "plan_enterprise",
    paymentStatus: "active",
    logoUrl: null,
  },
  role: "admin",
  isMaster: false,
  featurePermissions: {
    "pipeline.view": true,
    "pipeline.edit": true,
    "inbox.view": true,
    "inbox.reply": true,
    "workflows.view": true,
    "workflows.edit": true,
    "campaigns.view": true,
    "campaigns.edit": true,
    "copilot.view": true,
    "copilot.configure": true,
    "analytics.view": true,
    "settings.view": true,
    "settings.edit": true,
  },
  quotas: {
    leads: { effectiveLimit: 10_000, currentUsage: 1_247, canAdd: true },
    members: { effectiveLimit: 25, currentUsage: 8, canAdd: true },
    workflows: { effectiveLimit: 50, currentUsage: 12, canAdd: true },
    campaigns: { effectiveLimit: 20, currentUsage: 3, canAdd: true },
  },
};

// ---------------------------------------------------------------------------
// Navigation helper — uses the router instance directly since AuthProvider
// lives above RouterProvider in the component tree.
// ---------------------------------------------------------------------------

function navigateToLogin(): void {
  void router.navigate("/login", { replace: true });
}

// ---------------------------------------------------------------------------
// Provider
// ---------------------------------------------------------------------------

interface AuthProviderProps {
  children: ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps) {
  const [session, setSession] = useState<SessionBundle | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    async function bootstrap() {
      try {
        const data = await get<SessionBundle>("/auth/me");
        if (!cancelled) {
          setSession(data);
        }
      } catch {
        // Backend not available — fall back to mock
        if (!cancelled) {
          setSession(MOCK_SESSION);
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    void bootstrap();

    // Listen for forced logout from client.ts 401 handler
    const handleForceLogout = () => {
      setSession(null);
      queryClient.clear();
      navigateToLogin();
    };

    window.addEventListener("auth:logout", handleForceLogout);

    return () => {
      cancelled = true;
      window.removeEventListener("auth:logout", handleForceLogout);
    };
  }, []);

  const logout = useCallback(async () => {
    try {
      await post("/auth/logout");
    } catch {
      // Best-effort — clear local state regardless
    }
    setSession(null);
    queryClient.clear();
    navigateToLogin();
  }, []);

  const value: AuthContextValue = {
    session,
    isLoading,
    isAuthenticated: session !== null,
    logout,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within <AuthProvider>");
  }
  return ctx;
}
