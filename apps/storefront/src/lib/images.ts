/** An image with optional smaller copies (the API's `Image`, or a gallery image). */
export interface ResponsiveImage {
	url: string;
	width?: number;
	variants?: { url: string; width: number }[];
}

/**
 * `srcset` listing the image's smaller copies and the image itself, so the browser downloads
 * the size it displays. Undefined for images without copies (they load `src` as before).
 */
export function srcset(img: ResponsiveImage): string | undefined {
	const copies = img.variants ?? [];
	if (copies.length === 0 || !img.width) return undefined;
	return [...copies.map((v) => `${v.url} ${v.width}w`), `${img.url} ${img.width}w`].join(', ');
}

/** The smallest copy at least `minWidth` px wide (for fixed-size thumbnails), else the image. */
export function sized(img: ResponsiveImage, minWidth: number): string {
	return (img.variants ?? []).find((v) => v.width >= minWidth)?.url ?? img.url;
}

/** `sizes` for the product grids (1 / 2 / 3 / 4 columns). */
export const GRID_SIZES =
	'(min-width: 1280px) 25vw, (min-width: 1024px) 33vw, (min-width: 640px) 50vw, 100vw';
