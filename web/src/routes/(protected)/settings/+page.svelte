<script lang="ts">
	import type { components } from '$lib/api-types';
	import { user } from '$lib/stores/auth';
	import { get } from 'svelte/store';

	type AvatarUploadURLRequest = components['schemas']['api.AvatarUploadURLRequest'];
	type AvatarUploadURLResponse = components['schemas']['api.AvatarUploadURLResponse'];
	type AvatarConfirmRequest = components['schemas']['api.AvatarConfirmRequest'];
	type AvatarURLResponse = components['schemas']['api.AvatarURLResponse'];

	let currentPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');
	let error = $state<string | null>(null);
	let success = $state<string | null>(null);
	let loading = $state(false);
	let avatarError = $state<string | null>(null);
	let avatarSuccess = $state<string | null>(null);
	let avatarUploading = $state(false);
	let selectedFile = $state<File | null>(null);
	let previewUrl = $state<string | null>(null);

	$effect(() => {
		if (!selectedFile) {
			previewUrl = null;
			return;
		}
		const url = URL.createObjectURL(selectedFile);
		previewUrl = url;
		return () => {
			URL.revokeObjectURL(url);
		};
	});

	function handleFileChange(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		selectedFile = input.files?.[0] ?? null;
		avatarError = null;
		avatarSuccess = null;
	}

	async function handleAvatarUpload() {
		avatarError = null;
		avatarSuccess = null;
		const file = selectedFile;
		if (!file) {
			avatarError = 'Choose an image first.';
			return;
		}

		avatarUploading = true;
		try {
			const requestBody: AvatarUploadURLRequest = {
				content_type: file.type,
				size: file.size
			};

			const uploadRes = await fetch('/api/auth/avatar/upload-url', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(requestBody)
			});

			if (!uploadRes.ok) {
				const payload = (await uploadRes.json()) as { error?: string };
				throw new Error(payload.error ?? 'Unable to create upload URL');
			}

			const uploadData = (await uploadRes.json()) as AvatarUploadURLResponse;
			if (!uploadData?.url || !uploadData?.key) {
				throw new Error('Invalid upload URL response');
			}

			const headers = new Headers();
			if (uploadData.headers) {
				for (const [key, values] of Object.entries(uploadData.headers)) {
					for (const value of values) {
						headers.append(key, value);
					}
				}
			}
			if (!headers.has('content-type')) {
				headers.set('content-type', file.type);
			}

			const putRes = await fetch(uploadData.url, {
				method: uploadData.method ?? 'PUT',
				headers,
				body: file
			});
			if (!putRes.ok) {
				throw new Error('Upload failed');
			}

			const confirmBody: AvatarConfirmRequest = { key: uploadData.key };
			const confirmRes = await fetch('/api/auth/avatar/confirm', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(confirmBody)
			});

			if (!confirmRes.ok) {
				const payload = (await confirmRes.json()) as { error?: string };
				throw new Error(payload.error ?? 'Unable to confirm upload');
			}

			const confirmed = (await confirmRes.json()) as AvatarURLResponse;
			const current = get(user);
			if (current) {
				user.set({
					...current,
					picture: confirmed.url ?? current.picture ?? null
				});
			}

			selectedFile = null;
			avatarSuccess = 'Profile image updated.';
		} catch (e) {
			avatarError = e instanceof Error ? e.message : 'Unable to upload image';
		} finally {
			avatarUploading = false;
		}
	}

	async function handleChangePassword() {
		error = null;
		success = null;
		if (newPassword !== confirmPassword) {
			error = 'Passwords do not match';
			return;
		}

		loading = true;
		try {
			const res = await fetch('/api/auth/password', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					current_password: currentPassword,
					new_password: newPassword
				})
			});
			if (!res.ok) {
				const payload = (await res.json()) as { error?: string };
				throw new Error(payload.error ?? 'Unable to change password');
			}
			currentPassword = '';
			newPassword = '';
			confirmPassword = '';
			success = 'Password updated';
		} catch (e) {
			error = e instanceof Error ? e.message : 'Unable to change password';
		} finally {
			loading = false;
		}
	}
</script>

<main class="mx-auto max-w-md space-y-8 p-8">
	<h1 class="mb-6 text-3xl font-bold">Settings</h1>

	<section class="space-y-4">
		<h2 class="text-xl font-semibold">Profile image</h2>
		<div class="flex items-center gap-4">
			{#if previewUrl}
				<img src={previewUrl} alt="Avatar preview" class="h-16 w-16 rounded-full object-cover" />
		{:else if $user?.picture}
				<img src={$user.picture} alt={$user.name} class="h-16 w-16 rounded-full object-cover" />
			{:else}
				<div class="flex h-16 w-16 items-center justify-center rounded-full bg-gray-100 text-sm font-semibold text-gray-500">
					{$user?.name?.[0] ?? '?'}
				</div>
			{/if}
			<div>
				<p class="text-sm text-gray-600">JPG, PNG, or WebP up to 5MB.</p>
				<label class="mt-2 inline-flex cursor-pointer items-center gap-2 text-sm font-medium text-black">
					<input
						type="file"
						accept="image/jpeg,image/png,image/webp"
						class="hidden"
						onchange={handleFileChange}
					/>
					<span class="rounded border px-3 py-1.5">Choose image</span>
				</label>
				{#if selectedFile}
					<p class="mt-1 text-xs text-gray-500">{selectedFile.name}</p>
				{/if}
			</div>
		</div>
		{#if avatarError}
			<div class="rounded bg-red-100 p-3 text-sm text-red-700">{avatarError}</div>
		{/if}
		{#if avatarSuccess}
			<div class="rounded bg-green-100 p-3 text-sm text-green-700">{avatarSuccess}</div>
		{/if}
		<button
			type="button"
			disabled={avatarUploading || !selectedFile}
			onclick={handleAvatarUpload}
			class="rounded bg-black px-4 py-2 text-sm text-white disabled:opacity-60"
		>
			{avatarUploading ? 'Uploading...' : 'Upload image'}
		</button>
	</section>

	{#if error}
		<div class="mb-4 rounded bg-red-100 p-3 text-sm text-red-700">{error}</div>
	{/if}
	{#if success}
		<div class="mb-4 rounded bg-green-100 p-3 text-sm text-green-700">{success}</div>
	{/if}

	<form
		onsubmit={(e: Event) => {
			e.preventDefault();
			handleChangePassword();
		}}
		class="space-y-4"
	>
		<div>
			<label for="current" class="mb-1 block text-sm font-medium">Current password</label>
			<input
				id="current"
				type="password"
				bind:value={currentPassword}
				required
				class="w-full rounded border px-3 py-2"
			/>
		</div>
		<div>
			<label for="new" class="mb-1 block text-sm font-medium">New password</label>
			<input
				id="new"
				type="password"
				bind:value={newPassword}
				required
				class="w-full rounded border px-3 py-2"
			/>
			<p class="mt-2 text-xs text-gray-500">
				Min 8 chars, include uppercase, number, and special character.
			</p>
		</div>
		<div>
			<label for="confirm" class="mb-1 block text-sm font-medium">Confirm new password</label>
			<input
				id="confirm"
				type="password"
				bind:value={confirmPassword}
				required
				class="w-full rounded border px-3 py-2"
			/>
		</div>
		<button
			type="submit"
			disabled={loading}
			class="w-full rounded bg-black px-4 py-2 text-white disabled:opacity-60"
		>
			{loading ? 'Updating...' : 'Update password'}
		</button>
	</form>
</main>
