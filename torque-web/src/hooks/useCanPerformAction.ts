import { useAuth } from "@/providers/AuthProvider";

/**
 * Determines if the current user can perform a given action.
 *
 * Cascade:
 * 1. Master users can do everything.
 * 2. Admin users can do everything.
 * 3. Otherwise, check featurePermissions for the specific action key.
 * 4. Default: false.
 */
export function useCanPerformAction(action: string): boolean {
  const { session } = useAuth();

  if (!session) return false;
  if (session.isMaster) return true;
  if (session.role === "admin") return true;

  return session.featurePermissions[action] === true;
}
