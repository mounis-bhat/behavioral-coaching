<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { onMount } from 'svelte';
	import { fly } from 'svelte/transition';
	import { cubicOut } from 'svelte/easing';
	import { user } from '$lib/stores/auth';
	import { get } from 'svelte/store';
	import { SvelteSet } from 'svelte/reactivity';
	import {
		fetchProfile,
		fetchTodaysPlan,
		fetchTodaysLogs,
		fetchAdherence,
		fetchAdaptations,
		fetchAnalytics,
		generatePlan,
		logExecution
	} from '$lib/api';
	import type {
		AdaptationLogResponse,
		AdherenceResponse,
		AnalyticsResponse,
		PlanResponse,
		ProfileResponse,
		TaskResponse
	} from '$lib/api';

	let pageLoading = $state(true);
	let error = $state<string | null>(null);
	let visible = $state(false);

	let profile = $state<ProfileResponse | null>(null);
	let plan = $state<PlanResponse | null>(null);
	let adherence = $state<AdherenceResponse | null>(null);
	let adaptations = $state<AdaptationLogResponse[]>([]);
	let analytics = $state<AnalyticsResponse | null>(null);
	const completedTasks = new SvelteSet<string>();

	let generating = $state(false);
	let togglingTask = $state<string | null>(null);
	let logoutLoading = $state(false);
	let relapseNoticeDismissed = $state(false);

	let currentUser = $state(get(user));

	let firstName = $derived(currentUser?.name?.split(' ')[0] ?? '');
	let greeting = $derived.by(() => {
		const h = new Date().getHours();
		if (h < 12) return 'Good morning';
		if (h < 17) return 'Good afternoon';
		return 'Good evening';
	});

	let deliveryMode = $derived(profile?.delivery_mode ?? 'flexible_list');

	let showRelapseNotice = $derived(
		!relapseNoticeDismissed &&
			adaptations.length > 0 &&
			adaptations[0].adaptation_reason.startsWith('Automatic relapse recovery')
	);

	let anchorTasks = $derived(
		plan?.tasks.filter((t: TaskResponse) => t.task_type === 'anchor') ?? []
	);
	let regularTasks = $derived(
		plan?.tasks.filter((t: TaskResponse) => t.task_type !== 'anchor') ?? []
	);

	let completionPct = $derived(adherence ? Math.round(adherence.completion_rate * 100) : 0);

	const anchorLabels: Record<string, string> = {
		wake_window: 'Morning Anchor',
		activation_activity: 'Midday Anchor',
		sleep_winddown: 'Evening Anchor'
	};

	const categoryBarColors: Record<string, string> = {
		health: 'bg-green-400',
		mindset: 'bg-purple-400',
		discipline: 'bg-orange-400',
		relationships: 'bg-blue-400',
		productivity: 'bg-yellow-400'
	};

	const categoryTextColors: Record<string, string> = {
		health: 'text-green-300',
		mindset: 'text-purple-300',
		discipline: 'text-orange-300',
		relationships: 'text-blue-300',
		productivity: 'text-yellow-300'
	};

	const categoryColors: Record<string, string> = {
		health: 'bg-green-400/10 text-green-300 ring-green-400/20',
		mindset: 'bg-purple-400/10 text-purple-300 ring-purple-400/20',
		discipline: 'bg-orange-400/10 text-orange-300 ring-orange-400/20',
		relationships: 'bg-blue-400/10 text-blue-300 ring-blue-400/20',
		productivity: 'bg-yellow-400/10 text-yellow-300 ring-yellow-400/20'
	};

	onMount(() => {
		const unsub = user.subscribe((u) => (currentUser = u));

		(async () => {
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
				visible = true;
				return;
			}

			profile = profileResult.data;
			await loadData();
			pageLoading = false;
			visible = true;
		})();

		return unsub;
	});

	async function loadData() {
		const [planRes, logsRes, adhRes, adaptRes, analyticsRes] = await Promise.all([
			fetchTodaysPlan(),
			fetchTodaysLogs(),
			fetchAdherence(),
			fetchAdaptations(),
			fetchAnalytics()
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
		if (analyticsRes.data) analytics = analyticsRes.data;
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
			if (wasCompleted) completedTasks.delete(taskId);
			else completedTasks.add(taskId);

			const result = await logExecution(taskId, { completed: !wasCompleted });
			if (result.error) throw new Error(result.error);

			const [adhRes, adaptRes, analyticsRes] = await Promise.all([
				fetchAdherence(),
				fetchAdaptations(),
				fetchAnalytics()
			]);
			if (adhRes.data) adherence = adhRes.data;
			if (adaptRes.data) adaptations = adaptRes.data;
			if (analyticsRes.data) analytics = analyticsRes.data;
		} catch (e) {
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

	function formatWeek(dateStr: string): string {
		const d = new Date(dateStr + 'T00:00:00');
		return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
	}

	function formatDate(dateStr: string): string {
		const d = new Date(dateStr + 'T00:00:00');
		return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
	}
</script>

<svelte:head>
	<title>Dashboard — Behavioral Coaching</title>
</svelte:head>

{#snippet taskItem(task: TaskResponse)}
	{@const done = completedTasks.has(task.id)}
	<div
		class="flex items-start gap-3 rounded-xl border p-4 transition {done
			? 'border-white/5 bg-white/[0.02]'
			: 'border-white/10 bg-white/[0.04]'}"
	>
		<button
			type="button"
			onclick={() => handleToggleTask(task.id)}
			disabled={togglingTask === task.id}
			class="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded border transition {done
				? 'border-indigo-400 bg-indigo-500'
				: 'border-white/20 hover:border-white/40'} disabled:opacity-60"
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
				<p
					class="text-sm font-medium transition {done
						? 'text-indigo-300/40 line-through'
						: 'text-white'}"
				>
					{task.title}
				</p>
				{#if deliveryMode === 'strict_schedule' && task.suggested_time}
					<span class="shrink-0 text-xs font-medium text-indigo-200/60">
						{task.suggested_time}
					</span>
				{/if}
			</div>
			<p class="mt-0.5 text-xs text-indigo-100/80">{task.description}</p>
			<span
				class="mt-1.5 inline-block rounded-full px-2 py-0.5 text-xs font-medium ring-1 {categoryColors[
					task.category
				] ?? 'bg-white/5 text-indigo-200/60 ring-white/10'}"
			>
				{task.category}
			</span>
		</div>
	</div>
{/snippet}

