# syntax=docker/dockerfile:1.10
# Build context: repository root (pnpm workspace).
ARG NODE_VERSION=22

FROM node:${NODE_VERSION}-alpine AS build
ENV PNPM_HOME=/pnpm PATH=/pnpm:$PATH CI=1 DO_NOT_TRACK=1
RUN corepack enable
WORKDIR /repo
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY packages ./packages
COPY apps/storefront ./apps/storefront
RUN --mount=type=cache,id=pnpm,target=/pnpm/store     pnpm install --frozen-lockfile --filter @atcreations/storefront...
# CSP origins are compiled into the app (kit.csp is static), so they are build args.
ARG PUBLIC_API_URL=
ARG PUBLIC_ASSET_BASE_URL=
ARG CSP_IMG_ORIGINS=
ENV PUBLIC_API_URL=${PUBLIC_API_URL} PUBLIC_ASSET_BASE_URL=${PUBLIC_ASSET_BASE_URL} CSP_IMG_ORIGINS=${CSP_IMG_ORIGINS}
RUN pnpm --filter @atcreations/storefront build  && pnpm --filter @atcreations/storefront deploy --prod --legacy /out

# ---- runtime: distroless Node, no shell, non-root ----
FROM gcr.io/distroless/nodejs${NODE_VERSION}-debian12:nonroot
WORKDIR /app
ENV NODE_ENV=production PORT=3000
COPY --from=build /out/package.json ./package.json
COPY --from=build /out/node_modules ./node_modules
COPY --from=build /repo/apps/storefront/build ./build
USER nonroot
EXPOSE 3000
CMD ["build/index.js"]
