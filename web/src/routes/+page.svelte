<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import { fade, fly, scale } from 'svelte/transition';
	import { cubicOut, quintOut } from 'svelte/easing';
	import { user, authLoading } from '$lib/stores/auth';

	let heroVisible = $state(false);
	let sectionVisible = $state<boolean[]>(Array(8).fill(false));

	onMount(() => {
		heroVisible = true;
	});

	function reveal(index: number) {
		return (node: HTMLElement) => {
			const observer = new IntersectionObserver(
				([entry]) => {
					if (entry.isIntersecting) {
						sectionVisible[index] = true;
						observer.disconnect();
					}
				},
				{ threshold: 0.05 }
			);
			observer.observe(node);
			return () => observer.disconnect();
		};
	}

	const steps = [
		{
			number: 1,
			title: 'Tell us about yourself',
			description:
				'A 5-minute AI conversation that understands your goals, constraints, and psychological state.'
		},
		{
			number: 2,
			title: 'Get your daily plan',
			description:
				'Personalized tasks adapted to your energy, schedule, and preferred difficulty level.'
		},
		{
			number: 3,
			title: 'Build momentum',
			description:
				'Track progress, maintain streaks, and watch the system adapt as you grow stronger.'
		}
	];

	const categories = [
		{
			name: 'Health',
			description: 'Exercise, nutrition, sleep, and physical well-being habits.',
			accent: 'border-green-400/30 bg-green-400/10',
			iconColor: 'text-green-400',
			titleColor: 'text-green-300',
			descColor: 'text-indigo-200/70',
			icon: 'M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z'
		},
		{
			name: 'Mindset',
			description: 'Meditation, journaling, gratitude, and mental resilience.',
			accent: 'border-purple-400/30 bg-purple-400/10',
			iconColor: 'text-purple-400',
			titleColor: 'text-purple-300',
			descColor: 'text-indigo-200/70',
			icon: 'M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z'
		},
		{
			name: 'Discipline',
			description: 'Routines, focus blocks, consistency, and self-control.',
			accent: 'border-orange-400/30 bg-orange-400/10',
			iconColor: 'text-orange-400',
			titleColor: 'text-orange-300',
			descColor: 'text-indigo-200/70',
			icon: 'M13 10V3L4 14h7v7l9-11h-7z'
		},
		{
			name: 'Relationships',
			description: 'Communication, boundaries, social connection, and empathy.',
			accent: 'border-blue-400/30 bg-blue-400/10',
			iconColor: 'text-blue-400',
			titleColor: 'text-blue-300',
			descColor: 'text-indigo-200/70',
			icon: 'M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z'
		},
		{
			name: 'Productivity',
			description: 'Time management, deep work, goal setting, and efficiency.',
			accent: 'border-yellow-400/30 bg-yellow-400/10',
			iconColor: 'text-yellow-400',
			titleColor: 'text-yellow-300',
			descColor: 'text-indigo-200/70',
			icon: 'M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4'
		}
	];

	const deliveryModes = [
		{
			title: 'Strict Schedule',
			description: 'Tasks with specific time slots throughout your day.',
			icon: 'M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z'
		},
		{
			title: 'Flexible List',
			description: 'An ordered list to complete at your own pace.',
			icon: 'M4 6h16M4 10h16M4 14h16M4 18h16'
		},
		{
			title: 'Gentle Structure',
			description: '3 daily anchors plus tiny self-care tasks.',
			icon: 'M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z'
		}
	];

	const chatMessages = [
		{
			role: 'ai',
			content:
				"I'd love to understand what's been getting in the way of your goals. What's the biggest barrier you've faced when trying to build new habits?"
		},
		{
			role: 'user',
			content:
				"I start strong but lose motivation after a week or two. Then I feel guilty and it's even harder to restart."
		},
		{
			role: 'ai',
			content:
				"That guilt-avoidance cycle is really common, and it's not a willpower problem. Your brain is protecting you from failure. We'll design your plan with a built-in relapse protocol — so when you miss a day, the system adapts instead of punishing you."
		}
	];

	const stats = [
		{ label: 'Completion Rate', value: '87%', bar: 87, color: 'bg-green-400' },
		{ label: 'Current Streak', value: '12 days', bar: 60, color: 'bg-purple-400' },
		{ label: 'Difficulty', value: '7 / 10', bar: 70, color: 'bg-orange-400' },
		{ label: 'Tasks Completed', value: '342', bar: 85, color: 'bg-blue-400' }
	];

	const trustSignals = [
		'Adaptive difficulty based on adherence research',
		'Relapse recovery protocols inspired by clinical frameworks',
		'AI profiling that respects your psychological state'
	];
