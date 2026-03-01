<script lang="ts">
	import { resolve } from '$app/paths';
	import { onMount } from 'svelte';
	import { fly } from 'svelte/transition';
	import { cubicOut } from 'svelte/easing';
	import { marked } from 'marked';
	import { listChatMessages, sendChatMessage, clearChatHistory } from '$lib/api';
	import type { ChatMessage, SendMessageResponse } from '$lib/api';

	// Configure marked for safe inline rendering
	marked.setOptions({ breaks: true });

	let loaded = $state(false);
	let messages = $state<ChatMessage[]>([]);
	let inputText = $state('');
	let streaming = $state(false);
	let streamingContent = $state('');
	// Optimistic user message shown immediately while waiting for server confirmation
	let pendingUserMessage = $state<string | null>(null);
	let error = $state<string | null>(null);
	let showClearConfirm = $state(false);
	let clearing = $state(false);

	let messagesEl = $state<HTMLDivElement | null>(null);

	function scrollToBottom() {
		if (messagesEl) {
			messagesEl.scrollTop = messagesEl.scrollHeight;
		}
	}

	function scheduleScroll() {
		requestAnimationFrame(scrollToBottom);
	}

	onMount(async () => {
		const result = await listChatMessages(100);
		if (result.data) {
			messages = result.data;
		}
		loaded = true;
		scheduleScroll();
	});

	async function handleSend() {
		const content = inputText.trim();
		if (!content || streaming) return;

		inputText = '';
		error = null;
		streaming = true;
		streamingContent = '';
		pendingUserMessage = content;
		scheduleScroll();

		await sendChatMessage(
			content,
			(chunk) => {
				streamingContent += chunk;
				scheduleScroll();
			},
			(resp: SendMessageResponse) => {
				pendingUserMessage = null;
				messages = [...messages, resp.user_message, resp.assistant_message];
				streamingContent = '';
				streaming = false;
				scheduleScroll();
			},
			(err: string) => {
				error = err;
				streaming = false;
				streamingContent = '';
				pendingUserMessage = null;
			}
		);
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			handleSend();
		}
	}

	async function handleClear() {
		clearing = true;
		const result = await clearChatHistory();
		if (result.error) {
			error = 'Failed to clear history';
		} else {
			messages = [];
		}
		showClearConfirm = false;
		clearing = false;
	}

	function formatTime(iso: string) {
		return new Date(iso).toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' });
	}

	function renderMarkdown(text: string): string {
		return marked.parse(text) as string;
	}
</script>

<svelte:head>
	<title>AI Coach — Behavioral Coaching</title>
</svelte:head>

