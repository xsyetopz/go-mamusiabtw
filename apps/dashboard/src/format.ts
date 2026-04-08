export function badgeColor(value: boolean): string {
	return value ? "success" : "gray";
}

export function prettyDate(value: string): string {
	if (!value) {
		return "Unknown";
	}
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) {
		return value;
	}
	return date.toLocaleString();
}

export function kindLabel(kind: string): string {
	switch (kind) {
		case "core_builtin":
			return "Built-in";
		case "official_plugin":
			return "Official plugin";
		case "user_plugin":
			return "User plugin";
		default:
			return kind;
	}
}
