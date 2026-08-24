/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        bg: "#070B14",
        surface: "#0E1524",
        surface2: "#141C2E",
        line: "#1E2A44",
        ink: "#E7EEF8",
        muted: "#8AA0C2",
        amber: "#F5A524",
        cyan: "#3EE0C6",
        rose: "#FF6B8A",
        violet: "#8B7CFF",
      },
      fontFamily: {
        sans: ["Sora", "Plus Jakarta Sans", "system-ui", "sans-serif"],
        mono: ["JetBrains Mono", "ui-monospace", "monospace"],
      },
      boxShadow: {
        glow: "0 0 0 1px rgba(245,165,36,0.25), 0 12px 40px rgba(245,165,36,0.12)",
        card: "0 10px 40px rgba(0,0,0,0.35)",
      },
    },
  },
  plugins: [],
};
