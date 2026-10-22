import type { Invitation, PageMeta, Role, StaffMember } from '@atcreations/api-types';
import {
	createMutation,
	queryOptions,
	useQueryClient,
	type QueryClient
} from '@tanstack/svelte-query';
import { api, unwrap } from '$lib/api';
import { keys, type StaffListParams } from './keys';

type StaffPage = { data: StaffMember[]; meta: PageMeta };

export function staffListQuery(params: StaffListParams) {
	return queryOptions({
		queryKey: keys.staff.list(params),
		queryFn: ({ signal }) =>
			unwrap(api().GET('/v1/admin/staff', { params: { query: params }, signal }))
	});
}

/** Pending invitations only. */
export function invitationsQuery() {
	return queryOptions({
		queryKey: keys.staff.invitations,
		queryFn: async ({ signal }) =>
			(await unwrap(api().GET('/v1/admin/staff/invitations', { signal }))).data.filter(
				(i) => i.status === 'pending'
			)
	});
}

/** Patch a member in every cached staff page, then refetch the active ones. */
function putMember(queryClient: QueryClient, member: StaffMember) {
	queryClient.setQueriesData<StaffPage>({ queryKey: keys.staff.lists() }, (old) =>
		old ? { ...old, data: old.data.map((s) => (s.id === member.id ? member : s)) } : old
	);
	return queryClient.invalidateQueries({ queryKey: keys.staff.lists() });
}

export function createStaffRoleMutation() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: async ({ userId, role }: { userId: string; role: Role }) =>
			(
				await unwrap(
					api().PUT('/v1/admin/staff/{userId}/role', {
						params: { path: { userId } },
						body: { role }
					})
				)
			).data,
		onSuccess: (member) => putMember(queryClient, member)
	}));
}

export function createStaffBanMutation() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: async ({ userId, ban }: { userId: string; ban: boolean }) =>
			(ban
				? await unwrap(api().POST('/v1/admin/staff/{userId}/ban', { params: { path: { userId } } }))
				: await unwrap(
						api().POST('/v1/admin/staff/{userId}/unban', { params: { path: { userId } } })
					)
			).data,
		onSuccess: (member) => putMember(queryClient, member)
	}));
}

export function createDeleteStaffMutation() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: (userId: string) =>
			unwrap(api().DELETE('/v1/admin/staff/{userId}', { params: { path: { userId } } })),
		onSuccess: () => queryClient.invalidateQueries({ queryKey: keys.staff.lists() })
	}));
}

export function createInviteMutation() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: async (body: { email: string; role: Role }) =>
			(await unwrap(api().POST('/v1/admin/staff/invitations', { body }))).data,
		onSuccess: (invitation) => {
			queryClient.setQueryData<Invitation[]>(keys.staff.invitations, (old) =>
				old ? [invitation, ...old.filter((i) => i.id !== invitation.id)] : old
			);
			return queryClient.invalidateQueries({ queryKey: keys.staff.invitations });
		}
	}));
}

export function createRevokeInvitationMutation() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: (invitationId: string) =>
			unwrap(
				api().DELETE('/v1/admin/staff/invitations/{invitationId}', {
					params: { path: { invitationId } }
				})
			),
		onSuccess: (_data, invitationId) => {
			queryClient.setQueryData<Invitation[]>(keys.staff.invitations, (old) =>
				old?.filter((i) => i.id !== invitationId)
			);
			return queryClient.invalidateQueries({ queryKey: keys.staff.invitations });
		}
	}));
}
