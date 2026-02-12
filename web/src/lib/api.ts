import type { components } from './api-types';

// Export useful types
export type HealthResponse = components['schemas']['api.HealthResponse'];

// Type-safe API client
const BASE_URL = '/api';

type ApiResponse<T> = { data: T; error: null } | { data: null; error: string };

export async function fetchHealth(): Promise<ApiResponse<HealthResponse>> {
	try {
		const res = await fetch(`${BASE_URL}/health`);
		if (!res.ok) throw new Error(`HTTP ${res.status}`);
		const data: HealthResponse = await res.json();
		return { data, error: null };
	} catch (e) {
		return { data: null, error: e instanceof Error ? e.message : 'Unknown error' };
	}
}
