export type PageItem = number | 'ellipsis-start' | 'ellipsis-end';

/**
 * Page items to render.
 * - 1 page or fewer: nothing (pagination hidden)
 * - up to 5 pages: every page
 * - otherwise: 1 … (current-1, current, current+1) … last
 */
export function getPageItems(current: number, totalPages: number): PageItem[] {
	if (!Number.isFinite(totalPages) || totalPages <= 1) return [];
	const total = Math.floor(totalPages);
	const cur = Math.min(Math.max(1, Math.floor(current) || 1), total);

	if (total <= 5) return Array.from({ length: total }, (_, i) => i + 1);

	const items: PageItem[] = [1];
	if (cur > 3) items.push('ellipsis-start');
	const start = Math.max(2, cur - 1);
	const end = Math.min(total - 1, cur + 1);
	for (let i = start; i <= end; i++) items.push(i);
	if (cur < total - 2) items.push('ellipsis-end');
	items.push(total);
	return items;
}
