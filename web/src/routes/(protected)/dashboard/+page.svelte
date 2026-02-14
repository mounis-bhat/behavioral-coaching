<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { onMount } from 'svelte';
	import { user } from '$lib/stores/auth';
	import { SvelteSet } from 'svelte/reactivity';
	import {
		fetchProfile,
		fetchTodaysPlan,
		fetchTodaysLogs,
		fetchAdherence,
		fetchAdaptations,
		generatePlan,
		logExecution
	} from '$lib/api';
	import type {
		AdaptationLogResponse,
		AdherenceResponse,
		PlanResponse,
		ProfileResponse,
		TaskResponse
	} from '$lib/api-types';

	let pageLoading = $state(true);
	let error = $state<string | null>(null);

	let profile = $state<ProfileResponse | null>(null);
	let plan = $state<PlanResponse | null>(null);
	let adherence = $state<AdherenceResponse | null>(null);
	let adaptations = $state<AdaptationLogResponse[]>([]);
	const completedTasks = new SvelteSet<string>();

	let generating = $state(false);
	let togglingTask = $state<string | null>(null);
	let logoutLoading = $state(false);
	let relapseNoticeDismissed = $state(false);

	let deliveryMode = $derived(profile?.delivery_mode ?? 'flexible_list');

	let showRelapseNotice = $derived(
		!relapseNoticeDismissed &&
			adaptations.length > 0 &&
			adaptations[0].adaptation_reason.startsWith('Automatic relapse recovery')
	);

	// For gentle_structure mode: split tasks into anchors and regular
	let anchorTasks = $derived(
		plan?.tasks.filter((t: TaskResponse) => t.task_type === 'anchor') ?? []
	);
	let regularTasks = $derived(
		plan?.tasks.filter((t: TaskResponse) => t.task_type !== 'anchor') ?? []
	);

	const anchorLabels: Record<string, string> = {
		wake_window: 'Morning Anchor',
		activation_activity: 'Midday Anchor',
		sleep_winddown: 'Evening Anchor'
	};

	onMount(async () => {
		const profileResult = await fetchProfile();
		if (
			profileResult.error === 'not_found' ||
			(profileResult.data && !profileResult.data.onboarding_completed)
		) {
			await goto(resolve('/onboarding'));
			return;
		}
		if (profileResult.error) {
			error = profileResult.error;
			pageLoading = false;
			return;
		}

		profile = profileResult.data;
		await loadData();
		pageLoading = false;
	});

	async function loadData() {
		const [planRes, logsRes, adhRes, adaptRes] = await Promise.all([
			fetchTodaysPlan(),
			fetchTodaysLogs(),
			fetchAdherence(),
			fetchAdaptations()
		]);

		if (planRes.data) plan = planRes.data;
		else plan = null;

		completedTasks.clear();
		if (logsRes.data) {
			for (const log of logsRes.data) {
				if (log.completed) {
					completedTasks.add(log.plan_task_id);
				}
			}
		}

		if (adhRes.data) adherence = adhRes.data;
		if (adaptRes.data) adaptations = adaptRes.data;
	}

	async function handleGenerate() {
		generating = true;
		error = null;
		try {
			const result = await generatePlan();
			if (result.error) throw new Error(result.error);
			plan = result.data;
			await loadData();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to generate plan';
		} finally {
			generating = false;
		}
	}

	async function handleToggleTask(taskId: string) {
		togglingTask = taskId;
		error = null;
		const wasCompleted = completedTasks.has(taskId);
		try {
			// Optimistic update
			if (wasCompleted) completedTasks.delete(taskId);
			else completedTasks.add(taskId);

			const result = await logExecution(taskId, { completed: !wasCompleted });
			if (result.error) throw new Error(result.error);

			// Refresh adherence and adaptations
			const [adhRes, adaptRes] = await Promise.all([fetchAdherence(), fetchAdaptations()]);
			if (adhRes.data) adherence = adhRes.data;
			if (adaptRes.data) adaptations = adaptRes.data;
		} catch (e) {
			// Revert optimistic update
			if (wasCompleted) completedTasks.add(taskId);
			else completedTasks.delete(taskId);
			error = e instanceof Error ? e.message : 'Failed to update task';
		} finally {
			togglingTask = null;
		}
	}

	async function handleLogout() {
		logoutLoading = true;
		try {
			const res = await fetch('/api/auth/logout', { method: 'POST' });
			if (!res.ok) throw new Error('Logout failed');
			user.set(null);
			await goto(resolve('/login'));
		} catch (e) {
			error = e instanceof Error ? e.message : 'Logout failed';
		} finally {
			logoutLoading = false;
		}
	}

	const categoryColors: Record<string, string> = {
		health: 'bg-green-100 text-green-800',
		mindset: 'bg-purple-100 text-purple-800',
		discipline: 'bg-orange-100 text-orange-800',
		relationships: 'bg-blue-100 text-blue-800',
		productivity: 'bg-yellow-100 text-yellow-800'
	};
