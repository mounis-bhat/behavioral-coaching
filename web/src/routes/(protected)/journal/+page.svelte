<script lang="ts">
	import { resolve } from '$app/paths';
	import { onMount, untrack } from 'svelte';
	import { fly } from 'svelte/transition';
	import { cubicOut } from 'svelte/easing';
	import {
		upsertJournalEntry,
		getTodaysJournalEntry,
		listJournalEntries,
		getJournalMoodTrend
	} from '$lib/api';
	import type { JournalEntryResponse, MoodTrendResponse } from '$lib/api';

	const PROMPTS = [
		'What is one thing you did today that moved you closer to who you want to be?',
		'What challenged you today, and how did you respond?',
		'What are three things that went well today, no matter how small?',
		'Where did you spend your energy today? Was it aligned with your priorities?',
		'What would you do differently if you could repeat today?',
		'What are you grateful for right now?',
		'What intention do you want to carry into tomorrow?'
	];

	const MOOD_EMOJIS = ['😩', '😢', '😟', '😕', '😐', '🙂', '😊', '😄', '😁', '🤩'];
	const MOOD_LABELS = [
		{ range: [1, 3], label: 'Struggling', color: 'text-red-400' },
		{ range: [4, 6], label: 'Neutral', color: 'text-yellow-400' },
		{ range: [7, 10], label: 'Thriving', color: 'text-green-400' }
	];

	let loaded = $state(false);
	let saving = $state(false);
	let toast = $state<string | null>(null);

	let content = $state('');
	let moodScore = $state(0);
	let expandedEntry = $state<string | null>(null);

	let todaysEntry = $state<JournalEntryResponse | null>(null);
	let entries = $state<JournalEntryResponse[]>([]);
	let moodTrend = $state<MoodTrendResponse | null>(null);

	let grid14 = $derived.by(() =>
		Array.from({ length: 14 }, (_, idx) => {
			const d = new Date();
			d.setDate(d.getDate() - (13 - idx));
			const dateStr = d.toISOString().slice(0, 10);
			const score = moodTrend?.data_points.find((p) => p.date === dateStr)?.score ?? null;
			return { date: dateStr, score };
		})
	);

	let daysLogged = $derived(moodTrend?.data_points.length ?? 0);

	let editorEl = $state<HTMLDivElement | null>(null);
	let isBold = $state(false);
	let isItalic = $state(false);
	let isList = $state(false);
	let alignment = $state<'left' | 'center' | 'right'>('left');

	$effect(() => {
		if (editorEl) {
			editorEl.innerHTML = untrack(() => normalizeForEditor(content));
		}
	});

	$effect(() => {
		const el = editorEl;
		if (!el) return;
		const update = () => {
			if (el === document.activeElement || el.contains(document.activeElement)) {
				isBold = document.queryCommandState('bold');
				isItalic = document.queryCommandState('italic');
				isList = document.queryCommandState('insertUnorderedList');
				if (document.queryCommandState('justifyCenter')) alignment = 'center';
				else if (document.queryCommandState('justifyRight')) alignment = 'right';
				else alignment = 'left';
			}
		};
		document.addEventListener('selectionchange', update);
		return () => document.removeEventListener('selectionchange', update);
	});

	let today = new Date();
	let promptIndex = today.getDay(); // 0-6
	let prompt = PROMPTS[promptIndex];

	let todayFormatted = $derived(
		today.toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric' })
	);

	let moodLabel = $derived(() => {
		if (moodScore === 0) return null;
		return MOOD_LABELS.find((m) => moodScore >= m.range[0] && moodScore <= m.range[1]) ?? null;
	});

	onMount(() => {
		(async () => {
			const [todayRes, listRes, trendRes] = await Promise.all([
				getTodaysJournalEntry(),
				listJournalEntries(10, 0),
				getJournalMoodTrend()
			]);

			if (todayRes.data) {
				todaysEntry = todayRes.data;
				content = todaysEntry.content;
				moodScore = todaysEntry.mood_score;
			}
			if (listRes.data) entries = listRes.data;
			if (trendRes.data) moodTrend = trendRes.data;

			loaded = true;
		})();
	});

	async function handleSave() {
		if (!editorEl?.innerText?.trim()) {
			showToast('Please write something before saving.');
			return;
		}
		if (moodScore === 0) {
			showToast('Please select a mood score.');
			return;
		}

		saving = true;
		try {
			const result = await upsertJournalEntry({
				content: editorEl?.innerHTML ?? content,
				mood_score: moodScore,
				prompt_used: prompt
			});

			if (result.error) throw new Error(result.error);
			todaysEntry = result.data;

			// Refresh list and trend
			const [listRes, trendRes] = await Promise.all([
				listJournalEntries(10, 0),
				getJournalMoodTrend()
			]);
			if (listRes.data) entries = listRes.data;
			if (trendRes.data) moodTrend = trendRes.data;

			showToast('Journal saved!');
		} catch (e) {
			showToast(e instanceof Error ? e.message : 'Failed to save');
		} finally {
			saving = false;
		}
	}

	function showToast(msg: string) {
		toast = msg;
		setTimeout(() => (toast = null), 3000);
	}

	function formatEntryDate(dateStr: string) {
		const d = new Date(dateStr + 'T00:00:00');
		return d.toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' });
	}

	function moodEmoji(score: number) {
		return MOOD_EMOJIS[score - 1] ?? '😐';
	}

	function trendColor(trend: string) {
		if (trend === 'improving') return 'text-green-400';
		if (trend === 'declining') return 'text-red-400';
		return 'text-yellow-400';
	}

	function trendIcon(trend: string) {
		if (trend === 'improving') return '↑';
		if (trend === 'declining') return '↓';
		return '→';
	}

	// --- RTE ---

	function normalizeForEditor(text: string): string {
		// Already HTML (from a previous RTE save) — use as-is
		if (/<[a-z][\s\S]*>/i.test(text)) return text;
		// Plain text: escape and convert newlines to <br>
		return text
			.replace(/&/g, '&amp;')
			.replace(/</g, '&lt;')
			.replace(/>/g, '&gt;')
			.replace(/\n/g, '<br>');
	}

	function updateFormattingState() {
		isBold = document.queryCommandState('bold');
		isItalic = document.queryCommandState('italic');
		isList = document.queryCommandState('insertUnorderedList');
		if (document.queryCommandState('justifyCenter')) alignment = 'center';
		else if (document.queryCommandState('justifyRight')) alignment = 'right';
		else alignment = 'left';
	}

	function execFormat(cmd: string) {
		editorEl?.focus();
		document.execCommand(cmd, false);
		content = editorEl?.innerHTML ?? '';
		updateFormattingState();
	}

	function handleEditorInput() {
		content = editorEl?.innerHTML ?? '';
	}

	function handleEditorKeydown(e: KeyboardEvent) {
		if (e.ctrlKey || e.metaKey) {
			if (e.key === 'b') {
				e.preventDefault();
				execFormat('bold');
			} else if (e.key === 'i') {
				e.preventDefault();
				execFormat('italic');
			}
		}
	}

	function handlePaste(e: ClipboardEvent) {
		e.preventDefault();
		const text = e.clipboardData?.getData('text/plain') ?? '';
		document.execCommand('insertText', false, text);
		content = editorEl?.innerHTML ?? '';
	}
