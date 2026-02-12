import type { components } from './api-types';

// The generated schema marks all response fields as optional, but the API
// always returns them. DeepRequired makes the full tree non-optional.
type DeepRequired<T> = {
	[K in keyof T]-?: NonNullable<T[K]> extends (infer U)[]
		? DeepRequired<U>[]
		: NonNullable<T[K]> extends object
			? DeepRequired<NonNullable<T[K]>>
			: NonNullable<T[K]>;
};

// Types derived from auto-generated OpenAPI schema
export type HealthResponse = components['schemas']['api.HealthResponse'];
export type ProfileResponse = DeepRequired<components['schemas']['behavior.ProfileResponse']>;
export type PlanResponse = DeepRequired<components['schemas']['behavior.PlanResponse']>;
export type TaskResponse = DeepRequired<components['schemas']['behavior.TaskResponse']>;
export type ExecutionLogResponse = DeepRequired<
	components['schemas']['behavior.ExecutionLogResponse']
>;
export type AdherenceResponse = DeepRequired<
	components['schemas']['behavior.AdherenceResponse']
>;
export type AdaptationLogResponse = DeepRequired<
	components['schemas']['behavior.AdaptationLogResponse']
>;
export type AnalyticsResponse = DeepRequired<
	components['schemas']['behavior.AnalyticsResponse']
>;

// Input types use Record<string, never> for JSON fields in the generated schema,
// so we define them manually to accept actual data.
type CreateProfileInput = {
	goals?: Record<string, unknown>;
	constraints?: Record<string, unknown>;
	psychological_state?: Record<string, unknown>;
	difficulty_level?: number;
};
type UpdateProfileInput = {
	goals?: Record<string, unknown>;
	constraints?: Record<string, unknown>;
	psychological_state?: Record<string, unknown>;
	difficulty_level?: number;
	onboarding_completed?: boolean;
};
type LogExecutionInput = components['schemas']['behavior.LogExecutionInput'];

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

export async function fetchProfile(): Promise<ApiResponse<ProfileResponse>> {
	try {
		const res = await fetch(`${BASE_URL}/behavior/profile`);
		if (res.status === 404) return { data: null, error: 'not_found' };
		if (!res.ok) throw new Error(`HTTP ${res.status}`);
		const data: ProfileResponse = await res.json();
		return { data, error: null };
	} catch (e) {
		return { data: null, error: e instanceof Error ? e.message : 'Unknown error' };
	}
}

export async function createProfile(
	input: CreateProfileInput
): Promise<ApiResponse<ProfileResponse>> {
	try {
		const res = await fetch(`${BASE_URL}/behavior/profile`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(input)
		});
		if (!res.ok) {
			const payload = await res.json();
			throw new Error(payload.error ?? `HTTP ${res.status}`);
		}
		const data: ProfileResponse = await res.json();
		return { data, error: null };
	} catch (e) {
		return { data: null, error: e instanceof Error ? e.message : 'Unknown error' };
	}
}

export async function updateProfile(
	input: UpdateProfileInput
): Promise<ApiResponse<ProfileResponse>> {
	try {
		const res = await fetch(`${BASE_URL}/behavior/profile`, {
			method: 'PUT',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(input)
		});
		if (!res.ok) {
			const payload = await res.json();
			throw new Error(payload.error ?? `HTTP ${res.status}`);
		}
		const data: ProfileResponse = await res.json();
		return { data, error: null };
	} catch (e) {
		return { data: null, error: e instanceof Error ? e.message : 'Unknown error' };
	}
}

export async function generatePlan(): Promise<ApiResponse<PlanResponse>> {
	try {
		const res = await fetch(`${BASE_URL}/behavior/plan/generate`, { method: 'POST' });
		if (!res.ok) {
			const payload = await res.json();
			throw new Error(payload.error ?? `HTTP ${res.status}`);
		}
		const data: PlanResponse = await res.json();
		return { data, error: null };
	} catch (e) {
		return { data: null, error: e instanceof Error ? e.message : 'Unknown error' };
	}
}

export async function fetchTodaysPlan(): Promise<ApiResponse<PlanResponse>> {
	try {
		const res = await fetch(`${BASE_URL}/behavior/plan/today`);
		if (res.status === 404) return { data: null, error: 'not_found' };
		if (!res.ok) throw new Error(`HTTP ${res.status}`);
		const data: PlanResponse = await res.json();
		return { data, error: null };
	} catch (e) {
		return { data: null, error: e instanceof Error ? e.message : 'Unknown error' };
	}
}

export async function fetchTodaysLogs(): Promise<ApiResponse<ExecutionLogResponse[]>> {
	try {
		const res = await fetch(`${BASE_URL}/behavior/plan/today/logs`);
		if (!res.ok) throw new Error(`HTTP ${res.status}`);
		const data: ExecutionLogResponse[] = await res.json();
		return { data, error: null };
	} catch (e) {
		return { data: null, error: e instanceof Error ? e.message : 'Unknown error' };
	}
}

export async function logExecution(
	taskId: string,
	input: LogExecutionInput
): Promise<ApiResponse<ExecutionLogResponse>> {
	try {
		const res = await fetch(`${BASE_URL}/behavior/tasks/${taskId}/log`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(input)
		});
		if (!res.ok) {
			const payload = await res.json();
			throw new Error(payload.error ?? `HTTP ${res.status}`);
		}
		const data: ExecutionLogResponse = await res.json();
		return { data, error: null };
	} catch (e) {
		return { data: null, error: e instanceof Error ? e.message : 'Unknown error' };
	}
}

export async function fetchAdherence(): Promise<ApiResponse<AdherenceResponse>> {
	try {
		const res = await fetch(`${BASE_URL}/behavior/adherence`);
		if (!res.ok) throw new Error(`HTTP ${res.status}`);
		const data: AdherenceResponse = await res.json();
		return { data, error: null };
	} catch (e) {
		return { data: null, error: e instanceof Error ? e.message : 'Unknown error' };
	}
}

export async function fetchAnalytics(): Promise<ApiResponse<AnalyticsResponse>> {
	try {
		const res = await fetch(`${BASE_URL}/behavior/analytics`);
		if (!res.ok) throw new Error(`HTTP ${res.status}`);
		const data: AnalyticsResponse = await res.json();
		return { data, error: null };
	} catch (e) {
		return { data: null, error: e instanceof Error ? e.message : 'Unknown error' };
	}
}

export async function fetchAdaptations(): Promise<ApiResponse<AdaptationLogResponse[]>> {
	try {
		const res = await fetch(`${BASE_URL}/behavior/adaptations`);
		if (!res.ok) throw new Error(`HTTP ${res.status}`);
		const data: AdaptationLogResponse[] = await res.json();
		return { data, error: null };
	} catch (e) {
		return { data: null, error: e instanceof Error ? e.message : 'Unknown error' };
	}
}
