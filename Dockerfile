# syntax=docker/dockerfile:1.7

# IAPro Nexus — Docker compatibility layer (CLI + web headless).
# No Wails/desktop. Provider auth/history live on host mounts (host-parity).

FROM oven/bun:1.3.9 AS web
WORKDIR /src
COPY web ./web
COPY internal/control/web/embedded/.keep ./internal/control/web/embedded/.keep
WORKDIR /src/web
RUN bun install --frozen-lockfile && bun run build

FROM golang:1.25.14-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/internal/control/web/embedded ./internal/control/web/embedded
ENV CGO_ENABLED=0
RUN go build -ldflags="-s -w" -o /out/nexus ./cmd/nexus

FROM debian:bookworm-slim AS runtime
RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates curl git jq ca-certificates \
  && rm -rf /var/lib/apt/lists/*

# Optional provider CLIs without credentials (bins only). Host PATH wins under host-parity.
# Failures are ignored so image builds succeed when a registry package rename occurs;
# host-parity remains the supported auth path.
ARG WITH_PROVIDER_CLIS=1
RUN if [ "$WITH_PROVIDER_CLIS" = "1" ]; then \
      apt-get update \
      && apt-get install -y --no-install-recommends nodejs npm \
      && npm install -g @anthropic-ai/claude-code 2>/dev/null || true \
      && npm install -g @google/gemini-cli 2>/dev/null || true \
      && npm install -g @openai/codex 2>/dev/null || true \
      && (curl -fsSL https://opencode.ai/install | bash) 2>/dev/null || true \
      && rm -rf /var/lib/apt/lists/*; \
    fi

COPY --from=build /out/nexus /usr/local/bin/nexus
COPY docker/entrypoint.sh /usr/local/bin/nexus-entrypoint
RUN chmod +x /usr/local/bin/nexus /usr/local/bin/nexus-entrypoint \
  && ln -sf /usr/local/bin/nexus /usr/local/bin/ai

ENV NEXUS_DOCKER=1 \
    NEXUS_COMPAT_DOCKER=1 \
    HOST_BIND_ADDRESS=127.0.0.1

USER 1000:1000
WORKDIR /workspace
EXPOSE 7420
ENTRYPOINT ["/usr/local/bin/nexus-entrypoint"]
CMD ["web", "--no-open", "--addr", "0.0.0.0:7420"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD nexus version || exit 1
