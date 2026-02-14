<script lang="ts">
	import { onMount } from 'svelte';
	import { fetchAnalytics } from '$lib/api';
	import type { AnalyticsResponse } from '$lib/api-types';
	import { resolve } from '$app/paths';

	let loading = $state(true);
	let error = $state<string | null>(null);
	let analytics = $state<AnalyticsResponse | null>(null);

	const categoryColors: Record<string, { bar: string; bg: string; text: string }> = {
		health: { bar: 'bg-green-500', bg: 'bg-green-50', text: 'text-green-800' },
		mindset: { bar: 'bg-purple-500', bg: 'bg-purple-50', text: 'text-purple-800' },
		discipline: { bar: 'bg-orange-500', bg: 'bg-orange-50', text: 'text-orange-800' },
		relationships: { bar: 'bg-blue-500', bg: 'bg-blue-50', text: 'text-blue-800' },
		productivity: { bar: 'bg-yellow-500', bg: 'bg-yellow-50', text: 'text-yellow-800' }
	};

	onMount(async () => {
		const result = await fetchAnalytics();
		if (result.error) {
			error = result.error;
		} else {
			analytics = result.data;
		}
		loading = false;
	});

	function formatWeek(dateStr: string): string {
		const d = new Date(dateStr + 'T00:00:00');
		return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
	}

	function formatDate(dateStr: string): string {
		const d = new Date(dateStr + 'T00:00:00');
		return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
	}
</script>

{#if loading}
	<main class="mx-auto max-w-2xl p-8">
		<p class="text-gray-500">Loading...</p>
	</main>
{:else}
	<main class="mx-auto max-w-2xl p-8">
		<div class="mb-6 flex items-center justify-between">
			<h1 class="text-3xl font-bold">Insights</h1>
			<a href={resolve('/dashboard')} class="text-sm underline">Back to dashboard</a>
		</div>

		{#if error}
			<div class="mb-4 rounded bg-red-100 p-3 text-sm text-red-700">{error}</div>
		{/if}

		{#if analytics}
			<!-- Weekly Trends -->
			<section class="mb-8">
				<h2 class="mb-3 text-lg font-semibold">Weekly Trends</h2>
				{#if analytics.weekly_trends.length === 0}
					<p class="text-sm text-gray-500">No data yet. Complete some tasks to see trends.</p>
				{:else}
					<div class="space-y-3">
						{#each analytics.weekly_trends as week (week.week_start)}
							{@const pct = Math.round(week.completion_rate * 100)}
							<div>
								<div class="mb-1 flex items-baseline justify-between text-sm">
									<span class="font-medium">Week of {formatWeek(week.week_start)}</span>
									<span class="text-gray-500">
										{week.completed_tasks}/{week.total_tasks} tasks ({pct}%)
									</span>
								</div>
								<div class="h-4 w-full overflow-hidden rounded-full bg-gray-100">
									<div
										class="h-full rounded-full bg-black transition-all"
										style="width: {pct}%"
									></div>
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</section>

			<!-- Category Breakdown -->
			<section class="mb-8">
				<h2 class="mb-3 text-lg font-semibold">Category Breakdown</h2>
				{#if analytics.category_performance.length === 0}
					<p class="text-sm text-gray-500">No category data yet.</p>
				{:else}
					<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
						{#each analytics.category_performance as cat (cat.category)}
							{@const colors = categoryColors[cat.category] ?? {
								bar: 'bg-gray-500',
								bg: 'bg-gray-50',
								text: 'text-gray-800'
							}}
							{@const pct = Math.round(cat.completion_rate * 100)}
							<div class="rounded-lg border p-4 {colors.bg}">
								<p class="text-sm font-semibold capitalize {colors.text}">{cat.category}</p>
								<p class="mt-1 text-xs text-gray-600">
									{cat.completed_tasks}/{cat.total_tasks} completed
								</p>
								<div class="mt-2 h-2 w-full overflow-hidden rounded-full bg-white/60">
									<div
										class="h-full rounded-full {colors.bar} transition-all"
										style="width: {pct}%"
									></div>
								</div>
								<p class="mt-1 text-right text-xs font-medium {colors.text}">{pct}%</p>
							</div>
						{/each}
					</div>
				{/if}
			</section>

			<!-- Difficulty Trajectory -->
			<section class="mb-8">
				<h2 class="mb-3 text-lg font-semibold">Difficulty Trajectory</h2>
				{#if analytics.difficulty_trajectory.length === 0}
					<p class="text-sm text-gray-500">No difficulty data yet.</p>
				{:else}
					<div class="overflow-hidden rounded-lg border">
						<div class="flex items-end gap-1 p-4" style="height: 160px;">
							{#each analytics.difficulty_trajectory as point (point.date)}
								{@const heightPct = (point.difficulty / 10) * 100}
								<div
									class="group relative flex flex-1 flex-col items-center justify-end"
									style="height: 100%;"
								>
									<div
										class="w-full min-w-1 rounded-t bg-black transition-all group-hover:bg-gray-700"
										style="height: {heightPct}%"
									></div>
									<div
										class="pointer-events-none absolute -top-8 hidden rounded bg-gray-800 px-2 py-1 text-xs text-white group-hover:block"
									>
										{formatDate(point.date)}: {point.difficulty}
									</div>
								</div>
							{/each}
						</div>
						<div class="flex justify-between border-t bg-gray-50 px-4 py-2 text-xs text-gray-500">
							<span>{formatDate(analytics.difficulty_trajectory[0].date)}</span>
							<span
								>{formatDate(
									analytics.difficulty_trajectory[analytics.difficulty_trajectory.length - 1].date
								)}</span
							>
						</div>
					</div>
				{/if}
			</section>
		{/if}

		<div class="mt-8 border-t pt-6">
			<a href={resolve('/dashboard')} class="text-sm underline">Back to dashboard</a>
		</div>
	</main>
{/if}