</script>

<svelte:head>
	<title>Behavioral Coaching — Build the habits that change your life</title>
	<meta name="description" content="A personal AI coach that understands your psychology, creates adaptive daily plans, and helps you build lasting behavioral change." />
</svelte:head>

<div class="page-bg overflow-x-hidden">
	<!-- Hero -->
	<section class="relative flex min-h-screen items-center justify-center overflow-hidden px-6">
		<div class="orb orb-green"></div>
		<div class="orb orb-purple"></div>
		<div class="orb orb-orange"></div>

		{#if heroVisible}
			<div class="relative z-10 mx-auto max-w-3xl text-center">
				<span
					in:fade={{ duration: 600, delay: 200 }}
					class="mb-6 inline-block rounded-full border border-white/15 bg-white/10 px-4 py-1.5 text-sm font-medium text-indigo-200 backdrop-blur-sm"
				>
					AI-Powered Behavioral Coaching
				</span>
				<h1
					in:fly={{ y: 30, duration: 800, delay: 400, easing: quintOut }}
					class="mt-4 text-5xl font-bold leading-tight tracking-tight text-white sm:text-6xl"
				>
					Build the habits that<br />change your life
				</h1>
				<p
					in:fly={{ y: 20, duration: 700, delay: 600, easing: cubicOut }}
					class="mx-auto mt-6 max-w-xl text-lg text-indigo-200/80"
				>
					A personal AI coach that understands your psychology, creates adaptive daily plans, and
					helps you build lasting behavioral change.
				</p>

				{#if !$authLoading}
					<div
						in:fly={{ y: 20, duration: 600, delay: 800, easing: cubicOut }}
						class="mt-8"
					>
						<a
							href={resolve($user ? '/dashboard' : '/register')}
							class="inline-block rounded-lg bg-white px-8 py-3.5 font-semibold text-slate-900 transition hover:bg-indigo-100"
						>
							Get Started
						</a>
					</div>
					{#if !$user}
						<p
							in:fade={{ duration: 500, delay: 1000 }}
							class="mt-4 text-sm text-indigo-300/50"
						>
							Free to use. No credit card required.
						</p>
					{/if}
				{/if}
			</div>
		{/if}
	</section>

	<!-- How It Works -->
	<section {@attach reveal(0)} class="border-t border-white/[0.06] bg-white/[0.03] px-6 py-24">
		<div class="mx-auto max-w-5xl">
			{#if sectionVisible[0]}
				<h2
					in:fly={{ y: 20, duration: 500, easing: cubicOut }}
					class="mb-4 text-center text-3xl font-bold text-white"
				>
					How it works
				</h2>
				<p
					in:fly={{ y: 20, duration: 500, delay: 100, easing: cubicOut }}
					class="mx-auto mb-12 max-w-xl text-center text-indigo-200/70"
				>
					Three steps to a personalized behavioral coaching experience.
				</p>
				<div class="grid gap-6 sm:grid-cols-3">
					{#each steps as step, i (step.number)}
						<div
							in:fly={{ y: 30, duration: 500, delay: 200 + i * 100, easing: cubicOut }}
							class="rounded-xl border border-white/10 bg-white/5 p-6 backdrop-blur-sm"
						>
							<div
								class="mb-4 flex h-10 w-10 items-center justify-center rounded-full bg-indigo-500/20 text-sm font-bold text-indigo-300 ring-1 ring-indigo-400/30"
							>
								{step.number}
							</div>
							<h3 class="mb-2 text-lg font-semibold text-white">{step.title}</h3>
							<p class="text-sm leading-relaxed text-indigo-200/70">{step.description}</p>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	</section>

	<!-- Goal Categories -->
	<section {@attach reveal(1)} class="border-t border-white/[0.06] px-6 py-24">
		<div class="mx-auto max-w-5xl">
			{#if sectionVisible[1]}
				<h2
					in:fly={{ y: 20, duration: 500, easing: cubicOut }}
					class="mb-4 text-center text-3xl font-bold text-white"
				>
					Five dimensions of growth
				</h2>
				<p
					in:fly={{ y: 20, duration: 500, delay: 100, easing: cubicOut }}
					class="mx-auto mb-12 max-w-xl text-center text-indigo-200/70"
				>
					Choose the areas that matter most to you. Your plan adapts to your priorities.
				</p>
				<div class="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-5">
					{#each categories as cat, i (cat.name)}
						<div
							in:scale={{
								start: 0.9,
								duration: 400,
								delay: 200 + i * 80,
								easing: cubicOut
							}}
							class="rounded-xl border {cat.accent} p-5"
						>
							<svg
								class="mb-3 h-6 w-6 {cat.iconColor}"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
								stroke-width="1.5"
							>
								<path stroke-linecap="round" stroke-linejoin="round" d={cat.icon} />
							</svg>
							<h3 class="font-semibold {cat.titleColor}">{cat.name}</h3>
							<p class="mt-1 text-sm {cat.descColor}">{cat.description}</p>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	</section>

	<!-- Key Features -->
	<section {@attach reveal(2)} class="border-t border-white/[0.06] bg-white/[0.03] px-6 py-24">
		<div class="mx-auto max-w-5xl">
			{#if sectionVisible[2]}
				<h2
					in:fly={{ y: 20, duration: 500, easing: cubicOut }}
					class="mb-12 text-center text-3xl font-bold text-white"
				>
					Built around how you actually work
				</h2>
				<div class="grid gap-10 md:grid-cols-2">
					<div in:fly={{ x: -30, duration: 600, delay: 200, easing: cubicOut }}>
						<h3 class="mb-4 text-xl font-semibold text-white">Delivery modes</h3>
						<p class="mb-6 text-sm text-indigo-200/70">
							Choose how your daily plan is structured. Switch anytime.
						</p>
						<div class="space-y-3">
							{#each deliveryModes as mode, i (mode.title)}
								<div
									in:fly={{ x: -20, duration: 400, delay: 400 + i * 100, easing: cubicOut }}
									class="flex gap-3 rounded-lg border border-white/10 bg-white/5 p-4 backdrop-blur-sm"
								>
									<svg
										class="mt-0.5 h-5 w-5 shrink-0 text-indigo-400"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
										stroke-width="1.5"
									>
										<path stroke-linecap="round" stroke-linejoin="round" d={mode.icon} />
									</svg>
									<div>
										<p class="text-sm font-medium text-white">{mode.title}</p>
										<p class="mt-0.5 text-sm text-indigo-200/60">{mode.description}</p>
									</div>
								</div>
							{/each}
						</div>
					</div>

					<div in:fly={{ x: 30, duration: 600, delay: 200, easing: cubicOut }}>
						<h3 class="mb-4 text-xl font-semibold text-white">Adaptive intelligence</h3>
						<p class="mb-6 text-sm text-indigo-200/70">
							The system learns from your behavior and adjusts automatically.
						</p>
						<div class="rounded-xl border border-white/10 bg-white/5 p-6 backdrop-blur-sm">
							<p class="mb-4 text-sm leading-relaxed text-indigo-200/70">
								Your plan difficulty adjusts based on your completion rate. Doing well? The system
								gradually increases the challenge. Struggling? It dials back to help you rebuild
								momentum.
							</p>
							<div class="mb-4">
								<div class="mb-1.5 flex items-baseline justify-between">
									<span class="text-xs text-indigo-300/60">Current difficulty</span>
									<span class="text-sm font-bold text-white">7 / 10</span>
								</div>
								<div class="h-2.5 w-full overflow-hidden rounded-full bg-white/10">
									<div
										class="difficulty-bar h-2.5 rounded-full bg-gradient-to-r from-orange-400 to-orange-500"
									></div>
								</div>
							</div>
							<p class="text-sm leading-relaxed text-indigo-200/70">
								If you fall off track, relapse recovery kicks in — lowering difficulty and rebuilding
								your routine gradually instead of starting from scratch.
							</p>
						</div>
					</div>
				</div>
			{/if}
		</div>
	</section>

	<!-- AI Conversation Preview -->
	<section {@attach reveal(3)} class="border-t border-white/[0.06] px-6 py-24">
		<div class="mx-auto max-w-3xl">
			{#if sectionVisible[3]}
				<h2
					in:fly={{ y: 20, duration: 500, easing: cubicOut }}
					class="mb-4 text-center text-3xl font-bold text-white"
				>
					Your AI coach gets it
				</h2>
				<p
					in:fly={{ y: 20, duration: 500, delay: 100, easing: cubicOut }}
					class="mx-auto mb-10 max-w-xl text-center text-indigo-200/70"
				>
					A thoughtful conversation that understands the real barriers to change.
				</p>
				<div
					in:scale={{ start: 0.95, duration: 500, delay: 200, easing: cubicOut }}
					class="space-y-3 rounded-2xl border border-white/10 bg-white/5 p-6 backdrop-blur-sm"
				>
					{#each chatMessages as msg, i (msg.content)}
						<div
							in:fly={{
								x: msg.role === 'user' ? 20 : -20,
								duration: 400,
								delay: 400 + i * 200,
								easing: cubicOut
							}}
							class="flex {msg.role === 'user' ? 'justify-end' : 'justify-start'}"
						>
							<div
								class="max-w-[80%] rounded-2xl px-4 py-3 text-sm leading-relaxed {msg.role ===
								'user'
									? 'bg-indigo-500/30 text-indigo-100 ring-1 ring-indigo-400/20'
									: 'bg-white/10 text-indigo-100 ring-1 ring-white/10'}"
							>
								{msg.content}
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	</section>

	<!-- Progress Preview -->
	<section {@attach reveal(4)} class="border-t border-white/[0.06] bg-white/[0.03] px-6 py-24">
		<div class="mx-auto max-w-3xl">
			{#if sectionVisible[4]}
				<h2
					in:fly={{ y: 20, duration: 500, easing: cubicOut }}
					class="mb-4 text-center text-3xl font-bold text-white"
				>
					Track real progress
				</h2>
				<p
					in:fly={{ y: 20, duration: 500, delay: 100, easing: cubicOut }}
					class="mx-auto mb-10 max-w-xl text-center text-indigo-200/70"
				>
					Clear metrics that show how your habits are compounding over time.
				</p>
				<div class="grid grid-cols-2 gap-4 sm:grid-cols-4">
					{#each stats as stat, i (stat.label)}
						<div
							in:fly={{ y: 20, duration: 400, delay: 200 + i * 100, easing: cubicOut }}
							class="rounded-xl border border-white/10 bg-white/5 p-4 backdrop-blur-sm"
						>
							<p class="text-xs text-indigo-300/60">{stat.label}</p>
							<p class="mt-1 text-2xl font-bold text-white">{stat.value}</p>
							<div class="mt-2 h-1.5 w-full overflow-hidden rounded-full bg-white/10">
								<div class="h-1.5 rounded-full {stat.color}" style="width: {stat.bar}%"></div>
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	</section>

	<!-- Trust Signals -->
	<section {@attach reveal(5)} class="border-t border-white/[0.06] px-6 py-24">
		<div class="mx-auto max-w-4xl">
			{#if sectionVisible[5]}
				<div class="grid gap-6 sm:grid-cols-3">
					{#each trustSignals as signal, i (signal)}
						<div
							in:fly={{ y: 20, duration: 400, delay: i * 100, easing: cubicOut }}
							class="flex gap-3"
						>
							<svg
								class="mt-0.5 h-5 w-5 shrink-0 text-emerald-400"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
								stroke-width="2"
							>
								<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
							</svg>
							<p class="text-sm leading-relaxed text-indigo-200/70">{signal}</p>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	</section>

	<!-- Final CTA -->
	<section {@attach reveal(6)} class="border-t border-white/[0.06] bg-white/[0.03] px-6 py-24">
		<div class="mx-auto max-w-2xl text-center">
			{#if sectionVisible[6]}
				<h2
					in:fly={{ y: 20, duration: 500, easing: cubicOut }}
					class="text-3xl font-bold text-white"
				>
					Start building better habits today
				</h2>
				<p
					in:fly={{ y: 20, duration: 500, delay: 100, easing: cubicOut }}
					class="mt-4 text-indigo-200/70"
				>
					Your personalized plan is a few minutes away.
				</p>

				{#if !$authLoading}
					<div
						in:fly={{ y: 20, duration: 500, delay: 200, easing: cubicOut }}
						class="mt-8"
					>
						<a
							href={resolve($user ? '/dashboard' : '/register')}
							class="inline-block rounded-lg bg-white px-8 py-3.5 font-semibold text-slate-900 transition hover:bg-indigo-100"
						>
							Get Started
						</a>
					</div>
				{/if}
			{/if}
		</div>
	</section>

	<!-- Footer -->
	<footer class="border-t border-white/10 px-6 py-8">
		<div class="mx-auto flex max-w-5xl flex-col items-center justify-between gap-4 sm:flex-row">
			<p class="text-sm text-indigo-300/40">&copy; {new Date().getFullYear()} Behavioral Coaching</p>
			{#if !$authLoading && !$user}
				<a href={resolve('/login')} class="text-sm text-indigo-300/40 transition hover:text-indigo-200">Sign in</a>
			{/if}
		</div>
	</footer>
</div>

<style>
	.page-bg {
		background: linear-gradient(
			180deg,
			#0f172a 0%,
			#1e1b4b 10%,
			#1e1b4b 30%,
			#312e81 50%,
			#3b0764 70%,
			#1e1b4b 85%,
			#0f172a 100%
		);
	}

	.orb {
		position: absolute;
		border-radius: 50%;
		filter: blur(120px);
		opacity: 0.2;
		pointer-events: none;
	}

	.orb-green {
		width: 500px;
		height: 500px;
		background: #22c55e;
		top: -150px;
		left: -150px;
		animation: float 8s ease-in-out infinite;
	}

	.orb-purple {
		width: 450px;
		height: 450px;
		background: #a855f7;
		top: 20%;
		right: -100px;
		animation: float 10s ease-in-out infinite reverse;
	}

	.orb-orange {
		width: 400px;
		height: 400px;
		background: #f97316;
		bottom: -100px;
		left: 25%;
		animation: float 12s ease-in-out infinite;
	}

	@keyframes float {
		0%,
		100% {
			transform: translateY(0) translateX(0);
		}
		33% {
			transform: translateY(-30px) translateX(15px);
		}
		66% {
			transform: translateY(15px) translateX(-20px);
		}
	}

	.difficulty-bar {
		width: 0%;
		animation: fill-bar 1.2s 0.8s cubic-bezier(0.25, 0.46, 0.45, 0.94) forwards;
	}

	@keyframes fill-bar {
		to {
			width: 70%;
		}
	}
</style>
