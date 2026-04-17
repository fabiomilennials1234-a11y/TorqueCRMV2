import type { Config } from "tailwindcss";
import animate from "tailwindcss-animate";

const config: Config = {
  darkMode: "class",
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    container: { center: true, padding: "1.5rem" },
    extend: {
      colors: {
        bg: "hsl(var(--bg) / <alpha-value>)",
        surface: "hsl(var(--surface) / <alpha-value>)",
        elevated: "hsl(var(--elevated) / <alpha-value>)",
        hairline: "hsl(var(--hairline) / <alpha-value>)",
        ink: "hsl(var(--ink) / <alpha-value>)",
        "ink-muted": "hsl(var(--ink-muted) / <alpha-value>)",
        "ink-dim": "hsl(var(--ink-dim) / <alpha-value>)",
        accent: "hsl(var(--accent) / <alpha-value>)",
        "accent-soft": "hsl(var(--accent-soft) / <alpha-value>)",
        success: "hsl(var(--success) / <alpha-value>)",
        warning: "hsl(var(--warning) / <alpha-value>)",
        danger: "hsl(var(--danger) / <alpha-value>)",
        info: "hsl(var(--info) / <alpha-value>)",
        heat: {
          1: "hsl(var(--heat-1) / <alpha-value>)",
          2: "hsl(var(--heat-2) / <alpha-value>)",
          3: "hsl(var(--heat-3) / <alpha-value>)",
          4: "hsl(var(--heat-4) / <alpha-value>)",
          5: "hsl(var(--heat-5) / <alpha-value>)",
        },
        channel: {
          whatsapp: "hsl(var(--channel-whatsapp) / <alpha-value>)",
          messenger: "hsl(var(--channel-messenger) / <alpha-value>)",
          instagram: "hsl(var(--channel-instagram) / <alpha-value>)",
          sz: "hsl(var(--channel-sz) / <alpha-value>)",
        },
        countdown: {
          safe: "hsl(var(--countdown-safe) / <alpha-value>)",
          warn: "hsl(var(--countdown-warn) / <alpha-value>)",
          urgent: "hsl(var(--countdown-urgent) / <alpha-value>)",
        },
        job: {
          pending: "hsl(var(--job-pending) / <alpha-value>)",
          running: "hsl(var(--job-running) / <alpha-value>)",
          completed: "hsl(var(--job-completed) / <alpha-value>)",
          failed: "hsl(var(--job-failed) / <alpha-value>)",
        },
      },
      fontFamily: {
        display: ['"DM Serif Display"', "ui-serif", "Georgia", "serif"],
        sans: ["Inter", "ui-sans-serif", "system-ui", "sans-serif"],
        mono: ['"JetBrains Mono"', "ui-monospace", "Menlo", "monospace"],
      },
      fontSize: {
        "2xs": ["0.6875rem", { lineHeight: "1rem", letterSpacing: "0.04em" }],
      },
      letterSpacing: {
        tightest: "-0.035em",
      },
      borderRadius: {
        xs: "3px",
        sm: "5px",
        DEFAULT: "8px",
        md: "10px",
        lg: "14px",
        xl: "20px",
      },
      boxShadow: {
        hairline: "inset 0 0 0 1px hsl(var(--hairline) / 0.6)",
        "hairline-b": "inset 0 -1px 0 0 hsl(var(--hairline) / 0.6)",
        "elev-1":
          "0 1px 0 0 hsl(var(--hairline) / 0.4), 0 2px 8px -2px rgb(0 0 0 / 0.08), 0 1px 2px -1px rgb(0 0 0 / 0.06)",
        "elev-2":
          "0 1px 0 0 hsl(var(--hairline) / 0.4), 0 8px 24px -8px rgb(0 0 0 / 0.12), 0 2px 6px -2px rgb(0 0 0 / 0.06)",
        "elev-3":
          "0 1px 0 0 hsl(var(--hairline) / 0.4), 0 24px 64px -16px rgb(0 0 0 / 0.2), 0 4px 12px -4px rgb(0 0 0 / 0.08)",
        glow: "0 0 0 1px hsl(var(--accent) / 0.4), 0 0 24px -4px hsl(var(--accent) / 0.2)",
        /* Glass / neumorphism shadows */
        "glass-sm":
          "0 1px 2px 0 rgb(0 0 0 / 0.04), 0 1px 3px 0 rgb(0 0 0 / 0.02)",
        "glass-md":
          "0 4px 16px -2px rgb(0 0 0 / 0.06), 0 1px 2px 0 rgb(0 0 0 / 0.03)",
        "neu-hover":
          "0 4px 16px -4px rgb(0 0 0 / 0.08), inset 0 1px 0 0 hsl(var(--hairline) / 0.3)",
        "neu-focus":
          "0 0 0 1px hsl(var(--accent) / 0.5), 0 0 12px -2px hsl(var(--accent) / 0.15)",
        "btn-primary":
          "0 1px 0 0 hsl(44 93% 68% / 0.5) inset, 0 -1px 0 0 hsl(44 93% 38% / 0.5) inset, 0 4px 12px -4px hsl(var(--accent) / 0.35)",
        /* Claymorphism — soft 3D depth for premium feel */
        "clay-1":
          "6px 6px 16px -1px rgb(0 0 0 / var(--clay-depth, 0.25)), -2px -2px 12px -1px rgb(255 255 255 / var(--clay-highlight, 0.02)), inset 0 1px 1px 0 rgb(255 255 255 / var(--clay-inset, 0.04)), inset 0 -1px 1px 0 rgb(0 0 0 / 0.1)",
        "clay-2":
          "8px 8px 24px -2px rgb(0 0 0 / var(--clay-depth, 0.3)), -3px -3px 16px -2px rgb(255 255 255 / var(--clay-highlight, 0.03)), inset 0 1px 2px 0 rgb(255 255 255 / var(--clay-inset, 0.06)), inset 0 -1px 2px 0 rgb(0 0 0 / 0.12)",
        "clay-3":
          "12px 12px 32px -4px rgb(0 0 0 / 0.35), -4px -4px 20px -4px rgb(255 255 255 / 0.04), inset 0 2px 3px 0 rgb(255 255 255 / 0.07), inset 0 -2px 3px 0 rgb(0 0 0 / 0.15)",
        "clay-pressed":
          "inset 2px 2px 6px 0 rgb(0 0 0 / 0.2), inset -1px -1px 4px 0 rgb(255 255 255 / 0.03), 0 1px 2px 0 rgb(0 0 0 / 0.05)",
        "clay-glow":
          "6px 6px 16px -1px rgb(0 0 0 / 0.25), -2px -2px 12px -1px rgb(255 255 255 / 0.02), inset 0 1px 1px 0 rgb(255 255 255 / 0.04), inset 0 -1px 1px 0 rgb(0 0 0 / 0.1), 0 0 20px -6px hsl(var(--accent) / 0.15)",
      },
      backgroundImage: {
        grid: "linear-gradient(hsl(var(--hairline) / 0.5) 1px, transparent 1px), linear-gradient(90deg, hsl(var(--hairline) / 0.5) 1px, transparent 1px)",
        "dot-grid":
          "radial-gradient(hsl(var(--hairline)) 1px, transparent 1px)",
        "fade-top":
          "linear-gradient(to bottom, hsl(var(--bg)) 0%, transparent 100%)",
        "fade-bottom":
          "linear-gradient(to top, hsl(var(--bg)) 0%, transparent 100%)",
      },
      keyframes: {
        "fade-in": {
          from: { opacity: "0", transform: "translateY(4px)" },
          to: { opacity: "1", transform: "translateY(0)" },
        },
        "scale-in": {
          from: { opacity: "0", transform: "scale(0.96)" },
          to: { opacity: "1", transform: "scale(1)" },
        },
        shimmer: {
          "0%": { backgroundPosition: "-200% 0" },
          "100%": { backgroundPosition: "200% 0" },
        },
        "torque-tick": {
          "0%, 100%": { transform: "scale(1)" },
          "50%": { transform: "scale(1.04)" },
        },
        "caret-blink": {
          "0%, 70%, 100%": { opacity: "1" },
          "20%, 50%": { opacity: "0" },
        },
      },
      animation: {
        "fade-in": "fade-in 280ms cubic-bezier(0.2, 0.8, 0.2, 1)",
        "scale-in": "scale-in 180ms cubic-bezier(0.2, 0.8, 0.2, 1)",
        shimmer: "shimmer 1.2s linear infinite",
        "torque-tick": "torque-tick 280ms cubic-bezier(0.65, 0, 0.35, 1)",
        "caret-blink":
          "caret-blink 1.06s cubic-bezier(0.65, 0, 0.35, 1) infinite",
      },
    },
  },
  plugins: [animate],
};

export default config;
