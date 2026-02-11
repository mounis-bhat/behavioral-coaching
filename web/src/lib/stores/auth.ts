import type { components } from '$lib/api-types';
import { writable } from 'svelte/store';

type AuthMeResponse = components['schemas']['api.AuthMeResponse'];
type AvatarURLResponse = components['schemas']['api.AvatarURLResponse'];

export type AuthUser = AuthMeResponse;

export const user = writable<AuthUser | null>(null);
export const authLoading = writable(true);

let initialized = false;

export async function initAuth(force = false) {
	if (initialized && !force) return;
	initialized = true;

	try {
		const res = await fetch('/api/auth/me');
		if (res.ok) {
			const data = (await res.json()) as AuthMeResponse;
			user.set(data);

			const avatarRes = await fetch('/api/auth/avatar-url');
			if (avatarRes.ok) {
				const avatar = (await avatarRes.json()) as AvatarURLResponse;
				user.update((current) => {
					if (!current) return current;
					return {
						...current,
						picture: avatar.url ?? current.picture
					};
				});
			}
		}
	} finally {
		authLoading.set(false);
	}
}
