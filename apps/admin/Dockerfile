# syntax=docker/dockerfile:1.10
# Build context: repository root (pnpm workspace).
# Base images are pinned by digest (Dependabot proposes updates); keep both on the same Node major.

FROM node:22-alpine@sha256:fe4f6d74e20370acb04ad889077d14d0f64a9c8ea6a42bc60cddf1e844699914 AS build
ENV PNPM_HOME=/pnpm PATH=/pnpm:$PATH CI=1 DO_NOT_TRACK=1
RUN corepack enable
WORKDIR /repo
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY packages ./packages
COPY apps/admin ./apps/admin
RUN --mount=type=cache,id=pnpm,target=/pnpm/store     pnpm install --frozen-lockfile --filter @atcreations/admin...
# CSP origins are compiled into the app (kit.csp is static), so they are build args.
ARG PUBLIC_API_URL=
ARG PUBLIC_CLERK_PUBLISHABLE_KEY=
ARG CSP_CLERK_ORIGINS=
ARG CSP_IMG_ORIGINS=
ENV PUBLIC_API_URL=${PUBLIC_API_URL} PUBLIC_CLERK_PUBLISHABLE_KEY=${PUBLIC_CLERK_PUBLISHABLE_KEY} CSP_CLERK_ORIGINS=${CSP_CLERK_ORIGINS} CSP_IMG_ORIGINS=${CSP_IMG_ORIGINS}
RUN pnpm --filter @atcreations/admin build  && pnpm --filter @atcreations/admin deploy --prod --legacy /out

# ---- runtime: distroless Node, no shell, non-root ----
FROM gcr.io/distroless/nodejs22-debian12:nonroot@sha256:13593b7570658e8477de39e2f4a1dd25db2f836d68a0ba771251572d23bb4f8e
WORKDIR /app
ENV NODE_ENV=production PORT=3000
COPY --from=build /out/package.json ./package.json
COPY --from=build /out/node_modules ./node_modules
COPY --from=build /repo/apps/admin/build ./build
USER nonroot
EXPOSE 3000
CMD ["build/index.js"]
