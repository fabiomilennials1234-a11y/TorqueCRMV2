import { useQuery } from '@tanstack/react-query'
import { COCKPIT_MOCK } from '@/features/cockpit/__fixtures__/cockpit-mock'
import type { TaskCockpitBundle } from '@/contracts/manual'

export const COCKPIT_QUERY_KEY = ['cockpit', 'me'] as const

/**
 * Hidrata o cockpit do vendedor logado em uma única chamada.
 * Hoje retorna fixture. Quando o backend existir, troca para:
 *   () => get<TaskCockpitBundle>("/tasks/me/cockpit")
 * (transformer snake↔camel em src/api/tasks.ts).
 */
export function useCockpit() {
  return useQuery<TaskCockpitBundle>({
    queryKey: COCKPIT_QUERY_KEY,
    queryFn: async () => {
      await new Promise((r) => setTimeout(r, 180))
      // Retorna cópia para não mutar fixture entre ciclos.
      return structuredClone(COCKPIT_MOCK)
    },
    staleTime: 60_000,
    refetchOnWindowFocus: false,
  })
}
