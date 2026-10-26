import { viVN } from '@clerk/localizations';
import { getLocale } from '$lib/paraglide/runtime.js';

// Admin palette (app.css): zinc neutrals, lavender brand. Borders, focus rings and control
// heights are set on Clerk's `.cl-*` classes in app.css, because Clerk's own styles win over
// `elements` for those properties.
const brand700 = '#5a4fb0';
const brand800 = '#463d8c';
const brand900 = '#342d68';

/** Clerk's components themed to match the admin (lavender brand, Inter, zinc borders). */
export const clerkAppearance = {
	variables: {
		colorPrimary: brand700,
		colorText: '#18181b',
		colorTextSecondary: '#52525b',
		colorInputBackground: '#ffffff',
		colorInputText: '#18181b',
		colorDanger: '#be123c',
		fontFamily: "'Inter Variable', Inter, ui-sans-serif, system-ui, sans-serif",
		borderRadius: '0.625rem',
		fontSize: '0.875rem'
	},
	elements: {
		// The auth pages provide their own heading and frame.
		rootBox: { width: '100%' },
		cardBox: { width: '100%', boxShadow: 'none', border: 'none' },
		card: { boxShadow: 'none', padding: '0', background: 'transparent', gap: '1.5rem' },
		header: { display: 'none' },
		footer: { background: 'transparent', paddingTop: '1.25rem' },
		// Sign-up is invitation-only, so never offer "Don't have an account? Sign up".
		footerAction: { display: 'none' },

		socialButtonsBlockButtonText: { fontWeight: 500, color: '#27272a' },
		dividerLine: { backgroundColor: '#e4e4e7' },
		dividerText: { color: '#71717a' },
		formFieldLabel: { fontWeight: 500, color: '#3f3f46' },
		formFieldAction: { color: brand700, fontWeight: 500 },
		identityPreviewEditButton: { color: brand700 },

		// The one action on the page: solid and high-contrast (white on brand-800 is > 8:1).
		formButtonPrimary: {
			fontSize: '0.875rem',
			fontWeight: 600,
			textTransform: 'none',
			backgroundColor: brand800,
			color: '#ffffff',
			'&:hover, &:active': { backgroundColor: brand900 }
		}
	}
};

/** Clerk UI strings for the admin's current language (English is Clerk's default). */
export function clerkLocalization() {
	return getLocale() === 'vi' ? viVN : undefined;
}
