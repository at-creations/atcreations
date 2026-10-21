export const SLUG_PATTERN = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;

/** Turn arbitrary (including Vietnamese) text into a URL slug. */
export function slugify(input: string, maxLength = 120): string {
	const slug = input
		.normalize('NFD')
		.replace(/[̀-ͯ]/g, '')
		.replace(/[đĐ]/g, 'd')
		.toLowerCase()
		.replace(/&/g, ' and ')
		.replace(/[^a-z0-9]+/g, '-')
		.replace(/^-+|-+$/g, '');
	if (slug.length <= maxLength) return slug;
	return slug.slice(0, maxLength).replace(/-+$/g, '');
}

export function isValidSlug(slug: string, maxLength = 120): boolean {
	return slug.length > 0 && slug.length <= maxLength && SLUG_PATTERN.test(slug);
}