</script>

<svelte:head>
	<title>Daily Journal — Behavioral Coaching</title>
</svelte:head>

<div class="dashboard-bg min-h-screen">
	{#if !loaded}
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
				<span class="text-sm">Loading journal...</span>
			</div>
		</div>
	{:else}
		<main class="mx-auto max-w-3xl px-6 py-8">
			<div in:fly={{ y: -10, duration: 500, easing: cubicOut }} class="mb-8">
				<a href={resolve('/dashboard')} class="text-sm text-indigo-300/50 transition hover:text-indigo-200">
					&larr; Dashboard
				</a>
				<h1 class="mt-3 text-2xl font-bold text-white">Daily Journal</h1>
				<p class="mt-1 text-sm text-indigo-200/50">{todayFormatted}</p>
			</div>
			<!-- Toast -->
			{#if toast}
				<div
					in:fly={{ y: -8, duration: 200 }}
					class="fixed top-6 right-6 z-50 rounded-xl border border-white/10 bg-white/10 px-4 py-3 text-sm text-white backdrop-blur"
				>
					{toast}
				</div>
			{/if}

			<div class="space-y-6">
				<!-- Reflection prompt -->
				<div
					in:fly={{ y: 15, duration: 400, easing: cubicOut }}
					class="rounded-xl border border-indigo-400/20 bg-indigo-400/6 p-4"
				>
					<p class="mb-1 text-xs font-medium tracking-wide text-indigo-300/60 uppercase">
						Today's Prompt
					</p>
					<p class="text-sm text-indigo-100">{prompt}</p>
				</div>

				<!-- Mood selector -->
				<div in:fly={{ y: 15, duration: 400, delay: 50, easing: cubicOut }}>
					<div class="mb-3 flex items-center justify-between">
						<p class="text-sm font-medium text-white/80">How are you feeling?</p>
						{#if moodScore > 0}
							{@const label = moodLabel()}
							{#if label}
								<span class="text-sm font-medium {label.color}">{label.label}</span>
							{/if}
						{/if}
					</div>
					<div class="flex gap-2">
						{#each MOOD_EMOJIS as emoji, i}
							{@const score = i + 1}
							<button
								type="button"
								onclick={() => (moodScore = score)}
								class="flex flex-1 flex-col items-center rounded-lg border py-2 text-lg transition {moodScore ===
								score
									? 'border-indigo-400/50 bg-indigo-500/20'
									: 'border-white/10 bg-white/3 hover:border-white/20 hover:bg-white/6'}"
								aria-label="Mood {score}"
							>
								<span>{emoji}</span>
								<span class="mt-0.5 text-[10px] text-white/30">{score}</span>
							</button>
						{/each}
					</div>
					<div class="mt-1.5 flex justify-between text-[10px] text-white/30">
						<span>1–3 Struggling</span>
						<span>4–6 Neutral</span>
						<span>7–10 Thriving</span>
					</div>
				</div>

				<!-- Content editor (RTE) -->
				<div in:fly={{ y: 15, duration: 400, delay: 100, easing: cubicOut }}>
					<!-- Toolbar -->
					<div
						class="flex items-center gap-0.5 rounded-t-xl border border-b-0 border-white/10 bg-white/3 px-2 py-1.5"
					>
						<button
							type="button"
							onclick={() => execFormat('bold')}
							title="Bold (Ctrl+B)"
							class="rounded px-2.5 py-1 text-sm font-bold transition {isBold
								? 'bg-white/15 text-white'
								: 'text-white/40 hover:bg-white/10 hover:text-white/80'}"
						>B</button>
						<button
							type="button"
							onclick={() => execFormat('italic')}
							title="Italic (Ctrl+I)"
							class="rounded px-2.5 py-1 text-sm italic transition {isItalic
								? 'bg-white/15 text-white'
								: 'text-white/40 hover:bg-white/10 hover:text-white/80'}"
						>I</button>
						<div class="mx-1.5 h-3.5 w-px bg-white/10"></div>
						<button
							type="button"
							onclick={() => execFormat('insertUnorderedList')}
							title="Bullet list"
							class="rounded px-2 py-1 transition {isList
								? 'bg-white/15 text-white'
								: 'text-white/40 hover:bg-white/10 hover:text-white/80'}"
						>
							<svg class="h-4 w-4" viewBox="0 0 16 14" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round">
								<circle cx="2" cy="3" r="1" fill="currentColor" stroke="none"/>
								<circle cx="2" cy="7" r="1" fill="currentColor" stroke="none"/>
								<circle cx="2" cy="11" r="1" fill="currentColor" stroke="none"/>
								<line x1="5" y1="3" x2="15" y2="3"/>
								<line x1="5" y1="7" x2="15" y2="7"/>
								<line x1="5" y1="11" x2="15" y2="11"/>
							</svg>
						</button>
						<div class="mx-1.5 h-3.5 w-px bg-white/10"></div>
						<button
							type="button"
							onclick={() => execFormat('justifyLeft')}
							title="Align left"
							class="rounded px-2 py-1 transition {alignment === 'left'
								? 'bg-white/15 text-white'
								: 'text-white/40 hover:bg-white/10 hover:text-white/80'}"
						>
							<svg class="h-4 w-4" viewBox="0 0 16 14" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round">
								<line x1="1" y1="2" x2="15" y2="2"/>
								<line x1="1" y1="6" x2="10" y2="6"/>
								<line x1="1" y1="10" x2="13" y2="10"/>
							</svg>
						</button>
						<button
							type="button"
							onclick={() => execFormat('justifyCenter')}
							title="Align center"
							class="rounded px-2 py-1 transition {alignment === 'center'
								? 'bg-white/15 text-white'
								: 'text-white/40 hover:bg-white/10 hover:text-white/80'}"
						>
							<svg class="h-4 w-4" viewBox="0 0 16 14" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round">
								<line x1="1" y1="2" x2="15" y2="2"/>
								<line x1="3.5" y1="6" x2="12.5" y2="6"/>
								<line x1="2" y1="10" x2="14" y2="10"/>
							</svg>
						</button>
						<button
							type="button"
							onclick={() => execFormat('justifyRight')}
							title="Align right"
							class="rounded px-2 py-1 transition {alignment === 'right'
								? 'bg-white/15 text-white'
								: 'text-white/40 hover:bg-white/10 hover:text-white/80'}"
						>
							<svg class="h-4 w-4" viewBox="0 0 16 14" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round">
								<line x1="1" y1="2" x2="15" y2="2"/>
								<line x1="6" y1="6" x2="15" y2="6"/>
								<line x1="3" y1="10" x2="15" y2="10"/>
							</svg>
						</button>
					</div>
					<!-- Editor -->
					<div
						bind:this={editorEl}
						contenteditable="true"
						role="textbox"
						tabindex="0"
						aria-multiline="true"
						aria-label="Journal entry"
						data-placeholder="Write your reflection here…"
						oninput={handleEditorInput}
						onkeydown={handleEditorKeydown}
						onpaste={handlePaste}
						class="rte min-h-32 w-full rounded-b-xl border border-white/10 bg-white/4 px-4 py-3 text-sm text-white outline-none transition focus:border-indigo-400/40 focus:bg-white/6"
					></div>
				</div>

				<!-- Save button -->
				<div in:fly={{ y: 15, duration: 400, delay: 150, easing: cubicOut }}>
					<button
						type="button"
						onclick={handleSave}
						disabled={saving}
						class="w-full rounded-xl bg-indigo-600 px-4 py-3 text-sm font-medium text-white transition hover:bg-indigo-500 disabled:opacity-60"
					>
						{saving ? 'Saving…' : todaysEntry ? "Update Today's Entry" : 'Save Entry'}
					</button>
				</div>

				<!-- Mood trend sparkline -->
				<div
					in:fly={{ y: 15, duration: 400, delay: 200, easing: cubicOut }}
					class="rounded-xl border border-white/10 bg-white/3 p-4"
				>
					<div class="mb-3 flex items-center justify-between">
						<div>
							<p class="text-sm font-medium text-white/80">14-Day Mood Trend</p>
							<p class="text-xs text-white/30">{daysLogged} of 14 days logged</p>
						</div>
						{#if moodTrend && daysLogged > 0}
							<div class="flex items-center gap-2">
								<span class="text-sm font-bold text-white">{moodTrend.avg_score.toFixed(1)}/10</span>
								<span class="text-sm font-medium {trendColor(moodTrend.trend)}">
									{trendIcon(moodTrend.trend)}
									{moodTrend.trend}
								</span>
							</div>
						{/if}
					</div>
					<!-- SVG sparkline — always 14 fixed slots -->
					<svg viewBox="0 0 252 48" class="w-full overflow-visible">
						{#each grid14 as point, i}
							{@const x = i * 18 + 4}
							{#if point.score !== null}
								{@const barH = Math.max(4, Math.round((point.score / 10) * 40))}
								<rect
									{x}
									y={48 - barH}
									width="10"
									height={barH}
									rx="2"
									class={point.score >= 7
										? 'fill-green-400/60'
										: point.score >= 4
											? 'fill-yellow-400/60'
											: 'fill-red-400/60'}
								/>
							{:else}
								<rect {x} y={45} width="10" height="3" rx="1.5" class="fill-white/8" />
							{/if}
						{/each}
					</svg>
					<div class="mt-1 flex justify-between text-[10px] text-white/25">
						<span>{formatEntryDate(grid14[0].date)}</span>
						<span>{formatEntryDate(grid14[13].date)}</span>
					</div>
					{#if daysLogged === 0}
						<p class="mt-3 text-center text-xs text-white/30">
							Save your first entry to start tracking your mood
						</p>
					{/if}
				</div>

				<!-- Past entries -->
				{#if entries.length > 0}
					<div in:fly={{ y: 15, duration: 400, delay: 250, easing: cubicOut }}>
						<h2 class="mb-3 text-sm font-medium text-white/60">Past Entries</h2>
						<div class="space-y-2">
							{#each entries as entry (entry.id)}
								{@const isToday = entry.entry_date === today.toISOString().slice(0, 10)}
								<div class="overflow-hidden rounded-xl border border-white/10 bg-white/3">
									<button
										type="button"
										onclick={() => (expandedEntry = expandedEntry === entry.id ? null : entry.id)}
										class="flex w-full items-center justify-between px-4 py-3 text-left"
									>
										<div class="flex items-center gap-3">
											<span class="text-xl">{moodEmoji(entry.mood_score)}</span>
											<div>
												<p class="text-sm font-medium text-white/80">
													{isToday ? 'Today' : formatEntryDate(entry.entry_date)}
												</p>
												<p class="text-xs text-white/30">Mood {entry.mood_score}/10</p>
											</div>
										</div>
										<svg
											class="h-4 w-4 text-white/30 transition-transform {expandedEntry === entry.id
												? 'rotate-180'
												: ''}"
											fill="none"
											viewBox="0 0 24 24"
											stroke="currentColor"
											stroke-width="2"
										>
											<path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
										</svg>
									</button>
									{#if expandedEntry === entry.id}
										<div class="border-t border-white/6 px-4 py-3">
											{#if entry.prompt_used}
												<p class="mb-2 text-xs text-indigo-300/50 italic">{entry.prompt_used}</p>
											{/if}
											<!-- eslint-disable-next-line svelte/no-at-html-tags -->
										<div class="entry-content text-sm text-white/70">{@html entry.content}</div>
										</div>
									{/if}
								</div>
							{/each}
						</div>
					</div>
				{/if}
			</div>
		</main>
	{/if}
</div>

<style>
	.dashboard-bg {
		background: linear-gradient(135deg, #0f172a 0%, #1e1b4b 40%, #312e81 100%);
	}

	/* RTE editor */
	.rte:empty::before {
		content: attr(data-placeholder);
		color: rgba(255, 255, 255, 0.2);
		pointer-events: none;
		display: block;
	}

	.rte :global(b),
	.rte :global(strong),
	.entry-content :global(b),
	.entry-content :global(strong) {
		font-weight: 600;
	}

	.rte :global(i),
	.rte :global(em),
	.entry-content :global(i),
	.entry-content :global(em) {
		font-style: italic;
	}

	.rte :global(ul),
	.entry-content :global(ul) {
		list-style: disc;
		padding-left: 1.25rem;
		margin: 0.25rem 0;
	}
</style>
