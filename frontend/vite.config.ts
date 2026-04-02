import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import { loadEnv, type Plugin } from "vite";

type AppVersionManifest = {
  version: string;
  builtAt: string;
};

export default defineConfig(({ mode }) => {
  const appVersion = resolveAppVersion(mode);

  return {
    build: {
      rollupOptions: {
        output: {
          manualChunks,
        },
      },
    },
    define: {
      __APP_VERSION__: JSON.stringify(appVersion.version),
      __APP_BUILD_TIME__: JSON.stringify(appVersion.builtAt),
    },
    plugins: [react(), appVersionPlugin(appVersion)],
    server: {
      host: true,
      port: 5173,
    },
    test: {
      environment: "jsdom",
      setupFiles: "./src/test/setup.ts",
      testTimeout: 15000,
      coverage: {
        provider: "v8",
        reporter: ["text", "html", "json-summary"],
        reportsDirectory: "../coverage/frontend",
        all: true,
        include: ["src/**/*.{ts,tsx}"],
        exclude: ["src/test/**", "src/vite-env.d.ts"],
      },
    },
  };
});

function resolveAppVersion(mode: string): AppVersionManifest {
  const env = loadEnv(mode, process.cwd(), "");
  const packageJSON = JSON.parse(readFileSync(resolve(process.cwd(), "package.json"), "utf8")) as {
    version?: string;
  };
  const builtAt = new Date().toISOString();
  const defaultVersion = `${packageJSON.version ?? "0.0.0"}-${builtAt
    .replaceAll(/[-:.]/g, "")
    .replace("T", "-")
    .replace("Z", "")}`;
  const declaredVersion = process.env.VITE_APP_VERSION || env.VITE_APP_VERSION;

  return {
    version: declaredVersion?.trim() || defaultVersion,
    builtAt,
  };
}

function appVersionPlugin(manifest: AppVersionManifest): Plugin {
  const source = JSON.stringify(manifest, null, 2);

  return {
    name: "app-version-manifest",
    configureServer(server) {
      server.middlewares.use("/version.json", (_request, response) => {
        writeVersionResponse(response, source);
      });
    },
    configurePreviewServer(server) {
      server.middlewares.use("/version.json", (_request, response) => {
        writeVersionResponse(response, source);
      });
    },
    generateBundle() {
      this.emitFile({
        type: "asset",
        fileName: "version.json",
        source,
      });
    },
  };
}

function writeVersionResponse(
  response: {
    setHeader: (name: string, value: string) => void;
    end: (body?: string) => void;
  },
  source: string,
) {
  response.setHeader("Content-Type", "application/json; charset=utf-8");
  response.setHeader("Cache-Control", "no-store, no-cache, must-revalidate");
  response.end(source);
}

function manualChunks(id: string) {
  const normalizedID = id.replaceAll("\\", "/");

  if (!normalizedID.includes("/node_modules/")) {
    return undefined;
  }

  if (
    normalizedID.includes("/react/") ||
    normalizedID.includes("/react-dom/") ||
    normalizedID.includes("/react-router") ||
    normalizedID.includes("/scheduler/")
  ) {
    return "react-runtime";
  }

  if (normalizedID.includes("/@fortawesome/")) {
    return "fontawesome";
  }

  if (normalizedID.includes("/sonner/")) {
    return "feedback";
  }

  if (normalizedID.includes("/yaml/")) {
    return "i18n";
  }

  return "vendor";
}
