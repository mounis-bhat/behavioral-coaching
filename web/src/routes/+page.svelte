<script lang="ts">
	import { user } from '$lib/stores/auth';
</script>

<main class="mx-auto max-w-2xl p-8">
	<header class="mb-10">
		<p class="text-sm uppercase tracking-wide text-gray-500">Starter Template</p>
		<h1 class="mt-2 text-4xl font-bold">Build your AI app faster</h1>
		<p class="mt-3 text-gray-600">
			A clean, modular foundation with auth, API scaffolding, and an example AI flow.
		</p>
	</header>

	<div class="flex flex-wrap gap-3">
		{#if $user}
			<a href="/dashboard" class="rounded bg-black px-4 py-2 text-white">Go to dashboard</a>
			<a href="/settings" class="rounded border px-4 py-2">Account settings</a>
		{:else}
			<a href="/login" class="rounded border px-4 py-2">Sign in</a>
			<a href="/register" class="rounded bg-black px-4 py-2 text-white">Create account</a>
		{/if}
	</div>
</main>
