// The admin is a client-rendered app behind the server-side auth guard in hooks.server.ts.
// Pages fetch their data in the browser with the Clerk session token, so nothing is SSR'd.
export const ssr = false;
export const prerender = false;
