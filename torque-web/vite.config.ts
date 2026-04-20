import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'node:path'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { '@': path.resolve(__dirname, 'src') },
  },
  server: { port: 5173, strictPort: true },
  build: {
    target: 'es2022',
    sourcemap: true,
    rollupOptions: {
      output: {
        // Manual chunks isolate large, rarely-changing third-party code so
        // the user only redownloads app code on redeploy. v8 ships 6 chunks;
        // we target 6+ after S30 (react/radix/motion/query/dnd/sentry).
        manualChunks: {
          'react-vendor': ['react', 'react-dom', 'react-router-dom'],
          'radix-vendor': [
            '@radix-ui/react-dialog',
            '@radix-ui/react-dropdown-menu',
            '@radix-ui/react-tooltip',
            '@radix-ui/react-popover',
            '@radix-ui/react-tabs',
          ],
          'motion-vendor': ['motion'],
          'query-vendor': ['@tanstack/react-query'],
          // dnd-vendor reserved for when F07 WorkflowCanvas wires @xyflow
          // (S43) — leave commented until the dep lands to avoid Rollup
          // warnings.
          'sentry-vendor': ['@sentry/react'],
        },
      },
    },
  },
})
