/** A stored image with optional smaller copies (the API's `Image`). */
interface ResponsiveImage {
	url: string;
	variants?: { url: string; width: number }[];
}

/**
 * URL of the smallest copy at least `minWidth` px wide (pass ~2× the displayed CSS width for
 * sharp results on high-DPI screens), else the full image. Undefined without an image.
 */
export function imageUrl(
	img: ResponsiveImage | null | undefined,
	minWidth: number
): string | undefined {
	if (!img) return undefined;
	return (img.variants ?? []).find((v) => v.width >= minWidth)?.url ?? img.url;
}
