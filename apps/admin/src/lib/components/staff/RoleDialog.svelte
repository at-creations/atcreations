<script lang="ts">
	import type { Role, StaffMember } from '@atcreations/api-types';
	import { api, unwrap } from '$lib/api';
	import Field from '$lib/components/Field.svelte';
	import Icon from '$lib/components/Icon.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import StatusBadge from '$lib/components/StatusBadge.svelte';
	import { permissionLabel, roleLabel } from '$lib/i18n';
	import { toastError } from '$lib/notify';
	import { m } from '$lib/paraglide/messages.js';
	import { toast } from '$lib/state/toast.svelte';
	import { permissionDiff, ROLES } from '$lib/utils/permissions';
	import RoleLadder from './RoleLadder.svelte';

	let {
		open = $bindable(false),
		member,
		onupdated
	}: {
		open?: boolean;
		member: StaffMember | null;
		onupdated: (member: StaffMember) => void;
	} = $props();

	const uid = $props.id();
	let role = $state<Role>('staff');
	let saving = $state(false);

	export function reset() {
		role = member && member.role !== 'none' ? member.role : 'staff';
	}

	const diff = $derived(member ? permissionDiff(member.role, role) : { added: [], removed: [] });
	const unchanged = $derived(!member || member.role === role);

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (!member || saving || unchanged) return;
		saving = true;
		try {
			const res = await unwrap(
				api().PUT('/v1/admin/staff/{userId}/role', {
					params: { path: { userId: member.id } },
					body: { role }
				})
			);
			toast.success(m.staff_role_changed());
			open = false;
			onupdated(res.data);
		} catch (error) {
			// 409: self-modification / last admin — the problem detail is shown in the toast.
			toastError(error, m.staff_role_change_failed());
		} finally {
			saving = false;
		}
	}
</script>

<Modal bind:open busy={saving} size="lg" title={m.staff_change_role_title()}>
	{#if member}
		<form id="{uid}-form" class="space-y-4" onsubmit={submit}>
			<p class="text-sm text-slate-600">{m.staff_current_user({ email: member.email })}</p>
			{#if member.role === 'none'}
				<p class="rounded-lg bg-slate-100 p-3 text-sm text-slate-700">{m.staff_no_role_hint()}</p>
			{/if}
			<div class="flex flex-wrap items-end gap-4">
				<div>
					<p class="label">{m.staff_current_role()}</p>
					<StatusBadge kind="role" value={member.role} />
				</div>
				<Icon name="right" class="mb-2 text-slate-400" />
				<Field label={m.staff_new_role()} class="min-w-48 flex-1">
					{#snippet children(control)}
						<select {...control} class="input" bind:value={role}>
							{#each ROLES as r (r)}
								<option value={r}>{roleLabel(r)}</option>
							{/each}
						</select>
					{/snippet}
				</Field>
			</div>
			{#if role === 'admin' && member.role !== 'admin'}
				<p
					class="flex items-start gap-2 rounded-lg bg-amber-50 p-3 text-sm text-amber-900"
					role="alert"
				>
					<Icon name="alert" class="mt-0.5" />{m.role_admin_warning()}
				</p>
			{/if}
			{#if !unchanged}
				<div class="grid gap-3 sm:grid-cols-2" aria-live="polite">
					{#if diff.added.length}
						<div class="rounded-lg bg-emerald-50 p-3">
							<p class="text-xs font-semibold text-emerald-800 uppercase">
								{m.staff_permissions_gained()}
							</p>
							<ul class="mt-1 space-y-0.5 text-sm text-emerald-900">
								{#each diff.added as p (p)}<li>+ {permissionLabel(p)}</li>{/each}
							</ul>
						</div>
					{/if}
					{#if diff.removed.length}
						<div class="rounded-lg bg-red-50 p-3">
							<p class="text-xs font-semibold text-red-800 uppercase">
								{m.staff_permissions_lost()}
							</p>
							<ul class="mt-1 space-y-0.5 text-sm text-red-900">
								{#each diff.removed as p (p)}<li>− {permissionLabel(p)}</li>{/each}
							</ul>
						</div>
					{/if}
				</div>
			{/if}
			<RoleLadder highlight={role} />
		</form>
	{/if}
	{#snippet footer()}
		<button type="button" class="btn-secondary" disabled={saving} onclick={() => (open = false)}>
			{m.common_cancel()}
		</button>
		<button type="submit" form="{uid}-form" class="btn-primary" disabled={saving || unchanged}>
			{saving ? m.common_saving() : m.common_save_changes()}
		</button>
	{/snippet}
</Modal>
