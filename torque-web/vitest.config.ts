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
        // Current measured (S31): lines 48.95 / stmts 46.75 / funcs 41.79
        // / branches 43.18. Floor sits slightly under current so the first
        // CI run does not flake; every future sprint RATCHETS up.
        // Target by S40: 70 / 65 / 60 / 55.
        lines: 45,
        statements: 45,
        functions: 40,
        branches: 40,
      },
    },
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
})
