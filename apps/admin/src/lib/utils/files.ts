export const MAX_PRODUCT_IMAGES = 10;

/** Move an element within an array, returning a new array. */
export function moveItem<T>(items: readonly T[], from: number, to: number): T[] {
	const next = [...items];
	if (from < 0 || from >= next.length || to < 0 || to >= next.length || from === to) return next;
	const [item] = next.splice(from, 1);
	next.splice(to, 0, item as T);
	return next;
}
