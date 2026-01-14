# ==================== Base ====================
FROM golang:1.23-alpine AS base
WORKDIR /app
RUN apk add --no-cache git ca-certificates tzdata
# Go 1.24+ 요구하는 의존성을 위해 toolchain 자동 다운로드 허용
ENV GOTOOLCHAIN=auto

# ==================== Development ====================
FROM base AS development
# Air 핫 리로드 설치
RUN go install github.com/air-verse/air@latest
# golang-migrate 설치
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
COPY go.mod go.sum ./
RUN go mod download
# 소스 코드는 볼륨 마운트로 제공
CMD ["air"]

# ==================== Builder ====================
FROM base AS builder
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/server

# ==================== Production ====================
FROM alpine:3.21 AS production
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/server .
COPY --from=builder /app/migrations ./migrations
EXPOSE 8080
# 보안: 비루트 사용자로 실행
USER nobody:nobody
CMD ["./server"]
