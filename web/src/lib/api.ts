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

// Types derived from auto-generated OpenAPI schema, extended with new fields
// that may not yet be in the generated api-types.ts.
export type HealthResponse = components['schemas']['api.HealthResponse'];
export type ProfileResponse = DeepRequired<components['schemas']['behavior.ProfileResponse']> & {
	delivery_mode: string;
	emotional_state: string;
	ai_profile_summary: Record<string, unknown>;
};
export type TaskResponse = DeepRequired<components['schemas']['behavior.TaskResponse']> & {
	task_type: string;
	suggested_time: string;
	anchor_label: string;
};
export type PlanResponse = Omit<
	DeepRequired<components['schemas']['behavior.PlanResponse']>,
	'tasks'
> & {
	tasks: TaskResponse[];
};
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
	delivery_mode?: string;
	emotional_state?: string;
	ai_profile_summary?: Record<string, unknown>;
};
type UpdateProfileInput = {
	goals?: Record<string, unknown>;
	constraints?: Record<string, unknown>;
	psychological_state?: Record<string, unknown>;
	difficulty_level?: number;
	onboarding_completed?: boolean;
	delivery_mode?: string;
	emotional_state?: string;
	ai_profile_summary?: Record<string, unknown>;
};
type LogExecutionInput = components['schemas']['behavior.LogExecutionInput'];

// Onboarding chat types
export type OnboardingChatMessage = {
	role: 'user' | 'assistant';
	content: string;
};

export type OnboardingChatResponse = {
	ai_message: string;
	history: OnboardingChatMessage[];
	turn_number: number;
	is_complete: boolean;
	summary?: Record<string, unknown>;
	recommended_delivery_mode?: string;
	recommended_emotional_state?: string;
};

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

export async function onboardingChat(
	userMessage: string,
	history: OnboardingChatMessage[],
	context: {
		goals?: Record<string, unknown>;
		constraints?: Record<string, unknown>;
		psychological_state?: Record<string, unknown>;
	}
): Promise<ApiResponse<OnboardingChatResponse>> {
	try {
		const res = await fetch(`${BASE_URL}/behavior/onboarding/chat`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				user_message: userMessage,
				history,
				goals: context.goals ?? {},
				constraints: context.constraints ?? {},
				psychological_state: context.psychological_state ?? {}
			})
		});
		if (!res.ok) {
			const payload = await res.json();
			throw new Error(payload.error ?? `HTTP ${res.status}`);
		}
		const data: OnboardingChatResponse = await res.json();
		return { data, error: null };
	} catch (e) {
		return { data: null, error: e instanceof Error ? e.message : 'Unknown error' };
	}
}
