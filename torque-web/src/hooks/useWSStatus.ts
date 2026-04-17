import { useWS } from "@/providers/WSProvider";
import type { WSStatus } from "@/lib/ws";

/**
 * Returns the current WebSocket connection status.
 * Safe to use outside WSProvider — defaults to "disconnected".
 */
export function useWSStatus(): WSStatus {
  const { status } = useWS();
  return status;
}
