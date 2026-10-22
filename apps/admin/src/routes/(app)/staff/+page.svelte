<script lang="ts">
	import type { Invitation, StaffMember } from '@atcreations/api-types';
	import { createQuery, keepPreviousData } from '@tanstack/svelte-query';
	import { tick } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import Icon from '$lib/components/Icon.svelte';
	import LoadState from '$lib/components/LoadState.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import RequirePermission from '$lib/components/RequirePermission.svelte';
	import InviteDialog from '$lib/components/staff/InviteDialog.svelte';
	import RoleDialog from '$lib/components/staff/RoleDialog.svelte';
	import StatusBadge from '$lib/components/StatusBadge.svelte';
	import { formatDate, formatDateTime } from '$lib/i18n';
	import { safeImageUrl } from '$lib/links';
	import { toastError } from '$lib/notify';
	import { m } from '$lib/paraglide/messages.js';
	import {
		createDeleteStaffMutation,
		createRevokeInvitationMutation,
		createStaffBanMutation,
		invitationsQuery,
		staffListQuery
	} from '$lib/queries';
	import { confirmAction } from '$lib/state/confirm.svelte';
	import { getAdminSession } from '$lib/state/session.svelte';
	import { toast } from '$lib/state/toast.svelte';
	import { readInt, readString, withParams } from '$lib/utils/query';

	const session = getAdminSession();
	const uid = $props.id();
	const PAGE_SIZES = [10, 25, 50] as const;

	const filters = $derived.by(() => {
		const p = page.url.searchParams;
		return {
			page: readInt(p, 'page', 1),
			pageSize: readInt(p, 'pageSize', 25, PAGE_SIZES),
			q: readString(p, 'q')
		};
	});

	const canManage = $derived(session.can('staff:manage'));
	const staffQuery = createQuery(() => ({
		...staffListQuery({
			page: filters.page,
			pageSize: filters.pageSize,
			q: filters.q || undefined
		}),
		placeholderData: keepPreviousData,
		enabled: canManage
	}));
	const invitationsList = createQuery(() => ({ ...invitationsQuery(), enabled: canManage }));
	const banMutation = createStaffBanMutation();
	const deleteMutation = createDeleteStaffMutation();
	const revokeMutation = createRevokeInvitationMutation();

	const staff = $derived(staffQuery.data?.data ?? []);
	const meta = $derived(staffQuery.data?.meta ?? null);
	const showError = $derived(staffQuery.isError && !staffQuery.data);
	let busyId = $state<string | null>(null);

	let inviteOpen = $state(false);
	let inviteDialog = $state<ReturnType<typeof InviteDialog>>();
	let roleOpen = $state(false);
	let roleMember = $state<StaffMember | null>(null);
	let roleDialog = $state<ReturnType<typeof RoleDialog>>();

	function navigate(changes: Record<string, string | number | null | undefined>) {
		// Same page, only the query string changes (built from page.url).
		// eslint-disable-next-line svelte/no-navigation-without-resolve
		goto(withParams(page.url, changes), { keepFocus: true, noScroll: true });
	}

	let searchInput = $derived(filters.q);
	let timer: ReturnType<typeof setTimeout> | undefined;
	function onSearch(value: string) {
		searchInput = value;
		clearTimeout(timer);
		timer = setTimeout(() => navigate({ q: value.trim() }), 300);
	}
	$effect(() => () => clearTimeout(timer));

	const isSelf = (member: StaffMember) => member.id === session.me?.id;

	async function openInvite() {
		inviteDialog?.reset();
		inviteOpen = true;
	}

	async function openRole(member: StaffMember) {
		roleMember = member;
		await tick();
		roleDialog?.reset();
		roleOpen = true;
	}

	async function toggleBan(member: StaffMember) {
		const banning = !member.banned;
		const ok = await confirmAction({
			title: banning ? m.staff_ban_title() : m.staff_unban_title(),
			message: banning
				? m.staff_ban_message({ email: member.email })
				: m.staff_unban_message({ email: member.email }),
			confirmLabel: banning ? m.staff_ban() : m.staff_unban(),
			danger: banning
		});
		if (!ok) return;
		busyId = member.id;
		try {
			await banMutation.mutateAsync({ userId: member.id, ban: banning });
			toast.success(
				banning
					? m.staff_banned_ok({ email: member.email })
					: m.staff_unbanned_ok({ email: member.email })
			);
		} catch (error) {
			toastError(error, m.staff_action_failed());
		} finally {
			busyId = null;
		}
	}

	async function remove(member: StaffMember) {
		const ok = await confirmAction({
			title: m.staff_delete_title(),
			message: m.staff_delete_message({ email: member.email }),
			warning: m.staff_delete_warning(),
			confirmLabel: m.common_delete(),
			danger: true,
			requireText: member.email,
			requireTextLabel: m.staff_delete_type({ email: member.email })
		});
		if (!ok) return;
		busyId = member.id;
		try {
			await deleteMutation.mutateAsync(member.id);
			toast.success(m.staff_deleted());
			if (staff.length === 1 && filters.page > 1) navigate({ page: filters.page - 1 });
		} catch (error) {
			toastError(error, m.staff_delete_failed());
		} finally {
			busyId = null;
		}
	}

	async function revoke(invitation: Invitation) {
		const ok = await confirmAction({
			title: m.staff_revoke_title(),
			message: m.staff_revoke_message({ email: invitation.email }),
			confirmLabel: m.staff_revoke(),
			danger: true
		});
		if (!ok) return;
		try {
			await revokeMutation.mutateAsync(invitation.id);
			toast.success(m.staff_revoked());
		} catch (error) {
			toastError(error, m.staff_action_failed());
		}
	}
