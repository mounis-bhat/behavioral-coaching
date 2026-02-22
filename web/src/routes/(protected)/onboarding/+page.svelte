<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { onMount, tick } from 'svelte';
	import { SvelteSet } from 'svelte/reactivity';
	import { fly } from 'svelte/transition';
	import { cubicOut } from 'svelte/easing';
	import {
		fetchProfile,
		createProfile,
		updateProfile,
		generatePlan,
		streamOnboardingChat
	} from '$lib/api';
	import { user } from '$lib/stores/auth';
	import type { OnboardingChatMessage, OnboardingChatResponse } from '$lib/api';

	let step = $state(1);
	let loading = $state(false);
	let error = $state<string | null>(null);
	let initialLoading = $state(true);

	let visible = $state(false);
	let stepDirection = $state(1);
	let stepKey = $state(1);
	let progressPulse = $state(false);
	let celebrating = $state(false);

	const totalSteps = 7;

	let firstName = $derived($user?.name?.split(' ')[0] ?? '');

	let stepMeta = $derived<Record<number, { title: string; subtitle: string; icon: string }>>({
		1: {
			title: firstName ? `What drives you, ${firstName}?` : 'What drives you?',
			subtitle: 'Every great change starts with knowing what you want.',
			icon: '🎯'
		},
		2: {
			title: 'Your world, your rules',
			subtitle: firstName
				? `We'll work with your schedule, not against it, ${firstName}.`
				: "We'll work with your schedule, not against it.",
			icon: '⏱️'
		},
		3: {
			title: firstName ? `Know yourself, ${firstName}` : 'Know yourself',
			subtitle: 'Honest self-awareness is the foundation of growth.',
			icon: '🧠'
		},
		4: {
			title: 'Set your challenge',
			subtitle: 'The right difficulty turns effort into momentum.',
			icon: '⚡'
		},
		5: {
			title: firstName ? `Let's talk, ${firstName}` : "Let's talk",
			subtitle: 'A brief conversation so we can truly understand you.',
			icon: '💬'
		},
		6: {
			title: 'How are you feeling?',
			subtitle: "There's no wrong answer — this helps us meet you where you are.",
			icon: '❤️'
		},
		7: {
			title: firstName ? `Almost there, ${firstName}` : 'Your plan style',
			subtitle: 'Choose how you want your days structured.',
			icon: '📋'
		}
	});

	const categoryPillColors: Record<string, string> = {
		health: 'bg-green-400/20 text-green-300 ring-1 ring-green-400/30',
		mindset: 'bg-purple-400/20 text-purple-300 ring-1 ring-purple-400/30',
		discipline: 'bg-orange-400/20 text-orange-300 ring-1 ring-orange-400/30',
		relationships: 'bg-blue-400/20 text-blue-300 ring-1 ring-blue-400/30',
		productivity: 'bg-yellow-400/20 text-yellow-300 ring-1 ring-yellow-400/30'
	};

	// Step 1: Goals
	const categories = [
		{ id: 'health', label: 'Health' },
		{ id: 'mindset', label: 'Mindset' },
		{ id: 'discipline', label: 'Discipline' },
		{ id: 'relationships', label: 'Relationships' },
		{ id: 'productivity', label: 'Productivity' }
	];
	const selectedCategories = new SvelteSet<string>();
	let freeTextGoals = $state('');

	// Step 2: Constraints
	let timeAvailability = $state('1hr');
	let limitations = $state('');

	// Step 3: Self-assessment
	let motivation = $state(5);
	let stress = $state(5);
	let energy = $state(5);

	// Step 4: Difficulty
	let difficulty = $state(5);

	const difficultyLabels: Record<number, string> = {
		1: 'Very easy - just getting started',
		2: 'Easy - building basic habits',
		3: 'Gentle - light daily structure',
		4: 'Moderate-easy - some challenge',
		5: 'Moderate - balanced approach',
		6: 'Moderate-hard - pushing limits',
		7: 'Challenging - significant effort',
		8: 'Hard - intense commitment',
		9: 'Very hard - near maximum effort',
		10: 'Extreme - all-out dedication'
	};

	// Step 5: AI Conversation
	let chatHistory = $state<OnboardingChatMessage[]>([]);
	let chatInput = $state('');
	let chatLoading = $state(false);
	let chatComplete = $state(false);
	let streamingMessage = $state('');
	let chatSummary = $state<Record<string, unknown> | undefined>(undefined);
	let recommendedDeliveryMode = $state('');
	let recommendedEmotionalState = $state('');
	let chatStarted = $state(false);

	// Step 6: Emotional State
	let selectedEmotionalState = $state('');

	// Step 7: Delivery Mode
	let selectedDeliveryMode = $state('flexible_list');

	let chatMessagesEl: HTMLDivElement | null = null;

	const emotionalStates = [
		{
			id: 'burned_out',
			label: 'Burned Out',
			description: 'Depleted energy, feeling empty, everything feels like too much effort'
		},
		{
			id: 'anxious',
			label: 'Anxious',
			description: 'Worried, restless, difficulty relaxing, mind racing with concerns'
		},
		{
			id: 'stuck',
			label: 'Stuck',
			description: 'Lack of direction, procrastinating, feeling trapped or unable to move forward'
		}
	];

	const deliveryModes = [
		{
			id: 'strict_schedule',
			label: 'Strict Schedule',
			description:
				'Tasks with specific time slots throughout your day. Best if you thrive with clear structure and time blocks.'
		},
		{
			id: 'flexible_list',
			label: 'Flexible List',
			description:
				'An ordered list of tasks to complete at your own pace. Best if you want guidance but value flexibility.'
		},
		{
			id: 'gentle_structure',
			label: 'Gentle Structure',
			description:
				'3 daily anchors plus tiny tasks focused on self-care. Best if you need low-pressure support to rebuild momentum.'
		}
	];

	const confettiColors = ['#818cf8', '#a78bfa', '#06b6d4', '#22d3ee', '#f472b6', '#fbbf24', '#34d399'];

	function generateConfetti() {
		return Array.from({ length: 40 }, (_, i) => ({
			x: Math.random() * 100,
			color: confettiColors[Math.floor(Math.random() * confettiColors.length)],
			size: Math.random() * 8 + 4,
			delay: Math.random() * 0.8,
			rotation: Math.random() * 720 - 360
		}));
	}

	let confettiPieces = $state<ReturnType<typeof generateConfetti>>([]);

	onMount(async () => {
		const result = await fetchProfile();
		if (result.data && result.data.onboarding_completed) {
			await goto(resolve('/dashboard'));
			return;
		}
		initialLoading = false;
		visible = true;
	});

	function toggleCategory(id: string) {
		if (selectedCategories.has(id)) {
			selectedCategories.delete(id);
		} else {
			selectedCategories.add(id);
		}
	}

	function canAdvance(): boolean {
		if (step === 1) return selectedCategories.size > 0;
		return true;
	}

	function triggerPulse() {
		progressPulse = true;
		setTimeout(() => (progressPulse = false), 600);
	}

	function next() {
		error = null;
		if (!canAdvance()) {
			error = 'Please select at least one goal category.';
			return;
		}
		if (step < totalSteps) {
			stepDirection = 1;
			step++;
			stepKey = step;
			triggerPulse();
			if (step === 5 && !chatStarted) {
				startChat();
			}
		}
	}

	function back() {
		error = null;
		if (step > 1) {
			stepDirection = -1;
			step--;
			stepKey = step;
			triggerPulse();
		}
	}

	function getOnboardingContext() {
		return {
			goals: {
				categories: [...selectedCategories],
				free_text: freeTextGoals || undefined
			},
			constraints: {
				time_availability: timeAvailability,
				limitations: limitations || undefined
			},
			psychological_state: {
				motivation,
				stress,
				energy
			}
		};
	}

	async function startChat() {
		chatStarted = true;
		chatLoading = true;
		streamingMessage = '';
		error = null;
		await streamOnboardingChat(
			'',
			[],
			getOnboardingContext(),
			(chunk: string) => {
				streamingMessage += chunk;
				scrollChatToBottom();
			},
			(response: OnboardingChatResponse) => {
				chatHistory = response.history;
				chatComplete = response.is_complete;
				streamingMessage = '';
				if (response.is_complete) applyRecommendations(response);
			},
			(err: string) => {
				error = err;
				streamingMessage = '';
			}
		);
		chatLoading = false;
		await tick();
		scrollChatToBottom();
	}

	async function sendChatMessage() {
		if (!chatInput.trim() || chatLoading || chatComplete) return;
		const message = chatInput.trim();
		const optimisticMessage: OnboardingChatMessage = { role: 'user', content: message };
		const historySnapshot = chatHistory;
		chatHistory = [...historySnapshot, optimisticMessage];
		chatInput = '';
		chatLoading = true;
		streamingMessage = '';
		await tick();
		scrollChatToBottom();
		error = null;
		await streamOnboardingChat(
			message,
			historySnapshot,
			getOnboardingContext(),
			(chunk: string) => {
				streamingMessage += chunk;
				scrollChatToBottom();
			},
			(response: OnboardingChatResponse) => {
				chatHistory = ensureUserMessage(response.history, optimisticMessage);
				chatComplete = response.is_complete;
				streamingMessage = '';
				if (response.is_complete) applyRecommendations(response);
			},
			(err: string) => {
				error = err;
				streamingMessage = '';
			}
		);
		chatLoading = false;
		await tick();
		scrollChatToBottom();
	}

	function applyRecommendations(data: OnboardingChatResponse) {
		chatSummary = data.summary;
		if (data.recommended_delivery_mode) {
			recommendedDeliveryMode = data.recommended_delivery_mode;
			selectedDeliveryMode = data.recommended_delivery_mode;
		}
		if (data.recommended_emotional_state) {
			recommendedEmotionalState = data.recommended_emotional_state;
			selectedEmotionalState = data.recommended_emotional_state;
		}
	}

	function skipChat() {
		chatComplete = true;
		stepDirection = 1;
		step++;
		stepKey = step;
	}

	function scrollChatToBottom() {
		if (chatMessagesEl) {
			chatMessagesEl.scrollTop = chatMessagesEl.scrollHeight;
		}
	}

	function setChatMessagesEl(node: HTMLDivElement) {
		chatMessagesEl = node;
		return () => {
			if (chatMessagesEl === node) {
				chatMessagesEl = null;
			}
		};
	}

	function handleChatKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			sendChatMessage();
		}
	}

	function ensureUserMessage(history: OnboardingChatMessage[], message: OnboardingChatMessage) {
		for (let i = history.length - 1; i >= 0; i--) {
			if (history[i].role === 'user') {
				return history[i].content === message.content ? history : [...history, message];
			}
		}
		return [...history, message];
	}

	async function complete() {
		error = null;
		loading = true;
		try {
			const goals = {
				categories: [...selectedCategories],
				free_text: freeTextGoals || undefined
			};
			const constraints = {
				time_availability: timeAvailability,
				limitations: limitations || undefined
			};
			const psychological_state = {
				motivation,
				stress,
				energy
			};

			const createResult = await createProfile({
				goals,
				constraints,
				psychological_state,
				difficulty_level: difficulty,
				delivery_mode: selectedDeliveryMode,
				emotional_state: selectedEmotionalState,
				ai_profile_summary: chatSummary ?? {}
			});
			if (createResult.error) throw new Error(createResult.error);

			const updateResult = await updateProfile({ onboarding_completed: true });
			if (updateResult.error) throw new Error(updateResult.error);

			const planResult = await generatePlan();
			if (planResult.error) throw new Error(planResult.error);

			celebrating = true;
			confettiPieces = generateConfetti();
			await new Promise((r) => setTimeout(r, 1800));

			await goto(resolve('/dashboard'));
		} catch (e) {
			error = e instanceof Error ? e.message : 'Something went wrong';
			celebrating = false;
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Set up your profile — Behavioral Coaching</title>
</svelte:head>

{#if initialLoading}
	<div class="onboarding-bg flex min-h-screen items-center justify-center">
		<div class="aurora">
			<div class="aurora-layer aurora-1"></div>
			<div class="aurora-layer aurora-2"></div>
			<div class="aurora-layer aurora-3"></div>
		</div>
		<div class="orb orb-indigo"></div>
		<div class="orb orb-purple"></div>
		<div class="orb orb-teal"></div>
		<div
			class="h-8 w-8 animate-spin rounded-full border-2 border-indigo-400/30 border-t-indigo-400"
		></div>
	</div>
{:else}
	<div class="onboarding-bg min-h-screen px-6 py-8">
		<div class="aurora">
			<div class="aurora-layer aurora-1"></div>
			<div class="aurora-layer aurora-2"></div>
			<div class="aurora-layer aurora-3"></div>
		</div>
		<div class="orb orb-indigo"></div>
		<div class="orb orb-purple"></div>
		<div class="orb orb-teal"></div>

		{#if celebrating}
			<div class="confetti-overlay">
				{#each confettiPieces as piece}
					<div
						class="confetti-piece"
						style="--x: {piece.x}%; --color: {piece.color}; --size: {piece.size}px; --delay: {piece.delay}s; --rotation: {piece.rotation}deg;"
					></div>
				{/each}
				<div class="celebration-text">
					{firstName ? `You're all set, ${firstName}!` : "You're all set!"}
				</div>
			</div>
		{/if}

		{#if visible}
			<div class="relative z-10 mx-auto max-w-lg">
				<!-- Step indicator badge -->
				<div class="mb-4 flex items-center justify-center gap-2">
					<span class="text-2xl">{stepMeta[step].icon}</span>
					<span
						class="rounded-full bg-white/10 px-3 py-1 text-xs font-medium tracking-wide text-indigo-200/70"
					>
						Step {step} of {totalSteps}
					</span>
				</div>

				<!-- Motivational title + subtitle -->
				<div class="mb-2 text-center">
					<h1 class="text-2xl font-bold text-white">{stepMeta[step].title}</h1>
					<p class="mt-1 text-sm text-indigo-200/60">{stepMeta[step].subtitle}</p>
				</div>

				<!-- Progress bar -->
				<div class="mt-6 mb-8 h-1.5 w-full rounded-full bg-white/10">
					<div
						class="progress-fill h-1.5 rounded-full transition-all duration-500"
						class:pulse={progressPulse}
						style="width: {(step / totalSteps) * 100}%"
					></div>
				</div>

				<!-- Error banner -->
				{#if error}
					<div
						class="mb-4 rounded-lg border border-red-400/20 bg-red-500/10 px-4 py-3 text-sm text-red-300"
					>
						{error}
					</div>
				{/if}

				<!-- Step content with transitions -->
				{#key stepKey}
					<section in:fly={{ x: stepDirection * 80, duration: 400, easing: cubicOut }}>
						{#if step === 1}
							<div class="stagger-in rounded-2xl border border-white/10 bg-white/5 p-6 backdrop-blur-sm">
								<p class="stagger-item mb-4 text-sm text-indigo-200/60" style="--stagger: 0">
									Select the areas you want to focus on.
								</p>
								<div class="stagger-item flex flex-wrap gap-2" style="--stagger: 1">
									{#each categories as cat (cat.id)}
										<button
											type="button"
											onclick={() => toggleCategory(cat.id)}
											class="rounded-full px-4 py-2 text-sm font-medium transition {selectedCategories.has(
												cat.id
											)
												? categoryPillColors[cat.id]
												: 'border border-white/10 bg-white/5 text-indigo-200/70 hover:bg-white/10'}"
										>
											{cat.label}
										</button>
									{/each}
								</div>
								<div class="stagger-item mt-5" style="--stagger: 2">
									<label
										for="free-goals"
										class="mb-1.5 block text-sm font-medium text-indigo-200/80"
									>
										Additional goals
										<span class="text-indigo-200/40">(optional)</span>
									</label>
									<textarea
										id="free-goals"
										bind:value={freeTextGoals}
										rows={3}
										placeholder="Describe any specific goals..."
										class="w-full rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm text-white placeholder-indigo-300/30 focus:border-indigo-400/40 focus:ring-1 focus:ring-indigo-400/30 focus:outline-none"
									></textarea>
								</div>
							</div>
						{:else if step === 2}
							<div class="stagger-in rounded-2xl border border-white/10 bg-white/5 p-6 backdrop-blur-sm">
								<div class="stagger-item mb-5" style="--stagger: 0">
									<label for="time" class="mb-1.5 block text-sm font-medium text-indigo-200/80">
										Daily time available
									</label>
									<select
										id="time"
										bind:value={timeAvailability}
										class="dark-select w-full rounded-lg border border-white/10 bg-white/5 px-3 py-2.5 pr-10 text-sm text-white focus:border-indigo-400/40 focus:ring-1 focus:ring-indigo-400/30 focus:outline-none"
									>
										<option value="30min">30 minutes</option>
										<option value="1hr">1 hour</option>
										<option value="2hr">2 hours</option>
										<option value="3hr+">3+ hours</option>
									</select>
								</div>
								<div class="stagger-item" style="--stagger: 1">
									<label
										for="limitations"
										class="mb-1.5 block text-sm font-medium text-indigo-200/80"
									>
										Limitations
										<span class="text-indigo-200/40">(optional)</span>
									</label>
									<textarea
										id="limitations"
										bind:value={limitations}
										rows={3}
										placeholder="Any physical, schedule, or other constraints..."
										class="w-full rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm text-white placeholder-indigo-300/30 focus:border-indigo-400/40 focus:ring-1 focus:ring-indigo-400/30 focus:outline-none"
									></textarea>
								</div>
							</div>
						{:else if step === 3}
							<div class="stagger-in rounded-2xl border border-white/10 bg-white/5 p-6 backdrop-blur-sm">
								<div class="space-y-6">
									<div class="stagger-item" style="--stagger: 0">
										<label
											for="motivation"
											class="mb-2 flex items-center justify-between text-sm font-medium text-indigo-200/80"
										>
											<span>Motivation</span>
											<span
												class="rounded-full bg-indigo-500/20 px-2.5 py-0.5 text-xs font-semibold text-indigo-300"
												>{motivation}</span
											>
										</label>
										<input
											id="motivation"
											type="range"
											min="1"
											max="10"
											bind:value={motivation}
											class="onboarding-slider"
										/>
									</div>
									<div class="stagger-item" style="--stagger: 1">
										<label
											for="stress"
											class="mb-2 flex items-center justify-between text-sm font-medium text-indigo-200/80"
										>
											<span>Stress level</span>
											<span
												class="rounded-full bg-indigo-500/20 px-2.5 py-0.5 text-xs font-semibold text-indigo-300"
												>{stress}</span
											>
										</label>
										<input
											id="stress"
											type="range"
											min="1"
											max="10"
											bind:value={stress}
											class="onboarding-slider"
										/>
									</div>
									<div class="stagger-item" style="--stagger: 2">
										<label
											for="energy"
											class="mb-2 flex items-center justify-between text-sm font-medium text-indigo-200/80"
										>
											<span>Energy</span>
											<span
												class="rounded-full bg-indigo-500/20 px-2.5 py-0.5 text-xs font-semibold text-indigo-300"
												>{energy}</span
											>
										</label>
										<input
											id="energy"
											type="range"
											min="1"
											max="10"
											bind:value={energy}
											class="onboarding-slider"
										/>
									</div>
								</div>
							</div>
						{:else if step === 4}
							<div class="stagger-in rounded-2xl border border-white/10 bg-white/5 p-6 backdrop-blur-sm">
								<div class="stagger-item" style="--stagger: 0">
									<label
										for="difficulty"
										class="mb-2 flex items-center justify-between text-sm font-medium text-indigo-200/80"
									>
										<span>Difficulty</span>
										<span
											class="rounded-full bg-indigo-500/20 px-2.5 py-0.5 text-xs font-semibold text-indigo-300"
											>{difficulty}</span
										>
									</label>
									<input
										id="difficulty"
										type="range"
										min="1"
										max="10"
										bind:value={difficulty}
										class="onboarding-slider"
									/>
									<p class="mt-3 text-sm text-indigo-200/60">{difficultyLabels[difficulty]}</p>
								</div>
							</div>
						{:else if step === 5}
							<div class="stagger-in">
								<div
									class="stagger-item mb-4 overflow-hidden rounded-2xl border border-white/6 bg-white/3"
									style="--stagger: 0"
								>
									<div
										{@attach setChatMessagesEl}
										class="chat-scroll max-h-80 space-y-3 overflow-y-auto p-4"
									>
										{#each chatHistory as msg, idx (idx)}
											<div class="flex {msg.role === 'user' ? 'justify-end' : 'justify-start'}">
												<div
													class="max-w-[80%] rounded-2xl px-4 py-2.5 text-sm {msg.role === 'user'
														? 'bg-indigo-500 text-white'
														: 'bg-white/10 text-indigo-100'}"
												>
													{msg.content}
												</div>
											</div>
										{/each}
										{#if chatLoading}
											<div class="flex justify-start">
												{#if streamingMessage}
													<div
														class="max-w-[80%] rounded-2xl bg-white/10 px-4 py-2.5 text-sm text-indigo-100"
													>
														{streamingMessage}<span
															class="ml-0.5 inline-block h-3.5 w-0.5 animate-pulse bg-indigo-300/70 align-middle"
														></span>
													</div>
												{:else}
													<div
														class="rounded-2xl bg-white/10 px-4 py-2.5 text-sm text-indigo-300/50"
													>
														<span class="inline-flex gap-1">
															<span class="animate-bounce">.</span>
															<span class="animate-bounce" style="animation-delay: 0.1s"
																>.</span
															>
															<span class="animate-bounce" style="animation-delay: 0.2s"
																>.</span
															>
														</span>
													</div>
												{/if}
											</div>
										{/if}
									</div>
								</div>

								{#if !chatComplete}
									<div class="stagger-item flex gap-2" style="--stagger: 1">
										<input
											type="text"
											bind:value={chatInput}
											onkeydown={handleChatKeydown}
											placeholder="Type your response..."
											disabled={chatLoading || !chatStarted}
											class="flex-1 rounded-lg border border-white/10 bg-white/5 px-3 py-2.5 text-sm text-white placeholder-indigo-300/30 focus:border-indigo-400/40 focus:ring-1 focus:ring-indigo-400/30 focus:outline-none disabled:opacity-50"
										/>
										<button
											type="button"
											onclick={sendChatMessage}
											disabled={chatLoading || !chatInput.trim()}
											class="rounded-lg bg-white px-4 py-2.5 text-sm font-semibold text-slate-900 transition hover:bg-indigo-100 disabled:opacity-50"
										>
											Send
										</button>
									</div>
								{/if}
							</div>
						{:else if step === 6}
							<div class="stagger-in space-y-3">
								{#each emotionalStates as state, idx (state.id)}
									<button
										type="button"
										onclick={() => (selectedEmotionalState = state.id)}
										class="stagger-item relative w-full rounded-xl p-4 text-left transition {selectedEmotionalState ===
										state.id
											? 'border border-indigo-400/30 bg-indigo-500/10 ring-1 ring-indigo-400/20'
											: 'border border-white/8 bg-white/3 hover:bg-white/6'}"
										style="--stagger: {idx}"
									>
										{#if recommendedEmotionalState === state.id}
											<span
												class="absolute -top-2 right-3 rounded-full bg-indigo-500/20 px-2.5 py-0.5 text-xs font-medium text-indigo-300"
											>
												Recommended for you
											</span>
										{/if}
										<p class="font-medium text-white">{state.label}</p>
										<p class="mt-1 text-sm text-indigo-200/60">{state.description}</p>
									</button>
								{/each}
								<button
									type="button"
									onclick={() => (selectedEmotionalState = '')}
									class="stagger-item w-full rounded-xl p-4 text-left transition {selectedEmotionalState === ''
										? 'border border-indigo-400/30 bg-indigo-500/10 ring-1 ring-indigo-400/20'
										: 'border border-white/8 bg-white/3 hover:bg-white/6'}"
									style="--stagger: {emotionalStates.length}"
								>
									<p class="font-medium text-white">None of these</p>
									<p class="mt-1 text-sm text-indigo-200/60">
										I'm doing okay, just looking to improve.
									</p>
								</button>
							</div>
						{:else if step === 7}
							<div class="stagger-in space-y-3">
								{#each deliveryModes as mode, idx (mode.id)}
									<button
										type="button"
										onclick={() => (selectedDeliveryMode = mode.id)}
										class="stagger-item relative w-full rounded-xl p-4 text-left transition {selectedDeliveryMode ===
										mode.id
											? 'border border-indigo-400/30 bg-indigo-500/10 ring-1 ring-indigo-400/20'
											: 'border border-white/8 bg-white/3 hover:bg-white/6'}"
										style="--stagger: {idx}"
									>
										{#if recommendedDeliveryMode === mode.id}
											<span
												class="absolute -top-2 right-3 rounded-full bg-indigo-500/20 px-2.5 py-0.5 text-xs font-medium text-indigo-300"
											>
												Recommended for you
											</span>
										{/if}
										<p class="font-medium text-white">{mode.label}</p>
										<p class="mt-1 text-sm text-indigo-200/60">{mode.description}</p>
									</button>
								{/each}
							</div>
						{/if}
					</section>
				{/key}

				<!-- Navigation buttons -->
				<div class="mt-8 flex items-center justify-between">
					{#if step > 1}
						<button
							type="button"
							onclick={back}
							disabled={loading}
							class="rounded-lg border border-white/10 bg-white/5 px-5 py-2.5 text-sm text-indigo-200/70 transition hover:bg-white/10 disabled:opacity-50"
						>
							Back
						</button>
					{:else}
						<div></div>
					{/if}

					{#if step === 5}
						<div class="flex items-center gap-3">
							{#if !chatComplete}
								<button
									type="button"
									onclick={skipChat}
									class="text-sm text-indigo-300/50 transition hover:text-indigo-200"
								>
									Skip
								</button>
							{:else}
								<button
									type="button"
									onclick={next}
									class="rounded-lg bg-white px-5 py-2.5 text-sm font-semibold text-slate-900 transition hover:bg-indigo-100"
								>
									Continue
								</button>
							{/if}
						</div>
					{:else if step < totalSteps}
						<button
							type="button"
							onclick={next}
							class="rounded-lg bg-white px-5 py-2.5 text-sm font-semibold text-slate-900 transition hover:bg-indigo-100"
						>
							Next
						</button>
					{:else}
						<button
							type="button"
							onclick={complete}
							disabled={loading}
							class="rounded-lg bg-white px-5 py-2.5 text-sm font-semibold text-slate-900 transition hover:bg-indigo-100 disabled:opacity-50"
						>
							{loading ? 'Setting up...' : firstName ? `Launch ${firstName}'s plan` : 'Complete setup'}
						</button>
					{/if}
				</div>
			</div>
		{/if}
	</div>
{/if}

<style>
	.onboarding-bg {
		background: linear-gradient(135deg, #0f172a 0%, #1e1b4b 40%, #312e81 100%);
		position: relative;
		overflow: hidden;
	}

	/* Aurora layers */
	.aurora {
		position: fixed;
		inset: 0;
		pointer-events: none;
		z-index: 0;
	}

	.aurora-layer {
		position: absolute;
		width: 150%;
		height: 150%;
		border-radius: 50%;
		opacity: 0.15;
	}

	.aurora-1 {
		background: radial-gradient(ellipse at 30% 20%, rgba(129, 140, 248, 0.4), transparent 70%);
		animation: aurora-drift 18s ease-in-out infinite;
	}

	.aurora-2 {
		background: radial-gradient(ellipse at 70% 60%, rgba(167, 139, 250, 0.3), transparent 70%);
		animation: aurora-drift 15s ease-in-out infinite reverse;
		animation-delay: -5s;
	}

	.aurora-3 {
		background: radial-gradient(ellipse at 50% 80%, rgba(6, 182, 212, 0.25), transparent 70%);
		animation: aurora-drift 20s ease-in-out infinite;
		animation-delay: -10s;
	}

	@keyframes aurora-drift {
		0%, 100% {
			transform: translate(0, 0) scale(1);
		}
		33% {
			transform: translate(5%, -8%) scale(1.05);
		}
		66% {
			transform: translate(-3%, 5%) scale(0.95);
		}
	}

	/* Floating blobs */
	.orb {
		position: fixed;
		border-radius: 50%;
		filter: blur(120px);
		opacity: 0.2;
		pointer-events: none;
		z-index: 0;
	}

	.orb-indigo {
		width: 500px;
		height: 500px;
		background: #818cf8;
		top: -150px;
		left: -150px;
		animation: float 8s ease-in-out infinite;
	}

	.orb-purple {
		width: 450px;
		height: 450px;
		background: #a78bfa;
		top: 30%;
		right: -100px;
		animation: float 12s ease-in-out infinite reverse;
	}

	.orb-teal {
		width: 400px;
		height: 400px;
		background: #06b6d4;
		bottom: -100px;
		left: 30%;
		animation: float 16s ease-in-out infinite;
	}

	@keyframes float {
		0%, 100% {
			transform: translateY(0) translateX(0);
		}
		33% {
			transform: translateY(-30px) translateX(15px);
		}
		66% {
			transform: translateY(15px) translateX(-20px);
		}
	}

	/* Progress bar */
	.progress-fill {
		background: linear-gradient(90deg, #818cf8, #a78bfa);
		box-shadow: 0 0 12px rgba(129, 140, 248, 0.5);
	}

	.progress-fill.pulse {
		animation: progress-pulse 0.6s ease-out;
	}

	@keyframes progress-pulse {
		0% {
			box-shadow: 0 0 12px rgba(129, 140, 248, 0.5);
		}
		50% {
			box-shadow: 0 0 24px rgba(129, 140, 248, 0.9), 0 0 40px rgba(167, 139, 250, 0.4);
		}
		100% {
			box-shadow: 0 0 12px rgba(129, 140, 248, 0.5);
		}
	}

	/* Stagger animations */
	.stagger-item {
		opacity: 0;
		animation: stagger-fade-up 0.5s cubic-bezier(0.16, 1, 0.3, 1) forwards;
		animation-delay: calc(var(--stagger, 0) * 70ms + 100ms);
	}

	@keyframes stagger-fade-up {
		from {
			opacity: 0;
			transform: translateY(12px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}

	/* Confetti celebration */
	.confetti-overlay {
		position: fixed;
		inset: 0;
		z-index: 50;
		pointer-events: none;
		overflow: hidden;
	}

	.confetti-piece {
		position: absolute;
		top: -10vh;
		left: var(--x);
		width: var(--size);
		height: var(--size);
		background: var(--color);
		border-radius: 2px;
		animation: confetti-fall 1.8s cubic-bezier(0.25, 0.46, 0.45, 0.94) forwards;
		animation-delay: var(--delay);
		opacity: 0;
	}

	@keyframes confetti-fall {
		0% {
			transform: translateY(-10vh) rotate(0deg);
			opacity: 1;
		}
		100% {
			transform: translateY(110vh) rotate(var(--rotation));
			opacity: 0;
		}
	}

	.celebration-text {
		position: absolute;
		top: 50%;
		left: 50%;
		transform: translate(-50%, -50%) scale(0.8);
		font-size: 2rem;
		font-weight: 700;
		color: white;
		text-align: center;
		white-space: nowrap;
		opacity: 0;
		animation: celebration-appear 0.6s cubic-bezier(0.16, 1, 0.3, 1) 0.3s forwards;
		text-shadow: 0 0 40px rgba(129, 140, 248, 0.5);
	}

	@keyframes celebration-appear {
		from {
			opacity: 0;
			transform: translate(-50%, -50%) scale(0.8);
		}
		to {
			opacity: 1;
			transform: translate(-50%, -50%) scale(1);
		}
	}

	/* Range slider */
	.onboarding-slider {
		-webkit-appearance: none;
		appearance: none;
		width: 100%;
		height: 6px;
		border-radius: 9999px;
		background: rgba(255, 255, 255, 0.1);
		outline: none;
	}
	.onboarding-slider::-webkit-slider-thumb {
		-webkit-appearance: none;
		appearance: none;
		width: 20px;
		height: 20px;
		border-radius: 50%;
		background: #818cf8;
		cursor: pointer;
		box-shadow: 0 0 8px rgba(129, 140, 248, 0.5);
		transition: box-shadow 0.2s;
	}
	.onboarding-slider::-webkit-slider-thumb:hover {
		box-shadow: 0 0 16px rgba(129, 140, 248, 0.7);
	}
	.onboarding-slider::-moz-range-thumb {
		width: 20px;
		height: 20px;
		border-radius: 50%;
		background: #818cf8;
		cursor: pointer;
		border: none;
		box-shadow: 0 0 8px rgba(129, 140, 248, 0.5);
	}
	.onboarding-slider::-moz-range-track {
		height: 6px;
		border-radius: 9999px;
		background: rgba(255, 255, 255, 0.1);
	}

	/* Select dropdown */
	.dark-select {
		-webkit-appearance: none;
		appearance: none;
		color-scheme: dark;
		background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%3E%3Cpath fill='%23818cf8' d='M6 8L1 3h10z'/%3E%3C/svg%3E");
		background-repeat: no-repeat;
		background-position: right 12px center;
	}

	/* Chat scrollbar */
	.chat-scroll::-webkit-scrollbar {
		width: 4px;
	}
	.chat-scroll::-webkit-scrollbar-track {
		background: transparent;
	}
	.chat-scroll::-webkit-scrollbar-thumb {
		background: rgba(129, 140, 248, 0.2);
		border-radius: 9999px;
	}
</style>
