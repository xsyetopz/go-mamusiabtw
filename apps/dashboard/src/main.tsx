import { createTheme, MantineProvider } from "@mantine/core";
import { Notifications } from "@mantine/notifications";
import React from "react";
import ReactDOM from "react-dom/client";
import { App } from "./App";
import "@mantine/core/styles.css";
import "@mantine/notifications/styles.css";
import "./styles.css";

const theme = createTheme({
	primaryColor: "teal",
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

const root = document.getElementById("root");
if (!root) {
	throw new Error('Missing root element with id="root".');
}

ReactDOM.createRoot(root).render(
	<React.StrictMode>
		<MantineProvider theme={theme} defaultColorScheme="light">
			<Notifications />
			<App />
		</MantineProvider>
	</React.StrictMode>,
);
