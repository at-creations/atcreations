<script lang="ts">
	import type { Role } from '@atcreations/api-types';
	import { fieldErrorMap } from '$lib/api';
	import Field from '$lib/components/Field.svelte';
	import Icon from '$lib/components/Icon.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import { roleLabel } from '$lib/i18n';
	import { toastError } from '$lib/notify';
	import { m } from '$lib/paraglide/messages.js';
	import { createInviteMutation } from '$lib/queries';
	import { toast } from '$lib/state/toast.svelte';
	import { ROLES } from '$lib/utils/permissions';
	import RoleLadder from './RoleLadder.svelte';

	let { open = $bindable(false) }: { open?: boolean } = $props();

	const uid = $props.id();
	let email = $state('');
	let role = $state<Role>('staff');
	let errors = $state<Record<string, string>>({});
	const invite = createInviteMutation();
	const saving = $derived(invite.isPending);

	export function reset() {
		email = '';
		role = 'staff';
		errors = {};
	}

	const EMAIL = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (saving) return;
		const value = email.trim();
		errors = EMAIL.test(value) ? {} : { email: m.validation_email() };
		if (errors.email) return;
		try {
			await invite.mutateAsync({ email: value, role });
			toast.success(m.staff_invite_sent({ email: value }));
			open = false;
		} catch (error) {
			errors = fieldErrorMap(error);
			toastError(error, m.staff_invite_failed());
		}
	}
</script>

<Modal
	bind:open
	busy={saving}
	size="lg"
	title={m.staff_invite_title()}
	description={m.staff_invite_body()}
>
	<form id="{uid}-form" class="space-y-4" onsubmit={submit} novalidate>
		<Field label={m.staff_invite_email()} required error={errors.email}>
			{#snippet children(control)}
				<input
					{...control}
					type="email"
					class="input"
					autocomplete="off"
					maxlength="254"
					bind:value={email}
				/>
			{/snippet}
		</Field>
		<Field label={m.staff_invite_role()} required error={errors.role}>
			{#snippet children(control)}
				<Select
					{control}
					value={role}
					items={ROLES.map((r) => ({ value: r, label: roleLabel(r) }))}
					onchange={(v) => (role = v as typeof role)}
				/>
			{/snippet}
		</Field>
		{#if role === 'admin'}
			<p
				class="flex items-start gap-2 rounded-lg bg-amber-50 p-3 text-sm text-amber-900"
				role="alert"
			>
				<Icon name="alert" class="mt-0.5" />{m.role_admin_warning()}
			</p>
		{/if}
		<RoleLadder highlight={role} />
	</form>
	{#snippet footer()}
		<button type="button" class="btn-secondary" disabled={saving} onclick={() => (open = false)}>
			{m.common_cancel()}
		</button>
		<button type="submit" form="{uid}-form" class="btn-primary" disabled={saving}>
			<Icon name="mail" size={16} />{saving ? m.common_processing() : m.staff_invite_send()}
		</button>
	{/snippet}
</Modal>
