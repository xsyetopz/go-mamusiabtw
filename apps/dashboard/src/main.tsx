import {
	createTheme,
	localStorageColorSchemeManager,
	MantineProvider,
} from "@mantine/core";
import { Notifications } from "@mantine/notifications";
import React from "react";
import ReactDOM from "react-dom/client";
import { App } from "./App";
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
