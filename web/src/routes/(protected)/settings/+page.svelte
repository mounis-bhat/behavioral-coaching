<script lang="ts">
	import type { components } from '$lib/api-types';
	import { resolve } from '$app/paths';
	import { onMount } from 'svelte';
	import { fly } from 'svelte/transition';
	import { cubicOut } from 'svelte/easing';
	import { user } from '$lib/stores/auth';
	import { get } from 'svelte/store';

	type AvatarUploadURLRequest = components['schemas']['api.AvatarUploadURLRequest'];
	type AvatarUploadURLResponse = components['schemas']['api.AvatarUploadURLResponse'];
	type AvatarConfirmRequest = components['schemas']['api.AvatarConfirmRequest'];
	type AvatarURLResponse = components['schemas']['api.AvatarURLResponse'];

	let visible = $state(false);
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
	let verifyLoading = $state(false);
	let verifyError = $state<string | null>(null);
	let verifySuccess = $state<string | null>(null);

	onMount(() => {
		visible = true;
	});

	function updatePreviewUrl(file: File | null) {
		if (previewUrl) {
			URL.revokeObjectURL(previewUrl);
		}
		previewUrl = file ? URL.createObjectURL(file) : null;
	}

	function handleFileChange(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		const file = input.files?.[0] ?? null;
		selectedFile = file;
		updatePreviewUrl(file);
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
					picture: confirmed.url ?? current.picture ?? undefined
				});
			}

			selectedFile = null;
			updatePreviewUrl(null);
			avatarSuccess = 'Profile image updated.';
		} catch (e) {
			avatarError = e instanceof Error ? e.message : 'Unable to upload image';
		} finally {
			avatarUploading = false;
		}
	}

	async function handleResendVerification() {
		verifyError = null;
		verifySuccess = null;
		verifyLoading = true;
		try {
			const res = await fetch('/api/auth/verify-email/resend', { method: 'POST' });
			if (!res.ok) {
				const payload = (await res.json()) as { error?: string };
				throw new Error(payload.error ?? 'Unable to send verification email');
			}
			verifySuccess = 'Verification email sent. Check your inbox.';
		} catch (e) {
			verifyError = e instanceof Error ? e.message : 'Unable to send verification email';
		} finally {
			verifyLoading = false;
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

<svelte:head>
	<title>Settings — Behavioral Coaching</title>
</svelte:head>

<div class="settings-bg min-h-screen px-6 py-8">
	{#if visible}
		<div class="mx-auto max-w-lg">
			<!-- Header -->
			<div in:fly={{ y: -10, duration: 500, easing: cubicOut }} class="mb-8">
				<a
					href={resolve('/dashboard')}
					class="text-sm text-indigo-300/50 transition hover:text-indigo-200"
				>
					&larr; Dashboard
				</a>
				<h1 class="mt-3 text-2xl font-bold text-white">Settings</h1>
			</div>

			<!-- Profile Image -->
			<section
				in:fly={{ y: 20, duration: 600, delay: 100, easing: cubicOut }}
				class="mb-6 rounded-2xl border border-white/10 bg-white/5 p-6 backdrop-blur-sm"
			>
				<h2 class="mb-4 text-sm font-semibold text-white">Profile image</h2>
				<div class="flex items-center gap-4">
					{#if previewUrl}
						<img
							src={previewUrl}
							alt="Avatar preview"
							class="h-16 w-16 rounded-full object-cover ring-2 ring-white/10"
						/>
					{:else if $user?.picture}
						<img
							src={$user.picture}
							alt={$user.name}
							class="h-16 w-16 rounded-full object-cover ring-2 ring-white/10"
						/>
					{:else}
						<div
							class="flex h-16 w-16 items-center justify-center rounded-full bg-white/10 text-lg font-semibold text-indigo-200/70 ring-2 ring-white/10"
						>
							{$user?.name?.[0] ?? '?'}
						</div>
					{/if}
					<div>
						<p class="text-xs text-indigo-200/50">JPG, PNG, or WebP up to 5 MB</p>
						<label
							class="mt-2 inline-flex cursor-pointer items-center gap-2 text-sm font-medium text-indigo-200/70"
						>
							<input
								type="file"
								accept="image/jpeg,image/png,image/webp"
								class="hidden"
								onchange={handleFileChange}
							/>
							<span
								class="rounded-lg border border-white/10 bg-white/5 px-3 py-1.5 transition hover:bg-white/10 hover:text-white"
							>
								Choose image
							</span>
						</label>
						{#if selectedFile}
							<p class="mt-1 text-xs text-indigo-200/40">{selectedFile.name}</p>
						{/if}
					</div>
				</div>

				{#if avatarError}
					<div
						in:fly={{ y: -5, duration: 300 }}
						class="mt-4 rounded-lg bg-red-500/10 p-3 text-sm text-red-300 ring-1 ring-red-500/20"
					>
						{avatarError}
					</div>
				{/if}
				{#if avatarSuccess}
					<div
						in:fly={{ y: -5, duration: 300 }}
						class="mt-4 rounded-lg bg-green-500/10 p-3 text-sm text-green-300 ring-1 ring-green-500/20"
					>
						{avatarSuccess}
					</div>
				{/if}

				<button
					type="button"
					disabled={avatarUploading || !selectedFile}
					onclick={handleAvatarUpload}
					class="mt-4 rounded-lg bg-white px-4 py-2 text-sm font-semibold text-slate-900 transition hover:bg-indigo-100 disabled:opacity-50"
				>
					{avatarUploading ? 'Uploading...' : 'Upload image'}
				</button>
			</section>

			<!-- Account -->
			<section
				in:fly={{ y: 20, duration: 600, delay: 150, easing: cubicOut }}
				class="mb-6 rounded-2xl border border-white/10 bg-white/5 p-6 backdrop-blur-sm"
			>
				<h2 class="mb-4 text-sm font-semibold text-white">Account</h2>
				<div class="space-y-3">
					<div class="flex items-center justify-between">
						<div>
							<p class="text-xs font-medium text-indigo-200/50">Email</p>
							<p class="mt-0.5 text-sm text-white">{$user?.email ?? '—'}</p>
						</div>
						{#if $user?.email_verified}
							<span class="inline-flex items-center gap-1 rounded-full bg-emerald-400/10 px-2.5 py-1 text-xs font-medium text-emerald-300 ring-1 ring-emerald-400/20">
								<svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
									<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
								</svg>
								Verified
							</span>
						{:else}
							<span class="inline-flex items-center gap-1 rounded-full bg-amber-400/10 px-2.5 py-1 text-xs font-medium text-amber-300 ring-1 ring-amber-400/20">
								Not verified
							</span>
						{/if}
					</div>
					{#if $user?.provider}
						<div>
							<p class="text-xs font-medium text-indigo-200/50">Sign-in method</p>
							<p class="mt-0.5 text-sm capitalize text-white">{$user.provider}</p>
						</div>
					{/if}
				</div>

				{#if !$user?.email_verified}
					{#if verifyError}
						<div
							in:fly={{ y: -5, duration: 300 }}
							class="mt-4 rounded-lg bg-red-500/10 p-3 text-sm text-red-300 ring-1 ring-red-500/20"
						>
							{verifyError}
						</div>
					{/if}
					{#if verifySuccess}
						<div
							in:fly={{ y: -5, duration: 300 }}
							class="mt-4 rounded-lg bg-green-500/10 p-3 text-sm text-green-300 ring-1 ring-green-500/20"
						>
							{verifySuccess}
						</div>
					{/if}
					<button
						type="button"
						disabled={verifyLoading}
						onclick={handleResendVerification}
						class="mt-4 rounded-lg border border-white/10 bg-white/5 px-4 py-2 text-sm font-medium text-indigo-200/70 transition hover:bg-white/10 hover:text-white disabled:opacity-50"
					>
						{verifyLoading ? 'Sending...' : 'Send verification email'}
					</button>
				{/if}
			</section>

			<!-- Change Password -->
			{#if $user?.provider === 'credentials'}
			<section
				in:fly={{ y: 20, duration: 600, delay: 250, easing: cubicOut }}
				class="rounded-2xl border border-white/10 bg-white/5 p-6 backdrop-blur-sm"
			>
				<h2 class="mb-4 text-sm font-semibold text-white">Change password</h2>

				{#if error}
					<div
						in:fly={{ y: -5, duration: 300 }}
						class="mb-4 rounded-lg bg-red-500/10 p-3 text-sm text-red-300 ring-1 ring-red-500/20"
					>
						{error}
					</div>
				{/if}
				{#if success}
					<div
						in:fly={{ y: -5, duration: 300 }}
						class="mb-4 rounded-lg bg-green-500/10 p-3 text-sm text-green-300 ring-1 ring-green-500/20"
					>
						{success}
					</div>
				{/if}

				<form
					onsubmit={(e: Event) => {
						e.preventDefault();
						handleChangePassword();
					}}
					class="space-y-4"
				>
					<div>
						<label for="current" class="mb-1.5 block text-sm font-medium text-indigo-200/80">
							Current password
						</label>
						<input
							id="current"
							type="password"
							bind:value={currentPassword}
							required
							class="w-full rounded-lg border border-white/10 bg-white/5 px-3.5 py-2.5 text-sm text-white placeholder-indigo-300/30 outline-none transition focus:border-indigo-400/40 focus:ring-1 focus:ring-indigo-400/30"
							placeholder="••••••••"
						/>
					</div>
					<div>
						<label for="new" class="mb-1.5 block text-sm font-medium text-indigo-200/80">
							New password
						</label>
						<input
							id="new"
							type="password"
							bind:value={newPassword}
							required
							class="w-full rounded-lg border border-white/10 bg-white/5 px-3.5 py-2.5 text-sm text-white placeholder-indigo-300/30 outline-none transition focus:border-indigo-400/40 focus:ring-1 focus:ring-indigo-400/30"
							placeholder="••••••••"
						/>
						<p class="mt-1.5 text-xs text-indigo-200/40">
							Min 8 chars, include uppercase, number, and special character.
						</p>
					</div>
					<div>
						<label for="confirm" class="mb-1.5 block text-sm font-medium text-indigo-200/80">
							Confirm new password
						</label>
						<input
							id="confirm"
							type="password"
							bind:value={confirmPassword}
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
						{loading ? 'Updating...' : 'Update password'}
					</button>
				</form>
			</section>
			{/if}
		</div>
	{/if}
</div>

<style>
	.settings-bg {
		background: linear-gradient(
			135deg,
			#0f172a 0%,
			#1e1b4b 40%,
			#312e81 100%
		);
	}
</style>
