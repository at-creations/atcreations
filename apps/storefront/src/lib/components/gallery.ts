export interface GalleryImage {
	url: string;
	alt: string;
	width?: number;
	height?: number;
	/** Smaller copies for srcset (see $lib/images). */
	variants?: { url: string; width: number }[];
}
