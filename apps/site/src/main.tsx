import { colorSchemeManager, siteTheme } from "@mamusiabtw/web-ui";
import { MantineProvider } from "@mantine/core";
import React from "react";
import ReactDOM from "react-dom/client";
import { SiteApp } from "./site";
import "@mantine/core/styles.css";
import "@mamusiabtw/web-ui/styles.css";
import "./styles.css";

const root = document.getElementById("root");
if (!root) {
	throw new Error('Missing root element with id="root".');
}

ReactDOM.createRoot(root).render(
	<React.StrictMode>
		<MantineProvider
			theme={siteTheme}
			colorSchemeManager={colorSchemeManager}
			defaultColorScheme="auto"
		>
			<SiteApp />
		</MantineProvider>
	</React.StrictMode>,
);
