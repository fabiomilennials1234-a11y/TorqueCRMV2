import { useAuth } from "@/providers/AuthProvider";

/**
 * Direct read of a feature permission flag from the session.
 * Returns false if unauthenticated or the key is absent.
 */
export function usePermission(featureKey: string): boolean {
  const { session } = useAuth();

  if (!session) return false;

  return session.featurePermissions[featureKey] === true;
}
