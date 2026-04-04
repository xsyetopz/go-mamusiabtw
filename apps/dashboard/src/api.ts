const configuredAPIBase =
	(import.meta.env["VITE_ADMIN_API_BASE_URL"] as string | undefined)?.trim() ||
	"";
const fallbackDevAPIBase = "http://127.0.0.1:8081";
const isProductionBuild = import.meta.env.PROD;
const TRAILING_SLASH_RE = /\/$/;

function normalizeAPIBase(raw: string): { base: string; error: string | null } {
	if (raw.trim() === "") {
		return {
			base: "",
			error: isProductionBuild
				? "VITE_ADMIN_API_BASE_URL is required for production builds."
				: null,
		};
	}
	try {
		const url = new URL(raw);
		if (url.protocol !== "http:" && url.protocol !== "https:") {
			return { base: raw, error: "Admin API URL must use http or https." };
		}
		url.pathname = "";
		url.search = "";
		url.hash = "";
		return { base: url.toString().replace(TRAILING_SLASH_RE, ""), error: null };
	} catch {
		return { base: raw, error: "Admin API URL is not a valid absolute URL." };
	}
}

const normalized = normalizeAPIBase(
	configuredAPIBase || (isProductionBuild ? "" : fallbackDevAPIBase),
);

export const apiBase = normalized.base;
export const apiBaseError = normalized.error;

export class APIError extends Error {
	readonly status: number;

	constructor(message: string, status: number) {
		super(message);
		this.status = status;
	}
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
	if (apiBaseError) {
		throw new APIError(apiBaseError, 0);
	}

	let response: Response;
	try {
		response = await fetch(apiBase + path, {
			credentials: "include",
			...init,
			headers: {
				"Content-Type": "application/json",
				...(init?.headers ?? {}),
			},
		});
	} catch {
		throw new APIError("Could not reach the admin API.", 0);
	}

	if (!response.ok) {
		let message = response.statusText;
		try {
			const payload = (await response.json()) as { error?: string };
			if (payload.error) {
				message = payload.error;
			}
		} catch {
			// ignore body parse failures
		}
		throw new APIError(message, response.status);
	}

	if (response.status === 204) {
		return undefined as T;
	}
	return (await response.json()) as T;
}

export function get<T>(path: string): Promise<T> {
	return request<T>(path, { method: "GET" });
}

export function post<T>(
	path: string,
	body: unknown,
	csrfToken?: string,
): Promise<T> {
	const headers = csrfToken ? { "X-CSRF-Token": csrfToken } : {};
	return request<T>(path, {
		method: "POST",
		body: JSON.stringify(body),
		headers,
	});
}

export function put<T>(
	path: string,
	body: unknown,
	csrfToken: string,
): Promise<T> {
	return request<T>(path, {
		method: "PUT",
		body: JSON.stringify(body),
		headers: { "X-CSRF-Token": csrfToken },
	});
}
