<script lang="ts">
	import { m } from '$lib/paraglide/messages';
	import Seo from '$lib/components/Seo.svelte';
	import LegalPage from '$lib/components/LegalPage.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const sections = $derived([
		{ title: m.privacy_intro_title(), content: m.privacy_intro_content() },
		{
			title: m.privacy_collection_title(),
			content: m.privacy_collection_content(),
			items: [
				m.privacy_collection_personal(),
				m.privacy_collection_usage(),
				m.privacy_collection_cookies()
			]
		},
		{
			title: m.privacy_use_title(),
			content: m.privacy_use_content(),
			items: [m.privacy_use_service(), m.privacy_use_improvement(), m.privacy_use_communication()]
		},
		{ title: m.privacy_sharing_title(), content: m.privacy_sharing_content() },
		{ title: m.privacy_retention_title(), content: m.privacy_retention_content() },
		{ title: m.privacy_rights_title(), content: m.privacy_rights_content() },
		{ title: m.privacy_security_title(), content: m.privacy_security_content() },
		{ title: m.privacy_changes_title(), content: m.privacy_changes_content() }
	]);
</script>

<Seo title={m.privacy_title()} description={m.privacy_meta_description()} />

<LegalPage
	title={m.privacy_title()}
	{sections}
	contactTitle={m.privacy_contact_title()}
	contactContent={m.privacy_contact_content()}
	storeInfo={data.storeInfo}
/>
