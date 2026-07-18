import { defineConfig, loadEnv, type UserConfig } from "vite";
import react from "@vitejs/plugin-react";
import { resolve } from "node:path";

export default defineConfig(({ mode }): UserConfig => {
  const env =  loadEnv(mode, resolve(__dirname, ".."), ["PAGE", "VITE"]);

  return {
    plugins: [react()],
    server: {
      port: env.VITE_PORT ? Number(env.VITE_PORT) : undefined,
      proxy: {
        "/api": env.VITE_PROXY_HOST ?? "http://localhost:8010",
      },
    },
  };
});
