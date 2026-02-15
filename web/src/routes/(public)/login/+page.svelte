<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { onMount } from 'svelte';
	import { fade, fly } from 'svelte/transition';
	import { cubicOut } from 'svelte/easing';
	import { authLoading, initAuth, user } from '$lib/stores/auth';

	let visible = $state(false);
	let email = $state('');
	let password = $state('');
	let error = $state<string | null>(null);
	let loading = $state(false);

	onMount(() => {
		visible = true;
	});

	async function handleLogin() {
		loading = true;
		error = null;
		try {
			const res = await fetch('/api/auth/login', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ email, password })
			});
			if (!res.ok) {
				const payload = (await res.json()) as { error?: string };
				throw new Error(payload.error ?? 'Login failed');
			}
			await initAuth(true);
			await goto(resolve('/dashboard'));
		} catch (e) {
			error = e instanceof Error ? e.message : 'Login failed';
		} finally {
			loading = false;
		}
	}

	function handleGoogle() {
		window.location.href = '/api/auth/google';
	}

	$effect(() => {
		if (!$authLoading && $user) {
			goto(resolve('/dashboard'));
		}
	});
</script>

<svelte:head>
	<title>Sign in — Behavioral Coaching</title>
</svelte:head>

<div class="auth-bg flex min-h-screen items-center justify-center px-6 py-12">
	{#if visible}
		<div class="w-full max-w-sm">
			<div in:fly={{ y: -10, duration: 500, easing: cubicOut }} class="mb-8 text-center">
				<a href={resolve('/')} class="text-sm text-indigo-300/50 transition hover:text-indigo-200">
					&larr; Back
				</a>
			</div>

			<div
				in:fly={{ y: 20, duration: 600, delay: 100, easing: cubicOut }}
				class="rounded-2xl border border-white/10 bg-white/5 p-8 backdrop-blur-sm"
			>
				<h1 class="mb-1 text-2xl font-bold text-white">Welcome back</h1>
				<p class="mb-6 text-sm text-indigo-200/60">Sign in to your account</p>

				{#if error}
					<div
						in:fly={{ y: -5, duration: 300 }}
						class="mb-4 rounded-lg bg-red-500/10 p-3 text-sm text-red-300 ring-1 ring-red-500/20"
					>
						{error}
					</div>
				{/if}

				<form
					onsubmit={(e: Event) => {
						e.preventDefault();
						handleLogin();
					}}
					class="space-y-4"
				>
					<div>
						<label for="email" class="mb-1.5 block text-sm font-medium text-indigo-200/80">Email</label>
						<input
							id="email"
							type="email"
							bind:value={email}
							required
							class="w-full rounded-lg border border-white/10 bg-white/5 px-3.5 py-2.5 text-sm text-white placeholder-indigo-300/30 outline-none transition focus:border-indigo-400/40 focus:ring-1 focus:ring-indigo-400/30"
							placeholder="you@example.com"
						/>
					</div>
					<div>
						<label for="password" class="mb-1.5 block text-sm font-medium text-indigo-200/80">Password</label>
						<input
							id="password"
							type="password"
							bind:value={password}
							required
							class="w-full rounded-lg border border-white/10 bg-white/5 px-3.5 py-2.5 text-sm text-white placeholder-indigo-300/30 outline-none transition focus:border-indigo-400/40 focus:ring-1 focus:ring-indigo-400/30"
							placeholder="••••••••"
						/>
					</div>
					<button
						type="submit"
						disabled={loading}
						class="w-full rounded-lg bg-white py-2.5 text-sm font-semibold text-slate-900 transition hover:bg-indigo-100 disabled:opacity-50"
					>
						{loading ? 'Signing in...' : 'Sign in'}
					</button>
				</form>

				<div class="my-5 flex items-center gap-3">
					<div class="h-px flex-1 bg-white/10"></div>
					<span class="text-xs text-indigo-300/40">or</span>
					<div class="h-px flex-1 bg-white/10"></div>
				</div>

				<button
					onclick={handleGoogle}
					class="flex w-full items-center justify-center gap-2.5 rounded-lg border border-white/10 bg-white/5 py-2.5 text-sm font-medium text-indigo-200 transition hover:bg-white/10"
				>
					<img src="/google.svg" alt="" class="h-4 w-4" />
					Continue with Google
				</button>
			</div>

			<p
				in:fade={{ duration: 400, delay: 300 }}
				class="mt-6 text-center text-sm text-indigo-300/50"
			>
				No account?
				<a href={resolve('/register')} class="ml-1 text-indigo-300 transition hover:text-white">Create one</a>
			</p>
		</div>
	{/if}
</div>

<style>
	.auth-bg {
		background: linear-gradient(
			135deg,
			#0f172a 0%,
			#1e1b4b 40%,
			#312e81 100%
		);
	}
</style>
