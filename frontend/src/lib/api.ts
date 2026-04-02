export type ApiMeta = {
  total?: number;
  offset?: number;
  limit?: number;
  filters?: Record<string, unknown>;
};

export type ApiEnvelope<T> = {
  data: T;
  meta?: ApiMeta;
};

type ApiErrorEnvelope = {
  error?: {
    code?: string;
    message?: string;
  };
};

type RequestOptions = {
  method?: string;
  body?: unknown;
  token?: string | null;
};

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080/api/v1";
export const HEALTH_CHECK_URL = new URL("/health", API_BASE_URL).toString();

export async function apiRequest<T>(
  path: string,
  options: RequestOptions = {},
): Promise<ApiEnvelope<T>> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    method: options.method ?? "GET",
    headers: {
      "Content-Type": "application/json",
      ...(options.token ? { Authorization: `Bearer ${options.token}` } : {}),
    },
    body: options.body ? JSON.stringify(options.body) : undefined,
  });

  const payload = (await response.json().catch(() => ({}))) as ApiEnvelope<T> & ApiErrorEnvelope;
  if (!response.ok) {
    throw new Error(payload.error?.message ?? "Request failed");
  }

  return payload;
}
