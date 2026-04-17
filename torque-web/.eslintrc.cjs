module.exports = {
  root: true,
  env: { browser: true, es2023: true },
  parser: "@typescript-eslint/parser",
  parserOptions: { ecmaVersion: "latest", sourceType: "module", ecmaFeatures: { jsx: true } },
  ignorePatterns: ["dist", ".eslintrc.cjs", "vite.config.ts"],
};
