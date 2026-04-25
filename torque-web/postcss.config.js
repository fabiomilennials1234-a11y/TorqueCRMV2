// Tailwind v4: novo plugin '@tailwindcss/postcss' substitui o pacote
// 'tailwindcss' direto que v3 usava. autoprefixer mantido.
export default {
  plugins: {
    '@tailwindcss/postcss': {},
    autoprefixer: {},
  },
}
