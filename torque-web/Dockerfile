# Multi-stage Dockerfile for torque-web.
#
# Stage 1 builds the Vite bundle; Stage 2 serves it via a tiny nginx with
# the CSP nonce pipeline pre-wired. Non-root user, read-only root fs at
# runtime is set by the orchestrator (EasyPanel / compose).

# ---------- build ----------------------------------------------------
FROM node:20-alpine AS build
WORKDIR /app

# Leverage Docker layer cache on dependencies.
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund

COPY . .
RUN npm run build

# ---------- runtime --------------------------------------------------
FROM nginxinc/nginx-unprivileged:1.27-alpine AS runtime
USER 101
WORKDIR /usr/share/nginx/html

# Strip default nginx assets.
RUN rm -rf ./*

COPY --from=build /app/dist/ ./
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 8080

# S67 — HEALTHCHECK removed. nginx-unprivileged:1.27-alpine ships
# without wget OR curl, e adicionar uma das duas explode a superficie
# de ataque. EasyPanel/orchestrator poll /healthz externamente via
# HTTP GET — vide docker-compose.prod.yml comment.
