<script lang="ts">
	import { m } from '$lib/paraglide/messages';
	import Seo from '$lib/components/Seo.svelte';
	import SectionTitle from '$lib/components/SectionTitle.svelte';
	import ContactDetails from '$lib/components/ContactDetails.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<Seo title={m.contact_title()} description={m.contact_meta_description()} />

<div class="container-page animate-slide-up pt-28 pb-20">
	<SectionTitle level={1} title={m.contact_title()} />
	<ContactDetails storeInfo={data.storeInfo} locale={data.locale} headingLevel={2} />
</div>
