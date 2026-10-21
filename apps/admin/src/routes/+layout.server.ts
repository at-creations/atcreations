import { buildClerkProps } from 'svelte-clerk/server';
import type { LayoutServerLoad } from './$types';

// Only the Clerk auth state is serialised; all admin data is fetched client-side with a bearer token.
export const load: LayoutServerLoad = ({ locals }) => {
	return { ...buildClerkProps(locals.auth()) };
};
