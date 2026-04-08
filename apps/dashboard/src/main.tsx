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
import { ErrorBoundary } from "./components/ErrorBoundary";
import { DeveloperDetailsProvider } from "./developerDetails";
import "@mantine/core/styles.css";
import "@mantine/notifications/styles.css";
import "./styles.css";

const brandBlue = [
	"#edf4ff",
	"#d0e2ff",
	"#a6c8ff",
	"#78a9ff",
	"#4589ff",
	"#0f62fe", // IBM Blue 60
	"#0043ce",
	"#002d9c",
	"#001d6c",
	"#001141",
] as const;

const wongSuccess = [
	"#e6faf3",
	"#c2f0e0",
	"#96e6cb",
	"#62dbb4",
	"#2ccf9c",
	"#009e73", // Wong bluish green
	"#00865f",
	"#006c4d",
	"#00523b",
	"#003727",
] as const;

const wongWarning = [
	"#fff5e5",
	"#ffe8c2",
	"#ffd799",
	"#ffc270",
	"#ffad47",
	"#e69f00", // Wong orange
	"#c98800",
	"#9f6d00",
	"#755000",
	"#4d3300",
] as const;

const wongDanger = [
	"#ffece2",
	"#ffd7c2",
	"#ffbd99",
	"#ffa06f",
	"#ff7f47",
	"#d55e00", // Wong vermillion
	"#b44f00",
	"#8f3f00",
	"#6a2f00",
	"#401c00",
] as const;

const theme = createTheme({
	colors: {
		brand: brandBlue,
		success: wongSuccess,
		warning: wongWarning,
		danger: wongDanger,
	},
	primaryColor: "brand",
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
		Text: {
			// Mantine Text defaults to <p>, which cannot contain some Mantine
			// components that render block-level <div> (hydration warnings).
			defaultProps: {
				component: "div",
			},
		},
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
// TS control-flow narrowing does not always carry into nested async functions.
const rootEl = root;

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
	ReactDOM.createRoot(rootEl).render(
		<React.StrictMode>
			<MantineProvider
				theme={theme}
				colorSchemeManager={colorSchemeManager}
				defaultColorScheme="auto"
			>
				<Notifications />
				<DeveloperDetailsProvider>
					<ErrorBoundary>
						<App />
					</ErrorBoundary>
				</DeveloperDetailsProvider>
			</MantineProvider>
		</React.StrictMode>,
	);
}

bootstrap().catch(() => {
	setAPIBaseError("Dashboard failed to start.");
});
