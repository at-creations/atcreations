export interface GalleryImage {
	url: string;
	alt: string;
	width?: number;
	height?: number;
	/** Shown under the photo (e.g. where it was taken). */
	caption?: string;
	/** Smaller copies for srcset (see $lib/images). */
	variants?: { url: string; width: number }[];
}
