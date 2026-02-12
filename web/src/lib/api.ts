import type { components } from './api-types';

// Export useful types
export type HealthResponse = components['schemas']['api.HealthResponse'];

// Type-safe API client
const BASE_URL = '/api';

type ApiResponse<T> = { data: T; error: null } | { data: null; error: string };

export type ProfileResponse = {
	id: string;
	user_id: string;
	goals: unknown;
	constraints: unknown;
	psychological_state: unknown;
	difficulty_level: number;
	onboarding_completed: boolean;
	created_at: string;
	updated_at: string;
};

export type PlanResponse = {
	id: string;
	user_id: string;
	plan_date: string;
	difficulty_score: number;
	status: string;
	tasks: TaskResponse[];
	created_at: string;
	updated_at: string;
};

export type TaskResponse = {
	id: string;
	title: string;
	description: string;
	category: string;
	difficulty: number;
	position: number;
	created_at: string;
};

export type ExecutionLogResponse = {
	id: string;
	plan_task_id: string;
	user_id: string;
	completed: boolean;
	completed_at?: string;
	notes: string;
	created_at: string;
};

export type AdherenceResponse = {
	id: string;
	user_id: string;
	completion_rate: number;
	streak_count: number;
	total_tasks: number;
	completed_tasks: number;
	difficulty_mismatch: boolean;
	last_computed_at: string;
};

export type AdaptationLogResponse = {
	id: string;
	user_id: string;
	adaptation_reason: string;
	difficulty_change: number;
	previous_difficulty: number;
	new_difficulty: number;
	trigger_metrics: unknown;
	created_at: string;
};

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

export async function createProfile(input: {
	goals?: unknown;
	constraints?: unknown;
	psychological_state?: unknown;
	difficulty_level?: number;
}): Promise<ApiResponse<ProfileResponse>> {
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

export async function updateProfile(input: {
	goals?: unknown;
	constraints?: unknown;
	psychological_state?: unknown;
	difficulty_level?: number;
	onboarding_completed?: boolean;
}): Promise<ApiResponse<ProfileResponse>> {
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
	input: { completed: boolean; notes?: string }
): Promise<ApiResponse<ExecutionLogResponse>> {
	try {
		const res = await fetch(`${BASE_URL}/behavior/tasks/${taskId}/log`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ completed: input.completed, notes: input.notes ?? '' })
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
