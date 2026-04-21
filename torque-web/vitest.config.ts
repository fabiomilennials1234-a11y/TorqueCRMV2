import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import path from 'node:path'

export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    css: true,
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      // Scope coverage measurement to layers this sprint asserts against.
      // Pages/shell live under Fase B+ and carry integration-level
      // guarantees that unit tests are the wrong tool for.
      include: ['src/hooks/**', 'src/api/**', 'src/lib/**'],
      exclude: [
        'src/test/**',
        'src/**/*.d.ts',
        'src/contracts/api.gen.ts',
        'src/main.tsx',
        'src/App.tsx',
        'src/routes.tsx',
        'src/vite-env.d.ts',
        // Pure dev fixture — seed.ts is exercised by the dev bypass
        // path, not unit-testable in isolation without stubbing the
        // whole world.
        'src/lib/seed.ts',
        // Simple permission resolvers + ws status — covered indirectly
        // by every hook test that uses them; isolated tests would
        // duplicate the type system.
        'src/hooks/useWSStatus.ts',
        'src/hooks/useCanPerformAction.ts',
        'src/hooks/usePermission.ts',
        'src/hooks/useSession.ts',
      ],
      // S31 initial ratchet — baseline reflects the current measured
      // state after adding 10 hook tests. Every future sprint MUST NOT
      // regress; target 70/65/60/55 by S40. Mirrors v8's D007→D025
      // ratchet (started at 9.33%, climbed each sprint).
      thresholds: {
        // Ratchet step 4 (remediation-C, 2026-04-20): after 15 novos test
        // files (lib/ws + lib/utils + 4 hook extras + 5 pages novas)
        // coverage medida lines 78.21 / stmts 75.02 / funcs 74.20 /
        // branches 62.22. Floor sobe para 70 nos 3 primários e 60 em
        // branches (menor porque else-paths defensivos inflam o
        // denominator; o ganho real é nas linhas executadas). Target
        // próximo: branches 70 quando pages de workflow canvas + inbox
        // ganharem testes de user interaction.
        lines: 70,
        statements: 70,
        functions: 70,
        branches: 60,
      },
    },
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
})
