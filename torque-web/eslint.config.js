import js from '@eslint/js'
import globals from 'globals'
import tseslint from 'typescript-eslint'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import jsxA11y from 'eslint-plugin-jsx-a11y'
import importPlugin from 'eslint-plugin-import'
import prettierConfig from 'eslint-config-prettier'

export default tseslint.config(
  {
    ignores: [
      'dist',
      'node_modules',
      'coverage',
      'vite.config.ts',
      'vitest.config.ts',
      'eslint.config.js',
      'postcss.config.js',
      'tailwind.config.ts',
      'src/contracts/api.gen.ts',
      'src/test/**',
      'src/**/__tests__/**',
      'src/**/*.test.ts',
      'src/**/*.test.tsx',
    ],
  },
  {
    extends: [
      js.configs.recommended,
      ...tseslint.configs.recommendedTypeChecked,
      jsxA11y.flatConfigs.recommended,
      prettierConfig,
    ],
    files: ['src/**/*.{ts,tsx}'],
    languageOptions: {
      ecmaVersion: 2023,
      globals: { ...globals.browser, ...globals.es2023 },
      parserOptions: {
        project: ['./tsconfig.json'],
        tsconfigRootDir: import.meta.dirname,
        ecmaFeatures: { jsx: true },
      },
    },
    plugins: {
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
      import: importPlugin,
    },
    rules: {
      ...reactHooks.configs.recommended.rules,
      'react-hooks/exhaustive-deps': 'error',

      // Hard ban: no XSS escape hatch.
      'no-restricted-syntax': [
        'error',
        {
          selector: "JSXAttribute[name.name='dangerouslySetInnerHTML']",
          message:
            'dangerouslySetInnerHTML is banned. Use refs + textContent or DOMPurify if truly required.',
        },
      ],

      'no-console': ['warn', { allow: ['warn', 'error'] }],
      '@typescript-eslint/no-floating-promises': 'error',
      '@typescript-eslint/no-explicit-any': 'error',
      '@typescript-eslint/consistent-type-imports': [
        'error',
        { prefer: 'type-imports', fixStyle: 'inline-type-imports' },
      ],

      // Import hygiene — force @/ alias over deep relative paths.
      'import/no-relative-parent-imports': 'warn',

      '@typescript-eslint/no-misused-promises': [
        'error',
        { checksVoidReturn: { attributes: false } },
      ],
      '@typescript-eslint/no-unused-vars': [
        'error',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_' },
      ],

      // HMR hint only — off because provider files intentionally co-export
      // hooks and components (useAuth + AuthProvider, useTheme + ThemeProvider, etc).
      'react-refresh/only-export-components': 'off',

      // Off — the reconciliation patterns we use (syncing derived state when
      // parent prop changes, resetting input on id change) are legitimate.
      // React's own docs still describe these as acceptable use of effects.
      'react-hooks/set-state-in-effect': 'off',

      // async wrapper is intentional for forward-compatibility with `await patch(...)`.
      '@typescript-eslint/require-await': 'off',
    },
  }
)
