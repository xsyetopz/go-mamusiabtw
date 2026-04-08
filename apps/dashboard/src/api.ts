// The dashboard is served from the admin API origin, so we always call relative
// /api/... paths. This avoids all CORS and cookie host issues.
export const apiBase = "";
export const apiBaseError: string | null = null;

export class APIError extends Error {
	readonly status: number;

	constructor(message: string, status: number) {
		super(message);
		this.status = status;
	}
}

function looksLikeHTML(text: string): boolean {
	const head = text.trimStart().slice(0, 64).toLowerCase();
	return head.startsWith("<!doctype") || head.startsWith("<html");
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
	let response: Response;
	try {
		response = await fetch(path, {
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

	// Sometimes a misconfigured origin/proxy returns HTML (e.g. Vite index.html)
	// for an API route, which causes confusing "Unexpected token '<'" errors.
	const clone = response.clone();
	try {
		return (await response.json()) as T;
	} catch {
		let body = "";
		try {
			body = await clone.text();
		} catch {
			// ignore
		}
		if (looksLikeHTML(body)) {
			throw new APIError(
				"This URL is returning HTML for an API request. You are not talking to the mamusiabtw admin API. Start `go run ./cmd/mamusiabtw dev` and open the dashboard at `http://127.0.0.1:8081/` (or the `dashboard_url` it prints).",
				-1,
			);
		}
		throw new APIError("Admin API returned invalid JSON.", -1);
	}
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
