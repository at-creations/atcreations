import { viVN } from '@clerk/localizations';
import { getLocale } from '$lib/paraglide/runtime.js';

/** Clerk's components themed to match the admin (brand blue, Inter, rounded inputs). */
export const clerkAppearance = {
	variables: {
		colorPrimary: '#6f63cc',
		colorText: '#0f172a',
		colorTextSecondary: '#64748b',
		colorInputBackground: '#ffffff',
		colorInputText: '#0f172a',
		fontFamily: "'Inter Variable', Inter, ui-sans-serif, system-ui, sans-serif",
		borderRadius: '0.5rem',
		fontSize: '0.875rem'
	},
	elements: {
		// The auth pages provide their own heading and frame.
		rootBox: { width: '100%' },
		cardBox: { width: '100%', boxShadow: 'none', border: 'none' },
		card: { boxShadow: 'none', padding: '0', background: 'transparent' },
		header: { display: 'none' },
		footer: { background: 'transparent', paddingTop: '1rem' },
		// Sign-up is invitation-only, so never offer "Don't have an account? Sign up".
		footerAction: { display: 'none' },
		// Pastel primary, same as the admin's own buttons.
		formButtonPrimary: {
			height: '2.5rem',
			fontSize: '0.875rem',
			textTransform: 'none',
			backgroundColor: '#dedaff',
			color: '#342d68',
			boxShadow: 'none',
			'&:hover, &:focus, &:active': { backgroundColor: '#c8c1fb', color: '#342d68' }
		},
		formFieldInput: { height: '2.5rem' },
		socialButtonsBlockButton: { height: '2.5rem' }
	}
};

/** Clerk UI strings for the admin's current language (English is Clerk's default). */
export function clerkLocalization() {
	return getLocale() === 'vi' ? viVN : undefined;
}
