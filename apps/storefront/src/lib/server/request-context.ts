import { AsyncLocalStorage } from 'node:async_hooks';

/** Per-request data needed by server-side API calls. */
interface RequestContext {
	clientAddress?: string;
}

const storage = new AsyncLocalStorage<RequestContext>();

export function runWithRequestContext<T>(ctx: RequestContext, fn: () => T): T {
	return storage.run(ctx, fn);
}

export function requestContext(): RequestContext | undefined {
	return storage.getStore();
}
