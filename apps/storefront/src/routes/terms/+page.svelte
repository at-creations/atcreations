<script lang="ts">
	import { m } from '$lib/paraglide/messages';
	import Seo from '$lib/components/Seo.svelte';
	import LegalPage from '$lib/components/LegalPage.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const sections = $derived([
		{ title: m.terms_intro_title(), content: m.terms_intro_content() },
		{ title: m.terms_use_title(), content: m.terms_use_content() },
		{ title: m.terms_account_title(), content: m.terms_account_content() },
		{ title: m.terms_ip_title(), content: m.terms_ip_content() },
		{ title: m.terms_user_content_title(), content: m.terms_user_content_content() },
		{ title: m.terms_purchases_title(), content: m.terms_purchases_content() },
		{ title: m.terms_returns_title(), content: m.terms_returns_content() },
		{ title: m.terms_disclaimers_title(), content: m.terms_disclaimers_content() },
		{ title: m.terms_limitation_title(), content: m.terms_limitation_content() },
		{ title: m.terms_termination_title(), content: m.terms_termination_content() },
		{ title: m.terms_law_title(), content: m.terms_law_content() },
		{ title: m.terms_changes_title(), content: m.terms_changes_content() }
	]);
</script>

<Seo title={m.terms_title()} description={m.terms_meta_description()} />

<LegalPage
	title={m.terms_title()}
	{sections}
	contactTitle={m.terms_contact_title()}
	contactContent={m.terms_contact_content()}
	storeInfo={data.storeInfo}
/>
