import { get } from 'svelte/store';
import { redirect } from '@sveltejs/kit';
import { authLoading, initAuth, user } from '$lib/stores/auth';

export async function load() {
	if (get(authLoading)) {
		await initAuth();
	}

	if (!get(user)) {
		throw redirect(307, '/login');
	}
}