</script>

{#snippet rowActions(member: StaffMember)}
	{@const self = isSelf(member)}
	{@const busy = busyId === member.id}
	<div class="flex flex-wrap justify-end gap-1" title={self ? m.staff_self_hint() : undefined}>
		<button
			type="button"
			class="btn-ghost btn-sm"
			disabled={self || busy}
			onclick={() => openRole(member)}
		>
			<Icon name="shield" size={14} />{m.staff_change_role()}
		</button>
		<button
			type="button"
			class="btn-ghost btn-sm"
			disabled={self || busy}
			onclick={() => toggleBan(member)}
		>
			<Icon name="ban" size={14} />{member.banned ? m.staff_unban() : m.staff_ban()}
		</button>
		<button
			type="button"
			class="btn-ghost btn-sm text-red-600 hover:bg-red-50"
			disabled={self || busy}
			onclick={() => remove(member)}
		>
			<Icon name="trash" size={14} />{m.common_delete()}
		</button>
	</div>
{/snippet}

{#snippet avatar(member: StaffMember)}
	{@const src = safeImageUrl(member.imageUrl)}
	{#if src}
		<img {src} alt="" class="size-9 rounded-full object-cover" loading="lazy" />
	{:else}
		<span
			class="flex size-9 items-center justify-center rounded-full bg-brand-100 text-sm font-semibold text-brand-800"
		>
			{(member.name || member.email).charAt(0).toUpperCase()}
		</span>
	{/if}
{/snippet}

<RequirePermission permission="staff:manage">
	<PageHeader title={m.staff_title()}>
		{#snippet actions()}
			<button type="button" class="btn-primary" onclick={openInvite}>
				<Icon name="plus" />{m.staff_invite()}
			</button>
		{/snippet}
	</PageHeader>

	<div class="mb-4 card p-4">
		<div class="relative max-w-md">
			<label for="{uid}-q" class="sr-only">{m.common_search()}</label>
			<Icon
				name="search"
				size={16}
				class="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-slate-400"
			/>
			<input
				id="{uid}-q"
				type="search"
				class="input pl-9"
				maxlength="100"
				placeholder={m.staff_search_placeholder()}
				value={searchInput}
				oninput={(e) => onSearch(e.currentTarget.value)}
			/>
		</div>
	</div>

	<section class="overflow-hidden card" aria-busy={staffQuery.isFetching}>
		{#if showError}
			<LoadState
				error={staffQuery.error}
				fallback={m.staff_load_failed()}
				onretry={() => staffQuery.refetch()}
			/>
		{:else if staffQuery.isPending}
			<LoadState loading />
		{:else if staff.length === 0}
			<p class="p-10 text-center text-slate-500">{m.staff_none()}</p>
		{:else}
			<table
				class={[
					'hidden w-full text-left text-sm lg:table',
					staffQuery.isPlaceholderData && 'opacity-60'
				]}
			>
				<thead class="bg-slate-50 text-xs font-semibold tracking-wide text-slate-500 uppercase">
					<tr>
						<th scope="col" class="px-4 py-3">{m.staff_col_name()}</th>
						<th scope="col" class="px-4 py-3">{m.staff_col_role()}</th>
						<th scope="col" class="px-4 py-3">{m.staff_col_status()}</th>
						<th scope="col" class="px-4 py-3">{m.staff_col_last_sign_in()}</th>
						<th scope="col" class="px-4 py-3 text-right">{m.common_actions()}</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-100">
					{#each staff as member (member.id)}
						<tr class={['hover:bg-slate-50', isSelf(member) && 'bg-brand-50/40']}>
							<td class="px-4 py-3">
								<div class="flex items-center gap-3">
									{@render avatar(member)}
									<div class="min-w-0">
										<p class="truncate font-medium">
											{member.name || '—'}
											{#if isSelf(member)}<span class="ml-1 badge bg-brand-100 text-brand-800"
													>{m.staff_you()}</span
												>{/if}
										</p>
										<p class="truncate text-xs text-slate-500">{member.email}</p>
									</div>
								</div>
							</td>
							<td class="px-4 py-3"><StatusBadge kind="role" value={member.role} /></td>
							<td class="px-4 py-3">
								{#if member.banned}
									<span class="badge bg-red-100 text-red-800">{m.staff_banned()}</span>
								{:else}
									<span class="badge bg-emerald-100 text-emerald-800">{m.staff_active()}</span>
								{/if}
							</td>
							<td class="px-4 py-3 text-slate-600">
								{member.lastSignInAt ? formatDateTime(member.lastSignInAt) : m.staff_never()}
							</td>
							<td class="px-4 py-3">{@render rowActions(member)}</td>
						</tr>
					{/each}
				</tbody>
			</table>
			<ul
				class={[
					'divide-y divide-slate-100 lg:hidden',
					staffQuery.isPlaceholderData && 'opacity-60'
				]}
			>
				{#each staff as member (member.id)}
					<li class="space-y-2 p-4">
						<div class="flex items-center gap-3">
							{@render avatar(member)}
							<div class="min-w-0 flex-1">
								<p class="truncate font-medium">{member.name || member.email}</p>
								<p class="truncate text-xs text-slate-500">{member.email}</p>
							</div>
							<StatusBadge kind="role" value={member.role} />
						</div>
						<p class="text-xs text-slate-500">
							{member.banned ? m.staff_banned() : m.staff_active()} · {m.staff_col_last_sign_in()}:
							{member.lastSignInAt ? formatDateTime(member.lastSignInAt) : m.staff_never()}
						</p>
						{@render rowActions(member)}
					</li>
				{/each}
			</ul>
		{/if}
		{#if meta && !showError}
			<Pagination
				{meta}
				pageSizes={[...PAGE_SIZES]}
				onpage={(p) => navigate({ page: p })}
				onpagesize={(size) => navigate({ pageSize: size })}
			/>
		{/if}
	</section>

	<section class="mt-6 card" aria-labelledby="{uid}-pending">
		<h2 id="{uid}-pending" class="border-b border-slate-200 px-5 py-3 font-semibold">
			{m.staff_pending_title()}
		</h2>
		{#if invitationsList.isPending}
			<LoadState loading />
		{:else if invitationsList.isError && !invitationsList.data}
			<LoadState
				error={invitationsList.error}
				fallback={m.staff_invitations_load_failed()}
				onretry={() => invitationsList.refetch()}
			/>
		{:else if !invitationsList.data || invitationsList.data.length === 0}
			<p class="p-5 text-sm text-slate-500">{m.staff_pending_none()}</p>
		{:else}
			<ul class="divide-y divide-slate-100">
				{#each invitationsList.data as invitation (invitation.id)}
					<li class="flex flex-wrap items-center gap-3 px-5 py-3">
						<Icon name="mail" class="text-slate-400" />
						<span class="min-w-0 flex-1 truncate text-sm font-medium">{invitation.email}</span>
						<StatusBadge kind="role" value={invitation.role} />
						<span class="text-xs text-slate-500"
							>{m.staff_invited_on({ date: formatDate(invitation.createdAt) })}</span
						>
						<button
							type="button"
							class="btn-ghost btn-sm text-red-600"
							onclick={() => revoke(invitation)}
						>
							{m.staff_revoke()}
						</button>
					</li>
				{/each}
			</ul>
		{/if}
	</section>

	<InviteDialog bind:this={inviteDialog} bind:open={inviteOpen} />
	<RoleDialog bind:this={roleDialog} bind:open={roleOpen} member={roleMember} />
</RequirePermission>
