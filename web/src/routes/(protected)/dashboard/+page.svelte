<script lang="ts">
	import { goto } from '$app/navigation';
	import { generateRecipe, type Recipe, type RecipeRequest } from '$lib/api';
	import { user } from '$lib/stores/auth';

	let loading = $state(false);
	let error = $state<string | null>(null);
	let recipeLoading = $state(false);
	let recipeError = $state<string | null>(null);
	let recipe = $state<Recipe | null>(null);
	let ingredient = $state('chicken');
	let dietaryRestrictions = $state('');

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

	async function handleGenerate() {
		recipeLoading = true;
		recipeError = null;
		recipe = null;

		const request: RecipeRequest = {
			ingredient,
			dietaryRestrictions: dietaryRestrictions || undefined
		};

		const result = await generateRecipe(request);

		if (result.error) {
			recipeError = result.error;
		} else {
			recipe = result.data;
		}

		recipeLoading = false;
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
		<h2 class="mb-4 text-xl font-semibold">Recipe Generator</h2>

		<form
			onsubmit={(e) => {
				e.preventDefault();
				handleGenerate();
			}}
			class="space-y-4"
		>
			<div>
				<label for="ingredient" class="mb-1 block text-sm font-medium">Main Ingredient</label>
				<input
					id="ingredient"
					type="text"
					bind:value={ingredient}
					class="w-full rounded border px-3 py-2"
					placeholder="e.g. chicken, tofu, salmon"
				/>
			</div>

			<div>
				<label for="dietary" class="mb-1 block text-sm font-medium">
					Dietary Restrictions (optional)
				</label>
				<input
					id="dietary"
					type="text"
					bind:value={dietaryRestrictions}
					class="w-full rounded border px-3 py-2"
					placeholder="e.g. gluten-free, vegan"
				/>
			</div>

			<button
				type="submit"
				disabled={recipeLoading || !ingredient}
				class="rounded bg-blue-600 px-4 py-2 text-white hover:bg-blue-700 disabled:opacity-50"
			>
				{recipeLoading ? 'Generating...' : 'Generate Recipe'}
			</button>
		</form>
	</section>

	{#if recipeError}
		<div class="mb-4 mt-4 rounded bg-red-100 p-4 text-red-700">
			{recipeError}
		</div>
	{/if}

	{#if recipe}
		<section class="mt-4 rounded-lg border p-6">
			<h2 class="mb-2 text-2xl font-bold">{recipe.title}</h2>
			<p class="mb-4 text-gray-600">{recipe.description}</p>

			<div class="mb-4 flex gap-4 text-sm text-gray-500">
				<span>Prep: {recipe.prepTime}</span>
				<span>Cook: {recipe.cookTime}</span>
				<span>Servings: {recipe.servings}</span>
			</div>

			<h3 class="mb-2 font-semibold">Ingredients</h3>
			<ul class="mb-4 list-inside list-disc">
				{#each recipe.ingredients ?? [] as item, i (i)}
					<li>{item}</li>
				{/each}
			</ul>

			<h3 class="mb-2 font-semibold">Instructions</h3>
			<ol class="mb-4 list-inside list-decimal">
				{#each recipe.instructions ?? [] as step, i (i)}
					<li>{step}</li>
				{/each}
			</ol>

			{#if recipe.tips?.length}
				<h3 class="mb-2 font-semibold">Tips</h3>
				<ul class="list-inside list-disc text-gray-600">
					{#each recipe.tips as tip, i (i)}
						<li>{tip}</li>
					{/each}
				</ul>
			{/if}
		</section>
	{/if}

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
