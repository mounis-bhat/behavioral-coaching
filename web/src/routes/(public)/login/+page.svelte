<script lang="ts">
	import { goto } from '$app/navigation';
	import { authLoading, initAuth, user } from '$lib/stores/auth';

	let email = $state('');
	let password = $state('');
	let error = $state<string | null>(null);
	let loading = $state(false);

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
			await goto('/dashboard');
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
			goto('/dashboard');
		}
	});
</script>

<main class="mx-auto max-w-md p-8">
	<h1 class="mb-6 text-3xl font-bold">Sign in</h1>

	{#if error}
		<div class="mb-4 rounded bg-red-100 p-3 text-sm text-red-700">{error}</div>
	{/if}

	<form
		onsubmit={(e: Event) => {
			e.preventDefault();
			handleLogin();
		}}
		class="space-y-4"
	>
		<div>
			<label for="email" class="mb-1 block text-sm font-medium">Email</label>
			<input
				id="email"
				type="email"
				bind:value={email}
				required
				class="w-full rounded border px-3 py-2"
			/>
		</div>
		<div>
			<label for="password" class="mb-1 block text-sm font-medium">Password</label>
			<input
				id="password"
				type="password"
				bind:value={password}
				required
				class="w-full rounded border px-3 py-2"
			/>
		</div>
		<button
			type="submit"
			disabled={loading}
			class="w-full rounded bg-black px-4 py-2 text-white disabled:opacity-60"
		>
			{loading ? 'Signing in...' : 'Sign in'}
		</button>
	</form>

	<div class="my-6 text-center text-sm text-gray-500">or</div>
	<button
		onclick={handleGoogle}
		class="w-full rounded border px-4 py-2"
	>
		Continue with Google
	</button>

	<p class="mt-6 text-sm">
		No account?
		<a href="/register" class="ml-1 underline">Create one</a>
	</p>
</main>
