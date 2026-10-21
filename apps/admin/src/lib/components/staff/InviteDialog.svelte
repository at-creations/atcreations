<script lang="ts">
	import type { Invitation, Role } from '@atcreations/api-types';
	import { api, fieldErrorMap, unwrap } from '$lib/api';
	import Field from '$lib/components/Field.svelte';
	import Icon from '$lib/components/Icon.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import { roleLabel } from '$lib/i18n';
	import { toastError } from '$lib/notify';
	import { m } from '$lib/paraglide/messages.js';
	import { toast } from '$lib/state/toast.svelte';
	import { ROLES } from '$lib/utils/permissions';
	import RoleLadder from './RoleLadder.svelte';

	let {
		open = $bindable(false),
		oninvited
	}: { open?: boolean; oninvited: (invitation: Invitation) => void } = $props();

	const uid = $props.id();
	let email = $state('');
	let role = $state<Role>('staff');
	let errors = $state<Record<string, string>>({});
	let saving = $state(false);

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
		saving = true;
		try {
			const res = await unwrap(
				api().POST('/v1/admin/staff/invitations', { body: { email: value, role } })
			);
			toast.success(m.staff_invite_sent({ email: value }));
			open = false;
			oninvited(res.data);
		} catch (error) {
			errors = fieldErrorMap(error);
			toastError(error, m.staff_invite_failed());
		} finally {
			saving = false;
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
				<select {...control} class="input" bind:value={role}>
					{#each ROLES as r (r)}
						<option value={r}>{roleLabel(r)}</option>
					{/each}
				</select>
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
