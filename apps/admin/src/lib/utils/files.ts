export const IMAGE_TYPES = ['image/jpeg', 'image/png', 'image/webp'] as const;
export const MAX_IMAGE_BYTES = 10 * 1024 * 1024;
export const MAX_IMAGE_MB = 10;
export const MAX_PRODUCT_IMAGES = 10;

export type ImageRejection = 'type' | 'size' | 'count';

/** Client-side pre-checks; the API re-validates (and re-encodes) every upload. */
export function checkImageFile(file: { type: string; size: number }): ImageRejection | null {
	if (!(IMAGE_TYPES as readonly string[]).includes(file.type)) return 'type';
	if (file.size > MAX_IMAGE_BYTES) return 'size';
	return null;
}

/** Move an element within an array, returning a new array. */
export function moveItem<T>(items: readonly T[], from: number, to: number): T[] {
	const next = [...items];
	if (from < 0 || from >= next.length || to < 0 || to >= next.length || from === to) return next;
	const [item] = next.splice(from, 1);
	next.splice(to, 0, item as T);
	return next;
}