<div class="dashboard-bg min-h-screen">
	{#if pageLoading}
		<div class="flex items-center justify-center py-32">
			<div class="flex items-center gap-3 text-indigo-300/60">
				<svg class="h-5 w-5 animate-spin" fill="none" viewBox="0 0 24 24">
					<circle
						class="opacity-25"
						cx="12"
						cy="12"
						r="10"
						stroke="currentColor"
						stroke-width="4"
					/>
					<path
						class="opacity-75"
						fill="currentColor"
						d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
					/>
				</svg>
				<span class="text-sm">Loading your dashboard...</span>
			</div>
		</div>
	{:else if visible}
		<!-- Header -->
		<header class="border-b border-white/[0.06] bg-white/[0.02] py-5">
			<div
				in:fly={{ y: -10, duration: 500, easing: cubicOut }}
				class="mx-auto flex max-w-5xl items-center justify-between px-6"
			>
				<div class="flex items-center gap-4">
					{#if currentUser?.picture}
						<img
							src={currentUser.picture}
							alt=""
							class="h-10 w-10 rounded-full ring-2 ring-indigo-400/30"
							referrerpolicy="no-referrer"
						/>
					{:else}
						<div
							class="flex h-10 w-10 items-center justify-center rounded-full bg-indigo-500/20 text-sm font-semibold text-indigo-300 ring-2 ring-indigo-400/30"
						>
							{firstName ? firstName[0].toUpperCase() : '?'}
						</div>
					{/if}
					<div>
						<h1 class="text-lg font-semibold text-white">
							{greeting}{firstName ? `, ${firstName}` : ''}
						</h1>
						<p class="text-sm text-indigo-200/50">
							{new Date().toLocaleDateString('en-US', {
								weekday: 'long',
								month: 'long',
								day: 'numeric'
							})}
						</p>
					</div>
				</div>
				<div class="flex items-center gap-1">
					<a
						href={resolve('/journal')}
						title="Journal"
						aria-label="Journal"
						class="rounded-lg p-2 text-indigo-200/50 transition hover:bg-white/10 hover:text-white"
					>
						<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
							<path stroke-linecap="round" stroke-linejoin="round" d="M12 6.042A8.967 8.967 0 006 3.75c-1.052 0-2.062.18-3 .512v14.25A8.987 8.987 0 016 18c2.305 0 4.408.867 6 2.292m0-14.25a8.966 8.966 0 016-2.292c1.052 0 2.062.18 3 .512v14.25A8.987 8.987 0 0018 18a8.967 8.967 0 00-6 2.292m0-14.25v14.25" />
						</svg>
					</a>
					<a
						href={resolve('/settings')}
						title="Settings"
						aria-label="Settings"
						class="rounded-lg p-2 text-indigo-200/50 transition hover:bg-white/10 hover:text-white"
					>
						<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
							<path stroke-linecap="round" stroke-linejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.325.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 011.37.49l1.296 2.247a1.125 1.125 0 01-.26 1.431l-1.003.827c-.293.241-.438.613-.43.992a7.723 7.723 0 010 .255c-.008.378.137.75.43.991l1.004.827c.424.35.534.955.26 1.43l-1.298 2.247a1.125 1.125 0 01-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.47 6.47 0 01-.22.128c-.331.183-.581.495-.644.869l-.213 1.281c-.09.543-.56.94-1.11.94h-2.594c-.55 0-1.019-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 01-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 01-1.369-.49l-1.297-2.247a1.125 1.125 0 01.26-1.431l1.004-.827c.292-.24.437-.613.43-.991a6.932 6.932 0 010-.255c.007-.38-.138-.751-.43-.992l-1.004-.827a1.125 1.125 0 01-.26-1.43l1.297-2.247a1.125 1.125 0 011.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.086.22-.128.332-.183.582-.495.644-.869l.214-1.28z" />
							<path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
						</svg>
					</a>
					<div class="mx-2 h-4 w-px bg-white/10"></div>
					<button
						type="button"
						onclick={handleLogout}
						disabled={logoutLoading}
						class="rounded-lg px-3 py-1.5 text-sm text-indigo-200/40 transition hover:text-white disabled:opacity-50"
					>
						{logoutLoading ? 'Signing out...' : 'Sign out'}
					</button>
				</div>
			</div>
		</header>

		<main class="mx-auto max-w-5xl px-6 py-8">
			{#if error}
				<div
					in:fly={{ y: -5, duration: 300 }}
					class="mb-6 rounded-lg bg-red-500/10 p-3 text-sm text-red-300 ring-1 ring-red-500/20"
				>
					{error}
				</div>
			{/if}

			{#if showRelapseNotice}
				<div
					in:fly={{ y: -5, duration: 300 }}
					class="mb-6 flex items-start justify-between rounded-xl border border-blue-400/20 bg-blue-400/10 p-4"
				>
					<p class="text-sm text-blue-300">
						We've adjusted your plan to help you get back on track. Your difficulty has been lowered
						— focus on building momentum.
					</p>
					<button
						type="button"
						onclick={() => (relapseNoticeDismissed = true)}
						class="ml-3 shrink-0 text-blue-400/60 transition hover:text-blue-300"
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

			<!-- Stats row -->
			{#if adherence && plan}
				<div
					in:fly={{ y: 15, duration: 500, delay: 50, easing: cubicOut }}
					class="mb-8 grid grid-cols-2 gap-3 sm:grid-cols-4"
				>
					<div class="rounded-xl border border-emerald-400/15 bg-emerald-400/[0.06] p-4">
						<p class="text-xs font-medium text-emerald-300/70">Completion</p>
						<p class="mt-1 text-2xl font-bold text-white">{completionPct}%</p>
						<div class="mt-2 h-1 overflow-hidden rounded-full bg-emerald-400/15">
							<div
								class="h-full rounded-full bg-emerald-400 transition-all duration-500"
								style="width: {completionPct}%"
							></div>
						</div>
					</div>
					<div class="rounded-xl border border-amber-400/15 bg-amber-400/[0.06] p-4">
						<p class="text-xs font-medium text-amber-300/70">Streak</p>
						<p class="mt-1 text-2xl font-bold text-white">
							{adherence.streak_count}
							<span class="text-sm font-normal text-amber-200/50">
								day{adherence.streak_count !== 1 ? 's' : ''}
							</span>
						</p>
					</div>
					<div class="rounded-xl border border-sky-400/15 bg-sky-400/[0.06] p-4">
						<p class="text-xs font-medium text-sky-300/70">Tasks done</p>
						<p class="mt-1 text-2xl font-bold text-white">
							{adherence.completed_tasks}<span class="text-sm font-normal text-sky-200/50"
								>/{adherence.total_tasks}</span
							>
						</p>
					</div>
					<div class="rounded-xl border border-violet-400/15 bg-violet-400/[0.06] p-4">
						<p class="text-xs font-medium text-violet-300/70">Difficulty</p>
						<p class="mt-1 text-2xl font-bold text-white">
							{plan.difficulty_score}<span class="text-sm font-normal text-violet-200/50">/10</span>
						</p>
						<div class="mt-2 h-1 overflow-hidden rounded-full bg-violet-400/15">
							<div
								class="h-full rounded-full bg-violet-400 transition-all duration-500"
								style="width: {plan.difficulty_score * 10}%"
							></div>
						</div>
					</div>
				</div>
				{#if adherence.difficulty_mismatch}
					<div
						in:fly={{ y: -5, duration: 300 }}
						class="mb-6 rounded-lg border border-amber-400/20 bg-amber-400/10 p-3 text-sm text-amber-300"
					>
						Your completion rate is low for the current difficulty. The plan may be adjusted soon.
					</div>
				{/if}
			{/if}

			<!-- Main 2-column layout -->
			<div class="grid grid-cols-1 gap-8 lg:grid-cols-3">
				<!-- Left column: Today's tasks (2/3 width) -->
				<div class="lg:col-span-2">
					<div in:fly={{ y: 20, duration: 600, delay: 100, easing: cubicOut }}>
						<div class="mb-4 flex items-center justify-between">
							<h2 class="text-lg font-semibold text-white">Today's Tasks</h2>
							{#if plan}
								<span class="text-xs text-indigo-200/40">
									{completedTasks.size}/{plan.tasks.length} completed
								</span>
							{/if}
						</div>

						{#if !plan}
							<div
								class="rounded-2xl border border-dashed border-white/10 bg-white/[0.02] p-12 text-center"
							>
								<div
									class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-indigo-400/10"
								>
									<svg
										class="h-6 w-6 text-indigo-400"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
										stroke-width="1.5"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											d="M12 4.5v15m7.5-7.5h-15"
										/>
									</svg>
								</div>
								<p class="mb-1 text-sm font-medium text-white">No plan for today yet</p>
								<p class="mb-6 text-xs text-indigo-200/40">
									Generate a personalized plan based on your profile
								</p>
								<button
									type="button"
									onclick={handleGenerate}
									disabled={generating}
									class="rounded-lg bg-white px-6 py-2.5 text-sm font-semibold text-slate-900 transition hover:bg-indigo-100 disabled:opacity-50"
								>
									{generating ? 'Generating...' : "Generate today's plan"}
								</button>
							</div>
						{:else if deliveryMode === 'gentle_structure'}
							{#if anchorTasks.length > 0}
								<h3 class="mb-2 text-sm font-medium text-indigo-200/60">Daily Anchors</h3>
								<p class="mb-3 text-xs text-indigo-200/40">
									Your three grounding points for the day.
								</p>
								<div class="mb-5 space-y-4">
									{#each anchorTasks as task (task.id)}
										<div class="relative pt-3">
											{#if task.anchor_label && anchorLabels[task.anchor_label]}
												<span
													class="absolute -top-0 left-3 z-10 rounded-full bg-indigo-400/15 px-2 py-0.5 text-xs font-medium text-indigo-300 ring-1 ring-indigo-400/20"
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
								<h3 class="mb-2 text-sm font-medium text-indigo-200/60">Tiny Tasks</h3>
								<p class="mb-3 text-xs text-indigo-200/40">
									Small, gentle actions. Do what feels manageable.
								</p>
								<div class="space-y-2">
									{#each regularTasks as task (task.id)}
										{@render taskItem(task)}
									{/each}
								</div>
							{/if}
						{:else}
							<div class="space-y-2">
								{#each plan.tasks as task (task.id)}
									{@render taskItem(task)}
								{/each}
							</div>
						{/if}

						<!-- Adaptations -->
						{#if plan && adaptations.length > 0}
							<div class="mt-8">
								<h2 class="mb-3 text-lg font-semibold text-white">Plan adjustments</h2>
								<div class="space-y-2">
									{#each adaptations as log (log.id)}
										<div class="rounded-xl border border-white/10 bg-white/[0.04] p-4">
											<p class="text-sm text-indigo-200/70">{log.adaptation_reason}</p>
											<p class="mt-1.5 text-xs text-indigo-200/40">
												Difficulty: {log.previous_difficulty} → {log.new_difficulty}
												<span
													class="ml-1 font-medium {log.difficulty_change > 0
														? 'text-red-400'
														: 'text-green-400'}"
												>
													({log.difficulty_change > 0 ? '+' : ''}{log.difficulty_change})
												</span>
											</p>
										</div>
									{/each}
								</div>
							</div>
						{/if}
					</div>
				</div>

				<!-- Right column: Insights (1/3 width) -->
				<div class="space-y-6">
					<div in:fly={{ y: 20, duration: 600, delay: 200, easing: cubicOut }}>
						{#if analytics}
							<!-- Weekly Trends -->
							{#if analytics.weekly_trends.length > 0}
								<div class="rounded-xl border border-indigo-400/10 bg-indigo-400/[0.04] p-5">
									<h3 class="mb-4 text-sm font-semibold text-white">Weekly Trends</h3>
									<div class="space-y-3">
										{#each analytics.weekly_trends.slice(-4) as week (week.week_start)}
											{@const pct = Math.round(week.completion_rate * 100)}
											<div>
												<div class="mb-1 flex items-baseline justify-between">
													<span class="text-xs text-indigo-200/60">
														{formatWeek(week.week_start)}
													</span>
													<span class="text-xs font-medium text-white">{pct}%</span>
												</div>
												<div class="h-1.5 overflow-hidden rounded-full bg-white/10">
													<div
														class="h-full rounded-full bg-indigo-400 transition-all duration-500"
														style="width: {pct}%"
													></div>
												</div>
												<p class="mt-0.5 text-right text-xs text-indigo-200/30">
													{week.completed_tasks}/{week.total_tasks}
												</p>
											</div>
										{/each}
									</div>
								</div>
							{/if}

							<!-- Category Breakdown -->
							{#if analytics.category_performance.length > 0}
								<div class="mt-6 rounded-xl border border-indigo-400/10 bg-indigo-400/[0.04] p-5">
									<h3 class="mb-4 text-sm font-semibold text-white">Categories</h3>
									<div class="space-y-3">
										{#each analytics.category_performance as cat (cat.category)}
											{@const pct = Math.round(cat.completion_rate * 100)}
											<div>
												<div class="mb-1 flex items-baseline justify-between">
													<span
														class="text-xs capitalize {categoryTextColors[cat.category] ??
															'text-indigo-200/60'}"
													>
														{cat.category}
													</span>
													<span class="text-xs text-indigo-200/40">
														{cat.completed_tasks}/{cat.total_tasks}
													</span>
												</div>
												<div class="h-1.5 overflow-hidden rounded-full bg-white/10">
													<div
														class="h-full rounded-full transition-all duration-500 {categoryBarColors[
															cat.category
														] ?? 'bg-indigo-400'}"
														style="width: {pct}%"
													></div>
												</div>
											</div>
										{/each}
									</div>
								</div>
							{/if}

							<!-- Difficulty Trajectory -->
							{#if analytics.difficulty_trajectory.length > 1}
								<div class="mt-6 rounded-xl border border-indigo-400/10 bg-indigo-400/[0.04] p-5">
									<h3 class="mb-4 text-sm font-semibold text-white">Difficulty Over Time</h3>
									<div class="flex items-end gap-px" style="height: 80px;">
										{#each analytics.difficulty_trajectory.slice(-14) as point (point.date)}
											{@const heightPct = (point.difficulty / 10) * 100}
											<div
												class="group relative flex flex-1 flex-col items-center justify-end"
												style="height: 100%;"
											>
												<div
													class="w-full rounded-t bg-indigo-400/60 transition-all group-hover:bg-indigo-400"
													style="height: {heightPct}%; min-height: 2px;"
												></div>
												<div
													class="pointer-events-none absolute bottom-full -mb-6 hidden rounded bg-slate-800 px-1.5 py-0.5 text-xs text-white ring-1 ring-white/10 group-hover:block"
												>
													{point.difficulty}
												</div>
											</div>
										{/each}
									</div>
									<div class="mt-2 flex justify-between text-xs text-indigo-200/30">
										<span>{formatDate(analytics.difficulty_trajectory.slice(-14)[0].date)}</span>
										<span
											>{formatDate(
												analytics.difficulty_trajectory.slice(-14).at(-1)?.date ?? ''
											)}</span
										>
									</div>
								</div>
							{/if}
						{:else}
							<div
								class="rounded-xl border border-dashed border-white/10 bg-white/[0.02] p-8 text-center"
							>
								<p class="text-xs text-indigo-200/40">
									Insights will appear here as you complete tasks.
								</p>
							</div>
						{/if}
					</div>
				</div>
			</div>
		</main>
	{/if}
</div>

<style>
	.dashboard-bg {
		background: linear-gradient(135deg, #0f172a 0%, #1e1b4b 40%, #312e81 100%);
	}
</style>