</script>

{#snippet taskItem(task: TaskResponse)}
	{@const done = completedTasks.has(task.id)}
	<div class="flex items-start gap-3 rounded-lg border p-4 {done ? 'bg-gray-50' : ''}">
		<button
			type="button"
			onclick={() => handleToggleTask(task.id)}
			disabled={togglingTask === task.id}
			class="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded border {done
				? 'border-black bg-black'
				: 'border-gray-400'} disabled:opacity-60"
			aria-label={done ? 'Mark incomplete' : 'Mark complete'}
		>
			{#if done}
				<svg
					class="h-3 w-3 text-white"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="3"
				>
					<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
				</svg>
			{/if}
		</button>
		<div class="min-w-0 flex-1">
			<div class="flex items-baseline gap-2">
				<p class="text-sm font-medium {done ? 'text-gray-400 line-through' : ''}">
					{task.title}
				</p>
				{#if deliveryMode === 'strict_schedule' && task.suggested_time}
					<span class="shrink-0 text-xs font-medium text-gray-500">
						{task.suggested_time}
					</span>
				{/if}
			</div>
			<p class="mt-0.5 text-xs text-gray-500">{task.description}</p>
			<span
				class="mt-1.5 inline-block rounded-full px-2 py-0.5 text-xs font-medium {categoryColors[
					task.category
				] ?? 'bg-gray-100 text-gray-800'}"
			>
				{task.category}
			</span>
		</div>
	</div>
{/snippet}

