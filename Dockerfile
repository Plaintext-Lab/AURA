############################################################################
##### Stage 1: Build the backend application
############################################################################
FROM golang:alpine AS backend-builder

# Install required dependencies for cgo
RUN apk add --no-cache gcc musl-dev

# Set the working directory
WORKDIR /backend

# Copy the go.mod and go.sum files
COPY backend/go.mod backend/go.sum ./

# Download the dependencies
RUN go mod download

# Copy the source code
COPY backend/ ./

# Get the version from build arguments/environment variables
ARG APP_VERSION=dev

# Enable CGO and build the application 
ENV CGO_ENABLED=1
RUN go build -ldflags="-s -w -X main.APP_VERSION=$APP_VERSION" -o main .

############################################################################
##### Stage 2: Build the frontend application
############################################################################
FROM node:alpine AS frontend-builder

# Set the working directory
WORKDIR /frontend

# Copy the package.json and package-lock.json files
COPY frontend/package*.json ./

# Install the dependencies
RUN npm ci

# Copy the source code
COPY frontend/ ./

# Get the port number and version from build arguments/environment variables
ARG APP_VERSION=dev

# Set environment variables
ENV NEXT_PUBLIC_APP_VERSION=${APP_VERSION}
ENV NEXT_TELEMETRY_DISABLED=1

# Build the application
RUN npm run build || (echo "Build failed" && cat /frontend/.next/build-diagnostics.json 2>/dev/null || true && exit 1)

############################################################################
##### Stage 3: Build the final image
############################################################################
FROM node:alpine AS final

# Set the working directory
WORKDIR /app

# Install CA certificates and tzdata for timezone support
RUN apk update && apk add --no-cache ca-certificates tzdata

# Copy the backend application from the builder stage
COPY --from=backend-builder /backend/main .

# Copy the frontend build from the builder stage
COPY --from=frontend-builder /frontend/.next/standalone /app/
COPY --from=frontend-builder /frontend/.next/static /app/.next/static
COPY --from=frontend-builder /frontend/public /app/public

# Normalize runtime-readable permissions for static assets. Local git checkouts can
# carry restrictive modes (for example from a 077 umask), and COPY preserves them.
RUN find /app/public -type d -exec chmod 755 {} + \
	&& find /app/public -type f -exec chmod 644 {} + \
	&& find /app/.next/static -type d -exec chmod 755 {} + \
	&& find /app/.next/static -type f -exec chmod 644 {} +

# Get the version from build arguments/environment variables
ARG APP_VERSION=dev

# Set environment variables
ENV NODE_ENV=production
ENV HOME=/tmp
ENV XDG_CACHE_HOME=/tmp/.cache
ENV GOCACHE=/tmp/.cache/go-build
ENV GOMODCACHE=/tmp/.cache/go-mod

# Expose the port of the web interface. It forwards /api and /swagger to the backend
EXPOSE 3000

# Check that the backend is running and can read its database
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
	CMD wget -q -O /dev/null http://127.0.0.1:8888/api/health || exit 1

# Command to run both the backend and frontend
COPY --chmod=755 docker-entrypoint.sh /app/docker-entrypoint.sh
CMD ["/app/docker-entrypoint.sh"]
