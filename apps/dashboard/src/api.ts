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
