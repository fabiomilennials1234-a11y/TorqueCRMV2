import { useAuth } from "@/providers/AuthProvider";
import type { SessionBundle } from "@/contracts/manual";

interface UseSessionReturn {
  user: SessionBundle["user"];
  org: SessionBundle["org"];
  role: SessionBundle["role"];
  isMaster: boolean;
  isAuthenticated: boolean;
  isLoading: boolean;
}

export function useSession(): UseSessionReturn {
  const { session, isAuthenticated, isLoading } = useAuth();

  if (!session) {
    // Safe default while loading or unauthenticated
    return {
      user: { id: "", email: "", displayName: "" },
      org: {
        id: "",
        name: "",
        slug: "",
        planId: null,
        paymentStatus: "active",
        logoUrl: null,
      },
      role: "membro",
      isMaster: false,
      isAuthenticated,
      isLoading,
    };
  }

  return {
    user: session.user,
    org: session.org,
    role: session.role,
    isMaster: session.isMaster,
    isAuthenticated,
    isLoading,
  };
}