<div class="dashboard-bg flex h-screen flex-col overflow-hidden">
	<!-- Header -->
	<header class="shrink-0 border-b border-white/[0.06] bg-white/[0.02] py-4">
		<div class="mx-auto flex max-w-3xl items-center justify-between px-6">
			<div>
				<a
					href={resolve('/dashboard')}
					class="text-sm text-indigo-300/50 transition hover:text-indigo-200"
				>
					&larr; Dashboard
				</a>
				<h1 class="mt-1 text-xl font-semibold text-white">AI Coach</h1>
			</div>
			<button
				type="button"
				onclick={() => (showClearConfirm = true)}
				disabled={messages.length === 0 || streaming}
				class="rounded-lg px-3 py-1.5 text-xs text-indigo-200/40 transition hover:bg-white/10 hover:text-red-300 disabled:pointer-events-none disabled:opacity-30"
			>
				Clear history
			</button>
		</div>
	</header>

	<!-- Clear confirm modal -->
	{#if showClearConfirm}
		<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
			<div
				in:fly={{ y: 10, duration: 200 }}
				class="mx-4 w-full max-w-sm rounded-2xl border border-white/10 bg-slate-900/90 p-6 shadow-xl"
			>
				<h2 class="text-base font-semibold text-white">Clear chat history?</h2>
				<p class="mt-2 text-sm text-indigo-200/50">
					All messages will be permanently deleted. This cannot be undone.
				</p>
				<div class="mt-5 flex gap-3">
					<button
						type="button"
						onclick={() => (showClearConfirm = false)}
						class="flex-1 rounded-lg border border-white/10 py-2 text-sm text-indigo-200/60 transition hover:bg-white/5"
					>
						Cancel
					</button>
					<button
						type="button"
						onclick={handleClear}
						disabled={clearing}
						class="flex-1 rounded-lg bg-red-500/20 py-2 text-sm text-red-300 transition hover:bg-red-500/30 disabled:opacity-50"
					>
						{clearing ? 'Clearing...' : 'Clear'}
					</button>
				</div>
			</div>
		</div>
	{/if}

	{#if !loaded}
		<div class="flex flex-1 items-center justify-center">
			<div class="flex items-center gap-3 text-indigo-300/60">
				<svg class="h-5 w-5 animate-spin" fill="none" viewBox="0 0 24 24">
					<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"
					></circle>
					<path
						class="opacity-75"
						fill="currentColor"
						d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
					></path>
				</svg>
				<span class="text-sm">Loading conversation...</span>
			</div>
		</div>
	{:else}
		<!-- Messages area — fills remaining height and scrolls internally -->
		<div bind:this={messagesEl} class="min-h-0 flex-1 overflow-y-auto px-4 py-6">
			<div class="mx-auto max-w-3xl space-y-4">
				{#if messages.length === 0 && !pendingUserMessage && !streaming}
					<div
						in:fly={{ y: 10, duration: 400, easing: cubicOut }}
						class="flex flex-col items-center py-20 text-center"
					>
						<div
							class="mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-indigo-500/10 ring-1 ring-indigo-400/20"
						>
							<svg
								class="h-7 w-7 text-indigo-300/60"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
								stroke-width="1.5"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									d="M8.625 12a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H8.25m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H12m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0h-.375M21 12c0 4.556-4.03 8.25-9 8.25a9.764 9.764 0 01-2.555-.337A5.972 5.972 0 015.41 20.97a5.969 5.969 0 01-.474-.065 4.48 4.48 0 00.978-2.025c.09-.457-.133-.901-.467-1.226C3.93 16.178 3 14.189 3 12c0-4.556 4.03-8.25 9-8.25s9 3.694 9 8.25z"
								/>
							</svg>
						</div>
						<p class="text-sm font-medium text-indigo-200/60">Start a conversation</p>
						<p class="mt-1 text-xs text-indigo-200/30">
							Ask about your goals, progress, or how you're feeling today.
						</p>
					</div>
				{/if}

				{#each messages as msg (msg.id)}
					<div
						in:fly={{ y: 8, duration: 300, easing: cubicOut }}
						class="flex {msg.role === 'user' ? 'justify-end' : 'justify-start'}"
					>
						{#if msg.role === 'assistant'}
							<div
								class="mr-3 mt-1 flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-indigo-500/20 ring-1 ring-indigo-400/20"
							>
								<svg
									class="h-3.5 w-3.5 text-indigo-300"
									fill="none"
									viewBox="0 0 24 24"
									stroke="currentColor"
									stroke-width="2"
								>
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09z"
									/>
								</svg>
							</div>
						{/if}
						<div class="max-w-[80%]">
							{#if msg.role === 'user'}
								<div
									class="rounded-2xl bg-indigo-600/30 px-4 py-3 text-sm leading-relaxed text-white ring-1 ring-indigo-500/30"
								>
									{msg.content}
								</div>
							{:else}
								<div
									class="prose-chat rounded-2xl bg-white/5 px-4 py-3 text-sm leading-relaxed text-indigo-100 ring-1 ring-white/10"
								>
									{@html renderMarkdown(msg.content)}
								</div>
							{/if}
							<p
								class="mt-1 text-xs text-indigo-200/25 {msg.role === 'user'
									? 'text-right'
									: 'text-left'}"
							>
								{formatTime(msg.created_at)}
							</p>
						</div>
					</div>
				{/each}

				<!-- Optimistic user message (shows while waiting for server round-trip) -->
				{#if pendingUserMessage}
					<div in:fly={{ y: 8, duration: 200, easing: cubicOut }} class="flex justify-end">
						<div class="max-w-[80%]">
							<div
								class="rounded-2xl bg-indigo-600/30 px-4 py-3 text-sm leading-relaxed text-white ring-1 ring-indigo-500/30"
							>
								{pendingUserMessage}
							</div>
						</div>
					</div>
				{/if}

				<!-- Streaming assistant bubble -->
				{#if streaming}
					<div in:fly={{ y: 8, duration: 200 }} class="flex justify-start">
						<div
							class="mr-3 mt-1 flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-indigo-500/20 ring-1 ring-indigo-400/20"
						>
							<svg
								class="h-3.5 w-3.5 text-indigo-300"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
								stroke-width="2"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09z"
								/>
							</svg>
						</div>
						<div class="max-w-[80%]">
							<div
								class="prose-chat rounded-2xl bg-white/5 px-4 py-3 text-sm leading-relaxed text-indigo-100 ring-1 ring-white/10"
							>
								{#if streamingContent}
									{@html renderMarkdown(streamingContent)}<span
										class="ml-0.5 inline-block h-3.5 w-0.5 animate-pulse bg-indigo-300"
									></span>
								{:else}
									<span class="flex items-center gap-1.5 text-indigo-300/50">
										<span
											class="h-1.5 w-1.5 animate-bounce rounded-full bg-indigo-400/60 [animation-delay:0ms]"
										></span>
										<span
											class="h-1.5 w-1.5 animate-bounce rounded-full bg-indigo-400/60 [animation-delay:150ms]"
										></span>
										<span
											class="h-1.5 w-1.5 animate-bounce rounded-full bg-indigo-400/60 [animation-delay:300ms]"
										></span>
									</span>
								{/if}
							</div>
						</div>
					</div>
				{/if}

				{#if error}
					<div
						in:fly={{ y: -5, duration: 200 }}
						class="mx-auto max-w-sm rounded-lg bg-red-500/10 px-4 py-2 text-center text-xs text-red-300 ring-1 ring-red-500/20"
					>
						{error}
					</div>
				{/if}
			</div>
		</div>

		<!-- Input area -->
		<div class="shrink-0 border-t border-white/[0.06] bg-white/[0.02] px-4 py-4">
			<div class="mx-auto flex max-w-3xl items-end gap-3">
				<textarea
					bind:value={inputText}
					onkeydown={handleKeydown}
					disabled={streaming}
					placeholder="Message your coach…"
					rows="1"
					class="flex-1 resize-none rounded-xl border border-white/10 bg-white/5 px-4 py-3 text-sm text-white placeholder-indigo-200/30 outline-none transition focus:border-indigo-500/50 focus:ring-1 focus:ring-indigo-500/30 disabled:opacity-50"
					style="field-sizing: content; max-height: 160px;"
				></textarea>
				<button
					type="button"
					onclick={handleSend}
					disabled={!inputText.trim() || streaming}
					class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-indigo-600/80 text-white transition hover:bg-indigo-500 disabled:pointer-events-none disabled:opacity-30"
					aria-label="Send message"
				>
					{#if streaming}
						<svg class="h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
							<circle
								class="opacity-25"
								cx="12"
								cy="12"
								r="10"
								stroke="currentColor"
								stroke-width="4"
							></circle>
							<path
								class="opacity-75"
								fill="currentColor"
								d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
							></path>
						</svg>
					{:else}
						<svg
							class="h-4 w-4"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="2"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M6 12L3.269 3.126A59.768 59.768 0 0121.485 12 59.77 59.77 0 013.27 20.876L5.999 12zm0 0h7.5"
							/>
						</svg>
					{/if}
				</button>
			</div>
			<p class="mt-2 text-center text-xs text-indigo-200/20">
				Press Enter to send · Shift+Enter for new line
			</p>
		</div>
	{/if}
</div>

<style>
	.dashboard-bg {
		background: linear-gradient(135deg, #0f172a 0%, #1e1b4b 40%, #312e81 100%);
	}

	/* Markdown prose styles scoped to chat bubbles */
	:global(.prose-chat p) {
		margin: 0 0 0.5em;
	}
	:global(.prose-chat p:last-child) {
		margin-bottom: 0;
	}
	:global(.prose-chat strong) {
		color: #e0e7ff;
		font-weight: 600;
	}
	:global(.prose-chat em) {
		color: #c7d2fe;
	}
	:global(.prose-chat ul),
	:global(.prose-chat ol) {
		margin: 0.4em 0 0.6em 1.2em;
		padding: 0;
	}
	:global(.prose-chat li) {
		margin-bottom: 0.2em;
	}
	:global(.prose-chat code) {
		background: rgba(99, 102, 241, 0.15);
		border-radius: 4px;
		padding: 0.1em 0.35em;
		font-size: 0.85em;
		color: #a5b4fc;
	}
	:global(.prose-chat pre) {
		background: rgba(0, 0, 0, 0.3);
		border-radius: 8px;
		padding: 0.75em 1em;
		overflow-x: auto;
		margin: 0.5em 0;
	}
	:global(.prose-chat pre code) {
		background: none;
		padding: 0;
		font-size: 0.82em;
		color: #c7d2fe;
	}
	:global(.prose-chat h1),
	:global(.prose-chat h2),
	:global(.prose-chat h3) {
		color: #e0e7ff;
		font-weight: 600;
		margin: 0.6em 0 0.3em;
	}
	:global(.prose-chat h1) {
		font-size: 1.1em;
	}
	:global(.prose-chat h2) {
		font-size: 1em;
	}
	:global(.prose-chat h3) {
		font-size: 0.95em;
	}
	:global(.prose-chat blockquote) {
		border-left: 2px solid rgba(99, 102, 241, 0.4);
		padding-left: 0.75em;
		margin: 0.5em 0;
		color: #a5b4fc;
	}
	:global(.prose-chat a) {
		color: #818cf8;
		text-decoration: underline;
	}
</style>
