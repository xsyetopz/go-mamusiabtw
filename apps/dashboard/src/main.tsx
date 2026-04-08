import {
	createTheme,
	localStorageColorSchemeManager,
	MantineProvider,
} from "@mantine/core";
import { Notifications } from "@mantine/notifications";
import React from "react";
import ReactDOM from "react-dom/client";
import { App } from "./App";
import { setAPIBase, setAPIBaseError } from "./api";
import "@mantine/core/styles.css";
import "@mantine/notifications/styles.css";
import "./styles.css";

const goBlue = [
	"#e6f7fc",
	"#cceff8",
	"#a2e3f2",
	"#79d6eb",
	"#4fc9e4",
	"#2abde0",
	"#00add8", // Go blue anchor
	"#0096bc",
	"#007a99",
	"#005c73",
] as const;

const theme = createTheme({
	colors: {
		goblue: goBlue,
	},
	primaryColor: "goblue",
	defaultRadius: "md",
	fontFamily:
		'"Iosevka Aile", "IBM Plex Sans", "Segoe UI", -apple-system, BlinkMacSystemFont, sans-serif',
	headings: {
		fontFamily:
			'"Iosevka Aile", "IBM Plex Sans", "Segoe UI", -apple-system, BlinkMacSystemFont, sans-serif',
		fontWeight: "700",
	},
	spacing: {
		xs: "0.5rem",
		sm: "0.75rem",
		md: "1rem",
		lg: "1.25rem",
		xl: "1.75rem",
	},
	components: {
		Button: {
			defaultProps: {
				radius: "md",
			},
		},
		Card: {
			defaultProps: {
				radius: "md",
			},
		},
		Table: {
			defaultProps: {
				verticalSpacing: "xs",
				horizontalSpacing: "sm",
			},
		},
	},
});

const colorSchemeManager = localStorageColorSchemeManager({
	key: "mamusiabtw-color-scheme",
});

const root = document.getElementById("root");
if (!root) {
	throw new Error('Missing root element with id="root".');
}

const TRAILING_SLASH_RE = /\/$/;

async function loadDashboardConfig() {
	try {
		const resp = await fetch("/config.json", { cache: "no-store" });
		if (resp.status === 404) {
			return;
		}
		if (!resp.ok) {
			setAPIBaseError(`Could not load dashboard config (HTTP ${resp.status}).`);
			return;
		}
		const payload = (await resp.json()) as { api_origin?: unknown };
		const raw = String(payload.api_origin ?? "")
			.trim()
			.replace(TRAILING_SLASH_RE, "");
		if (raw === "") {
			return;
		}
		let url: URL;
		try {
			url = new URL(raw);
		} catch {
			setAPIBaseError(
				"Invalid dashboard config: api_origin must be a full URL.",
			);
			return;
		}
		if (url.protocol !== "http:" && url.protocol !== "https:") {
			setAPIBaseError("Invalid dashboard config: api_origin must be http(s).");
			return;
		}
		if (url.pathname !== "/" || url.search !== "" || url.hash !== "") {
			setAPIBaseError(
				"Invalid dashboard config: api_origin must be an origin only.",
			);
			return;
		}
		setAPIBase(url.origin);
	} catch {
		setAPIBaseError("Could not load dashboard config.");
	}
}

async function bootstrap() {
	await loadDashboardConfig();
	ReactDOM.createRoot(root).render(
		<React.StrictMode>
			<MantineProvider
				theme={theme}
				colorSchemeManager={colorSchemeManager}
				defaultColorScheme="auto"
			>
				<Notifications />
				<App />
			</MantineProvider>
		</React.StrictMode>,
	);
}

bootstrap().catch(() => {
	setAPIBaseError("Dashboard failed to start.");
});
