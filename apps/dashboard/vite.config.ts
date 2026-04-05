import fs from "node:fs";
import path from "node:path";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

function forbidNonStandardDotenv() {
	const cwd = process.cwd();
	const forbidden = [
		".env",
		".env.local",
		".env.development",
		".env.production",
		".env.production.local",
		".env.dev.local",
		".env.prod.local",
	];
	for (const name of forbidden) {
		const full = path.join(cwd, name);
		if (fs.existsSync(full)) {
			throw new Error(
				`Forbidden env file ${name} detected. Use only .env.dev (dev) or .env.prod (prod).`,
			);
		}
	}
}

export default defineConfig(({ mode }) => {
	if (mode !== "dev" && mode !== "prod") {
		throw new Error(
			`Unsupported Vite mode ${JSON.stringify(mode)}. Use --mode dev or --mode prod.`,
		);
	}
	forbidNonStandardDotenv();
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
			// Keep local dev deterministic: bind to IPv4 loopback.
			host: "127.0.0.1",
			port: 5173,
			strictPort: true,
		},
	};
});
