import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        ink: {
          950: "#06080d",
          900: "#0b1018",
          850: "#101722",
          800: "#141d2a"
        },
        signal: {
          cyan: "#35d3ff",
          blue: "#5f8cff",
          green: "#45e0a8",
          amber: "#f5b84b",
          red: "#ff6b76"
        }
      },
      boxShadow: {
        glow: "0 0 30px rgba(53, 211, 255, 0.18)"
      }
    }
  },
  plugins: []
};

export default config;
