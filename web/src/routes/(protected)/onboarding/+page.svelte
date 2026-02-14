<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { onMount, tick } from 'svelte';
	import { SvelteSet } from 'svelte/reactivity';
	import {
		fetchProfile,
		createProfile,
		updateProfile,
		generatePlan,
		onboardingChat
	} from '$lib/api';
	import type { OnboardingChatMessage, OnboardingChatResponse } from '$lib/api-types';

	let step = $state(1);
	let loading = $state(false);
	let error = $state<string | null>(null);
	let initialLoading = $state(true);

	const totalSteps = 7;

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

	onMount(async () => {
		const result = await fetchProfile();
		if (result.data && result.data.onboarding_completed) {
			await goto(resolve('/dashboard'));
			return;
		}
		initialLoading = false;
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

	function next() {
		error = null;
		if (!canAdvance()) {
			error = 'Please select at least one goal category.';
			return;
		}
		if (step < totalSteps) {
			step++;
			if (step === 5 && !chatStarted) {
				startChat();
			}
		}
	}

	function back() {
		error = null;
		if (step > 1) step--;
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
		error = null;
		try {
			const result = await onboardingChat('', [], getOnboardingContext());
			if (result.error) throw new Error(result.error);
			if (result.data) {
				chatHistory = result.data.history;
				chatComplete = result.data.is_complete;
				if (result.data.is_complete) {
					applyRecommendations(result.data);
				}
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to start conversation';
		} finally {
			chatLoading = false;
			await tick();
			scrollChatToBottom();
		}
	}

	async function sendChatMessage() {
		if (!chatInput.trim() || chatLoading || chatComplete) return;
		const message = chatInput.trim();
		chatInput = '';
		chatLoading = true;
		error = null;
		try {
			const result = await onboardingChat(message, chatHistory, getOnboardingContext());
			if (result.error) throw new Error(result.error);
			if (result.data) {
				chatHistory = result.data.history;
				chatComplete = result.data.is_complete;
				if (result.data.is_complete) {
					applyRecommendations(result.data);
				}
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to send message';
		} finally {
			chatLoading = false;
			await tick();
			scrollChatToBottom();
		}
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
		step++;
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

			await goto(resolve('/dashboard'));
		} catch (e) {
			error = e instanceof Error ? e.message : 'Something went wrong';
		} finally {
			loading = false;
		}
	}
</script>

{#if initialLoading}
	<main class="mx-auto max-w-xl p-8">
		<p class="text-gray-500">Loading...</p>
	</main>
{:else}
	<main class="mx-auto max-w-xl p-8">
		<h1 class="mb-2 text-3xl font-bold">Set up your profile</h1>
		<p class="mb-6 text-sm text-gray-500">Step {step} of {totalSteps}</p>

		<div class="mb-8 h-2 w-full rounded-full bg-gray-200">
			<div
				class="h-2 rounded-full bg-black transition-all duration-300"
				style="width: {(step / totalSteps) * 100}%"
			></div>
		</div>

		{#if error}
			<div class="mb-4 rounded bg-red-100 p-3 text-sm text-red-700">{error}</div>
		{/if}

		{#if step === 1}
			<section>
				<h2 class="mb-1 text-lg font-semibold">What are your goals?</h2>
				<p class="mb-4 text-sm text-gray-500">Select at least one category you want to focus on.</p>
				<div class="flex flex-wrap gap-2">
					{#each categories as cat (cat.id)}
						<button
							type="button"
							onclick={() => toggleCategory(cat.id)}
							class="rounded-full border px-4 py-2 text-sm transition {selectedCategories.has(
								cat.id
							)
								? 'border-black bg-black text-white'
								: 'border-gray-300 bg-white text-gray-700 hover:border-gray-400'}"
						>
							{cat.label}
						</button>
					{/each}
				</div>
				<div class="mt-4">
					<label for="free-goals" class="mb-1 block text-sm font-medium">
						Additional goals (optional)
					</label>
					<textarea
						id="free-goals"
						bind:value={freeTextGoals}
						rows={3}
						placeholder="Describe any specific goals..."
						class="w-full rounded border px-3 py-2 text-sm"
					></textarea>
				</div>
			</section>
		{:else if step === 2}
			<section>
				<h2 class="mb-1 text-lg font-semibold">Your constraints</h2>
				<p class="mb-4 text-sm text-gray-500">Help us tailor your plan to your schedule.</p>
				<div class="mb-4">
					<label for="time" class="mb-1 block text-sm font-medium"> Daily time available </label>
					<select
						id="time"
						bind:value={timeAvailability}
						class="w-full rounded border px-3 py-2 text-sm"
					>
						<option value="30min">30 minutes</option>
						<option value="1hr">1 hour</option>
						<option value="2hr">2 hours</option>
						<option value="3hr+">3+ hours</option>
					</select>
				</div>
				<div>
					<label for="limitations" class="mb-1 block text-sm font-medium">
						Limitations (optional)
					</label>
					<textarea
						id="limitations"
						bind:value={limitations}
						rows={3}
						placeholder="Any physical, schedule, or other constraints..."
						class="w-full rounded border px-3 py-2 text-sm"
					></textarea>
				</div>
			</section>
		{:else if step === 3}
			<section>
				<h2 class="mb-1 text-lg font-semibold">Self-assessment</h2>
				<p class="mb-4 text-sm text-gray-500">Rate yourself on a scale of 1-10.</p>
				<div class="space-y-5">
					<div>
						<label for="motivation" class="mb-1 flex justify-between text-sm font-medium">
							<span>Motivation</span>
							<span class="text-gray-500">{motivation}</span>
						</label>
						<input
							id="motivation"
							type="range"
							min="1"
							max="10"
							bind:value={motivation}
							class="w-full"
						/>
					</div>
					<div>
						<label for="stress" class="mb-1 flex justify-between text-sm font-medium">
							<span>Stress level</span>
							<span class="text-gray-500">{stress}</span>
						</label>
						<input id="stress" type="range" min="1" max="10" bind:value={stress} class="w-full" />
					</div>
					<div>
						<label for="energy" class="mb-1 flex justify-between text-sm font-medium">
							<span>Energy</span>
							<span class="text-gray-500">{energy}</span>
						</label>
						<input id="energy" type="range" min="1" max="10" bind:value={energy} class="w-full" />
					</div>
				</div>
			</section>
		{:else if step === 4}
			<section>
				<h2 class="mb-1 text-lg font-semibold">Difficulty level</h2>
				<p class="mb-4 text-sm text-gray-500">How challenging should your daily plans be?</p>
				<div>
					<label for="difficulty" class="mb-1 flex justify-between text-sm font-medium">
						<span>Difficulty</span>
						<span class="text-gray-500">{difficulty}</span>
					</label>
					<input
						id="difficulty"
						type="range"
						min="1"
						max="10"
						bind:value={difficulty}
						class="w-full"
					/>
					<p class="mt-2 text-sm text-gray-600">{difficultyLabels[difficulty]}</p>
				</div>
			</section>
		{:else if step === 5}
			<section>
				<h2 class="mb-1 text-lg font-semibold">Let's get to know you</h2>
				<p class="mb-4 text-sm text-gray-500">
					A brief conversation to understand your situation better. This helps us personalize your
					experience.
				</p>

				<div
					{@attach setChatMessagesEl}
					class="mb-4 max-h-80 space-y-3 overflow-y-auto rounded-lg border bg-gray-50 p-4"
				>
					{#each chatHistory as msg, idx (idx)}
						<div class="flex {msg.role === 'user' ? 'justify-end' : 'justify-start'}">
							<div
								class="max-w-[80%] rounded-2xl px-4 py-2 text-sm {msg.role === 'user'
									? 'bg-black text-white'
									: 'bg-white text-gray-800 shadow-sm'}"
							>
								{msg.content}
							</div>
						</div>
					{/each}
					{#if chatLoading}
						<div class="flex justify-start">
							<div class="rounded-2xl bg-white px-4 py-2 text-sm text-gray-400 shadow-sm">
								Thinking...
							</div>
						</div>
					{/if}
				</div>

				{#if !chatComplete}
					<div class="flex gap-2">
						<input
							type="text"
							bind:value={chatInput}
							onkeydown={handleChatKeydown}
							placeholder="Type your response..."
							disabled={chatLoading || !chatStarted}
							class="flex-1 rounded border px-3 py-2 text-sm disabled:opacity-60"
						/>
						<button
							type="button"
							onclick={sendChatMessage}
							disabled={chatLoading || !chatInput.trim()}
							class="rounded bg-black px-4 py-2 text-sm text-white disabled:opacity-60"
						>
							Send
						</button>
					</div>
				{/if}
			</section>
		{:else if step === 6}
			<section>
				<h2 class="mb-1 text-lg font-semibold">How are you feeling?</h2>
				<p class="mb-4 text-sm text-gray-500">
					Select the emotional state that best describes where you're at right now. This helps us
					calibrate your plan.
				</p>
				<div class="space-y-3">
					{#each emotionalStates as state (state.id)}
						<button
							type="button"
							onclick={() => (selectedEmotionalState = state.id)}
							class="relative w-full rounded-lg border p-4 text-left transition {selectedEmotionalState ===
							state.id
								? 'border-black bg-gray-50'
								: 'border-gray-200 hover:border-gray-400'}"
						>
							{#if recommendedEmotionalState === state.id}
								<span
									class="absolute -top-2 right-3 rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-700"
								>
									Recommended for you
								</span>
							{/if}
							<p class="font-medium">{state.label}</p>
							<p class="mt-1 text-sm text-gray-500">{state.description}</p>
						</button>
					{/each}
					<button
						type="button"
						onclick={() => (selectedEmotionalState = '')}
						class="w-full rounded-lg border p-4 text-left transition {selectedEmotionalState === ''
							? 'border-black bg-gray-50'
							: 'border-gray-200 hover:border-gray-400'}"
					>
						<p class="font-medium">None of these</p>
						<p class="mt-1 text-sm text-gray-500">I'm doing okay, just looking to improve.</p>
					</button>
				</div>
			</section>
		{:else if step === 7}
			<section>
				<h2 class="mb-1 text-lg font-semibold">How should we deliver your plan?</h2>
				<p class="mb-4 text-sm text-gray-500">Choose how you'd like your daily tasks structured.</p>
				<div class="space-y-3">
					{#each deliveryModes as mode (mode.id)}
						<button
							type="button"
							onclick={() => (selectedDeliveryMode = mode.id)}
							class="relative w-full rounded-lg border p-4 text-left transition {selectedDeliveryMode ===
							mode.id
								? 'border-black bg-gray-50'
								: 'border-gray-200 hover:border-gray-400'}"
						>
							{#if recommendedDeliveryMode === mode.id}
								<span
									class="absolute -top-2 right-3 rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-700"
								>
									Recommended for you
								</span>
							{/if}
							<p class="font-medium">{mode.label}</p>
							<p class="mt-1 text-sm text-gray-500">{mode.description}</p>
						</button>
					{/each}
				</div>
			</section>
		{/if}

		<div class="mt-8 flex justify-between">
			{#if step > 1}
				<button
					type="button"
					onclick={back}
					disabled={loading}
					class="rounded border px-4 py-2 text-sm disabled:opacity-60"
				>
					Back
				</button>
			{:else}
				<div></div>
			{/if}

			{#if step === 5}
				<div class="flex gap-2">
					{#if !chatComplete}
						<button
							type="button"
							onclick={skipChat}
							class="rounded border px-4 py-2 text-sm text-gray-600 hover:border-gray-400"
						>
							Skip
						</button>
					{:else}
						<button
							type="button"
							onclick={next}
							class="rounded bg-black px-4 py-2 text-sm text-white"
						>
							Continue
						</button>
					{/if}
				</div>
			{:else if step < totalSteps}
				<button type="button" onclick={next} class="rounded bg-black px-4 py-2 text-sm text-white">
					Next
				</button>
			{:else}
				<button
					type="button"
					onclick={complete}
					disabled={loading}
					class="rounded bg-black px-4 py-2 text-sm text-white disabled:opacity-60"
				>
					{loading ? 'Setting up...' : 'Complete setup'}
				</button>
			{/if}
		</div>
	</main>
{/if}
