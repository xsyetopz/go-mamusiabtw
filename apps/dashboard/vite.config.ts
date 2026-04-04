import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

export default defineConfig({
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
		port: 5173,
		strictPort: true,
	},
});
