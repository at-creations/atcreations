// Types generated from apps/api/openapi.yaml. Regenerate with `pnpm gen:api`.
import type { components, paths } from './schema';

export type { components, paths };

type Schemas = components['schemas'];

export type LocalizedText = Schemas['LocalizedText'];
export type Image = Schemas['Image'];
export type Category = Schemas['Category'];
export type CategoryRef = Schemas['CategoryRef'];
export type CategoryInput = Schemas['CategoryInput'];
export type Product = Schemas['Product'];
export type ProductInput = Schemas['ProductInput'];
export type ProductStatus = Schemas['ProductStatus'];
export type StoreInfo = Schemas['StoreInfo'];
export type StoreInfoInput = Schemas['StoreInfoInput'];
export type BusinessHours = Schemas['BusinessHours'];
export type Order = Schemas['Order'];
export type OrderStatus = Schemas['OrderStatus'];
export type OrderEvent = Schemas['OrderEvent'];
export type Address = Schemas['Address'];
export type Me = Schemas['Me'];
export type Role = Schemas['Role'];
export type StaffMember = Schemas['StaffMember'];
export type Invitation = Schemas['Invitation'];
export type AuditEntry = Schemas['AuditEntry'];
export type AnalyticsReport = Schemas['AnalyticsReport'];
export type AnalyticsTotals = Schemas['AnalyticsTotals'];
export type AnalyticsDay = Schemas['AnalyticsDay'];
export type AnalyticsProduct = Schemas['AnalyticsProduct'];
export type AnalyticsCount = Schemas['AnalyticsCount'];
export type PageViewInput = Schemas['PageViewInput'];
export type PageMeta = Schemas['PageMeta'];
export type Problem = Schemas['Problem'];
export type Permission = Me['permissions'][number];

export interface Envelope<T> {
	data: T;
}
export interface PagedEnvelope<T> {
	data: T[];
	meta: PageMeta;
}