{#if pageLoading}
	<main class="mx-auto max-w-2xl p-8">
		<p class="text-gray-500">Loading...</p>
	</main>
{:else}
	<main class="mx-auto max-w-2xl p-8">
		<h1 class="mb-6 text-3xl font-bold">Today's Plan</h1>

		{#if error}
			<div class="mb-4 rounded bg-red-100 p-3 text-sm text-red-700">{error}</div>
		{/if}

		{#if showRelapseNotice}
			<div
				class="mb-4 flex items-start justify-between rounded-lg border border-blue-200 bg-blue-50 p-4"
			>
				<p class="text-sm text-blue-800">
					We've adjusted your plan to help you get back on track. Your difficulty has been lowered —
					focus on building momentum.
				</p>
				<button
					type="button"
					onclick={() => (relapseNoticeDismissed = true)}
					class="ml-3 shrink-0 text-blue-400 hover:text-blue-600"
					aria-label="Dismiss notice"
				>
					<svg
						class="h-4 w-4"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
					>
						<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>
		{/if}

		{#if !plan}
			<div class="rounded-lg border border-dashed border-gray-300 p-8 text-center">
				<p class="mb-4 text-gray-600">No plan for today yet.</p>
				<button
					type="button"
					onclick={handleGenerate}
					disabled={generating}
					class="rounded bg-black px-5 py-2 text-white disabled:opacity-60"
				>
					{generating ? 'Generating...' : "Generate today's plan"}
				</button>
			</div>
		{:else if deliveryMode === 'gentle_structure'}
			<!-- Gentle Structure: Anchors section + Tiny tasks section -->
			<section class="mb-8">
				{#if anchorTasks.length > 0}
					<h2 class="mb-3 text-lg font-semibold">Daily Anchors</h2>
					<p class="mb-3 text-sm text-gray-500">
						Your three grounding points for the day. These are your foundation.
					</p>
					<div class="mb-6 space-y-2">
						{#each anchorTasks as task (task.id)}
							<div class="relative">
								{#if task.anchor_label && anchorLabels[task.anchor_label]}
									<span
										class="absolute -top-2 left-3 rounded-full bg-indigo-100 px-2 py-0.5 text-xs font-medium text-indigo-700"
									>
										{anchorLabels[task.anchor_label]}
									</span>
								{/if}
								{@render taskItem(task)}
							</div>
						{/each}
					</div>
				{/if}

				{#if regularTasks.length > 0}
					<h2 class="mb-3 text-lg font-semibold">Tiny Tasks</h2>
					<p class="mb-3 text-sm text-gray-500">Small, gentle actions. Do what feels manageable.</p>
					<div class="space-y-2">
						{#each regularTasks as task (task.id)}
							{@render taskItem(task)}
						{/each}
					</div>
				{/if}
			</section>
		{:else}
			<!-- Flexible List or Strict Schedule: standard task list -->
			<section class="mb-8">
				<div class="space-y-2">
					{#each plan.tasks as task (task.id)}
						{@render taskItem(task)}
					{/each}
				</div>
			</section>
		{/if}

		{#if plan}
			{#if adherence}
				<section class="mb-8">
					<h2 class="mb-3 text-lg font-semibold">Adherence</h2>
					<div class="grid grid-cols-2 gap-3">
						<div class="rounded-lg border p-3">
							<p class="text-xs text-gray-500">Completion</p>
							<p class="text-2xl font-bold">
								{Math.round(adherence.completion_rate * 100)}%
							</p>
						</div>
						<div class="rounded-lg border p-3">
							<p class="text-xs text-gray-500">Streak</p>
							<p class="text-2xl font-bold">
								{adherence.streak_count} day{adherence.streak_count !== 1 ? 's' : ''}
							</p>
						</div>
						<div class="rounded-lg border p-3">
							<p class="text-xs text-gray-500">Tasks done</p>
							<p class="text-2xl font-bold">
								{adherence.completed_tasks}/{adherence.total_tasks}
							</p>
						</div>
						<div class="rounded-lg border p-3">
							<p class="text-xs text-gray-500">Difficulty</p>
							<p class="text-2xl font-bold">{plan.difficulty_score}</p>
						</div>
					</div>
					{#if adherence.difficulty_mismatch}
						<div
							class="mt-3 rounded border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800"
						>
							Your completion rate is low for the current difficulty. The plan may be adjusted soon.
						</div>
					{/if}
				</section>
			{/if}

			{#if adaptations.length > 0}
				<section class="mb-8">
					<h2 class="mb-3 text-lg font-semibold">Plan adjustments</h2>
					<div class="space-y-2">
						{#each adaptations as log (log.id)}
							<div class="rounded-lg border p-3">
								<p class="text-sm">{log.adaptation_reason}</p>
								<p class="mt-1 text-xs text-gray-500">
									Difficulty: {log.previous_difficulty}
									→
									{log.new_difficulty}
									<span
										class="ml-1 {log.difficulty_change > 0 ? 'text-red-600' : 'text-green-600'}"
									>
										({log.difficulty_change > 0 ? '+' : ''}{log.difficulty_change})
									</span>
								</p>
							</div>
						{/each}
					</div>
				</section>
			{/if}
		{/if}

		<div class="mt-8 flex items-center justify-between border-t pt-6">
			<div class="flex gap-4">
				<a href={resolve('/insights')} class="text-sm underline">Insights</a>
				<a href={resolve('/settings')} class="text-sm underline">Settings</a>
			</div>
			<button
				type="button"
				onclick={handleLogout}
				disabled={logoutLoading}
				class="rounded bg-black px-4 py-2 text-sm text-white disabled:opacity-60"
			>
				{logoutLoading ? 'Signing out...' : 'Sign out'}
			</button>
		</div>
	</main>
{/if}
