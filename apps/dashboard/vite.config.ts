import react from "@vitejs/plugin-react";
import { defineConfig, loadEnv } from "vite";

function loadShortEnvAliases(mode: string) {
	const alias =
		mode === "development" ? "dev" : mode === "production" ? "prod" : "";
	if (!alias) {
		return;
	}

	// Vite supports .env.development/.env.production by default. We also support
	// the short human-friendly forms (.env.dev/.env.prod) as an alias by loading
	// them early and merging into process.env (without overriding).
	const env = loadEnv(alias, process.cwd(), "VITE_");
	for (const [key, value] of Object.entries(env)) {
		if (process.env[key] === undefined) {
			process.env[key] = value;
		}
	}
}

export default defineConfig(({ mode }) => {
	loadShortEnvAliases(mode);
	return {
		plugins: [react()],
		build: {
			rollupOptions: {
				output: {
					manualChunks(id) {
						if (!id.includes("node_modules")) {
							return undefined;
						}
						if (id.includes("/react/") || id.includes("/react-dom/")) {
							return "react";
						}
						if (id.includes("/@mantine/core/")) {
							return "mantine-core";
						}
						if (id.includes("/@mantine/notifications/")) {
							return "mantine-notifications";
						}
						if (id.includes("/@mantine/hooks/")) {
							return "mantine-hooks";
						}
						if (id.includes("/@tabler/icons-react/")) {
							return "tabler";
						}
						return "vendor";
					},
				},
			},
		},
		server: {
			// Keep local dev deterministic: OAuth callback redirects use 127.0.0.1 by
			// default, and binding Vite to localhost can be IPv6-only on some setups.
			host: "127.0.0.1",
			port: 5173,
			strictPort: true,
		},
	};
});
