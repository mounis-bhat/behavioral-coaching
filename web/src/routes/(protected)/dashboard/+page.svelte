<script lang="ts">
	import { goto } from '$app/navigation';
	import { user } from '$lib/stores/auth';

	let loading = $state(false);
	let error = $state<string | null>(null);

	async function handleLogout() {
		loading = true;
		error = null;
		try {
			const res = await fetch('/api/auth/logout', { method: 'POST' });
			if (!res.ok) throw new Error('Logout failed');
			user.set(null);
			await goto('/login');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Logout failed';
		} finally {
			loading = false;
		}
	}
</script>

<main class="mx-auto max-w-2xl p-8">
	<h1 class="mb-4 text-3xl font-bold">Dashboard</h1>

	{#if error}
		<div class="mb-4 rounded bg-red-100 p-3 text-sm text-red-700">{error}</div>
	{/if}

	{#if $user}
		<div class="rounded border p-4">
			<div class="flex items-center gap-4">
				{#if $user.picture}
					<img
						src={$user.picture}
						alt={$user.name}
						class="h-12 w-12 rounded-full"
					/>
				{/if}
				<div>
					<p class="text-sm text-gray-500">Signed in as</p>
					<p class="text-lg font-semibold">{$user.name}</p>
					<p class="text-sm text-gray-600">{$user.email}</p>
				</div>
			</div>
			<div class="mt-3 text-xs text-gray-500">
				<span>Provider: {$user.provider}</span>
				<span class="ml-3">Email verified: {$user.email_verified ? 'yes' : 'no'}</span>
			</div>
		</div>
	{/if}

	<section class="mt-8 rounded-lg bg-gray-50 p-6">
		<p class="text-gray-600">Behavioral coaching features coming soon.</p>
	</section>

	<button
		onclick={handleLogout}
		disabled={loading}
		class="mt-6 rounded bg-black px-4 py-2 text-white disabled:opacity-60"
	>
		{loading ? 'Signing out...' : 'Sign out'}
	</button>

	<p class="mt-4 text-sm">
		<a href="/settings" class="underline">Change password</a>
	</p>
</main>
