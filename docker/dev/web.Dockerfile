# syntax=docker/dockerfile:1.10
# Development image for the SvelteKit apps (vite dev server with HMR).
# Build context: repository root. Dependencies are installed inside the image
# (host node_modules are Windows/macOS-specific and never mounted); source
# changes are synced in by `docker compose watch`.
FROM node:22-alpine
ENV PNPM_HOME=/pnpm PATH=/pnpm:$PATH CI=1 DO_NOT_TRACK=1 NODE_ENV=development \
    COREPACK_HOME=/corepack
WORKDIR /repo
# Download pnpm at build time into a shared location so the unprivileged
# runtime user starts without fetching it from the network on every boot.
COPY package.json ./
RUN corepack enable && corepack install && chmod -R a+rX /corepack

ARG APP
ENV APP=${APP}
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY packages ./packages
COPY apps/${APP} ./apps/${APP}
RUN --mount=type=cache,id=pnpm,target=/pnpm/store \
    pnpm install --frozen-lockfile --filter "@atcreations/${APP}..."

# Unprivileged runtime user (the node image ships one with uid 1000).
RUN chown -R node:node /repo/apps/${APP}
USER node

ENV PORT=5173
EXPOSE 5173
CMD ["sh", "-c", "exec pnpm --filter \"@atcreations/${APP}\" exec vite dev --host 0.0.0.0 --port \"${PORT}\" --strictPort"]
