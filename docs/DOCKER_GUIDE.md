# GoChat Docker 개발 및 배포 가이드

## 문서 정보

| 항목 | 내용 |
|-----|------|
| 프로젝트명 | GoChat |
| 버전 | 1.0 |
| 작성일 | 2025-01-09 |
| 상태 | Draft |

---

## 1. 개요

이 문서는 GoChat 프로젝트의 로컬 Docker 개발 환경 구성과 프로덕션 배포 전략을 설명합니다.

### 1.1 목표

- 로컬에서 Docker를 사용하여 모든 의존성(PostgreSQL, MongoDB, Redis)을 실행
- 프로덕션 환경(Railway + 클라우드 서비스)으로 쉽게 전환
- 환경별 설정을 명확히 분리하여 일관성 유지

### 1.2 환경 구분

| 환경 | 데이터베이스 | 용도 |
|-----|------------|------|
| 로컬 개발 | Docker 컨테이너 | 개발 및 테스트 |
| 스테이징 | 클라우드 서비스 (로컬 앱) | 클라우드 연동 테스트 |
| 프로덕션 | 클라우드 서비스 (Railway) | 실제 서비스 |

---

## 2. 프로젝트 구조

```
gochat/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   └── ...
├── migrations/
│   └── ...
├── docs/
│   ├── PRD.md
│   ├── TRD.md
│   └── DOCKER_GUIDE.md          # 본 문서
├── docker-compose.yml            # 기본 설정
├── docker-compose.override.yml   # 로컬 개발 오버라이드
├── Dockerfile                    # 멀티 스테이지 빌드
├── .dockerignore
├── .env.example                  # 환경변수 템플릿 (커밋)
├── .env                          # 로컬 환경변수 (커밋 X)
├── .env.production               # 프로덕션 환경변수 (커밋 X)
├── Makefile
├── go.mod
└── go.sum
```

---

## 3. Docker 설정 파일

### 3.1 Dockerfile (멀티 스테이지 빌드)

```dockerfile
# ==================== Base ====================
FROM golang:1.23-alpine AS base
WORKDIR /app
RUN apk add --no-cache git ca-certificates tzdata

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
```

### 3.2 docker-compose.yml (기본 설정)

```yaml
services:
  # ==================== Application ====================
  app:
    build:
      context: .
      dockerfile: Dockerfile
      target: production
    ports:
      - "${PORT:-8080}:8080"
    environment:
      - PORT=8080
      - GIN_MODE=${GIN_MODE:-release}
      - DATABASE_URL=${DATABASE_URL:-postgres://gochat:gochat@postgres:5432/gochat?sslmode=disable}
      - MONGODB_URI=${MONGODB_URI:-mongodb://mongodb:27017}
      - MONGODB_DATABASE=${MONGODB_DATABASE:-gochat}
      - REDIS_URL=${REDIS_URL:-redis://redis:6379}
      - JWT_SECRET=${JWT_SECRET}
      - JWT_ACCESS_EXPIRY=${JWT_ACCESS_EXPIRY:-15m}
      - JWT_REFRESH_EXPIRY=${JWT_REFRESH_EXPIRY:-168h}
    depends_on:
      postgres:
        condition: service_healthy
      mongodb:
        condition: service_started
      redis:
        condition: service_healthy
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s

  # ==================== PostgreSQL ====================
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: ${POSTGRES_USER:-gochat}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-gochat}
      POSTGRES_DB: ${POSTGRES_DB:-gochat}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "${POSTGRES_PORT:-5432}:5432"
    restart: unless-stopped
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-gochat} -d ${POSTGRES_DB:-gochat}"]
      interval: 5s
      timeout: 5s
      retries: 5

  # ==================== MongoDB ====================
  mongodb:
    image: mongo:7
    volumes:
      - mongodb_data:/data/db
    ports:
      - "${MONGODB_PORT:-27017}:27017"
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "mongosh", "--eval", "db.adminCommand('ping')"]
      interval: 10s
      timeout: 5s
      retries: 5

  # ==================== Redis ====================
  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes --maxmemory 256mb --maxmemory-policy allkeys-lru
    volumes:
      - redis_data:/data
    ports:
      - "${REDIS_PORT:-6379}:6379"
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5

volumes:
  postgres_data:
  mongodb_data:
  redis_data:
```

### 3.3 docker-compose.override.yml (로컬 개발용)

이 파일은 `docker compose up` 실행 시 자동으로 적용됩니다.

```yaml
services:
  # ==================== Application (개발 모드) ====================
  app:
    build:
      target: development
    volumes:
      - .:/app
      - go_mod_cache:/go/pkg/mod
      - go_build_cache:/root/.cache/go-build
      - /app/tmp  # Air 빌드 디렉토리 제외
    environment:
      - GIN_MODE=debug
      - DATABASE_URL=postgres://gochat:gochat@postgres:5432/gochat?sslmode=disable
      - MONGODB_URI=mongodb://mongodb:27017
      - MONGODB_DATABASE=gochat
      - REDIS_URL=redis://redis:6379
      - JWT_SECRET=dev-secret-key-do-not-use-in-production
    command: air
    # 개발 환경에서는 healthcheck 비활성화 (빠른 재시작)
    healthcheck:
      disable: true

  # ==================== 개발 도구: pgAdmin ====================
  pgadmin:
    image: dpage/pgadmin4:latest
    environment:
      PGADMIN_DEFAULT_EMAIL: admin@gochat.local
      PGADMIN_DEFAULT_PASSWORD: admin
      PGADMIN_CONFIG_SERVER_MODE: "False"
    volumes:
      - pgadmin_data:/var/lib/pgadmin
    ports:
      - "5050:80"
    depends_on:
      - postgres
    profiles:
      - tools

  # ==================== 개발 도구: Mongo Express ====================
  mongo-express:
    image: mongo-express:latest
    environment:
      ME_CONFIG_MONGODB_URL: mongodb://mongodb:27017/
      ME_CONFIG_BASICAUTH: "false"
    ports:
      - "8081:8081"
    depends_on:
      - mongodb
    profiles:
      - tools

  # ==================== 개발 도구: Redis Commander ====================
  redis-commander:
    image: rediscommander/redis-commander:latest
    environment:
      REDIS_HOSTS: local:redis:6379
    ports:
      - "8082:8081"
    depends_on:
      - redis
    profiles:
      - tools

volumes:
  go_mod_cache:
  go_build_cache:
  pgadmin_data:
```

### 3.4 .dockerignore

```
# Git
.git
.gitignore

# IDE
.idea
.vscode
*.swp
*.swo

# Build artifacts
bin/
tmp/
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test
*.test
coverage.out
coverage.html

# Environment files
.env
.env.*
!.env.example

# Documentation
docs/
*.md
!README.md

# Docker
docker-compose*.yml
Dockerfile*

# Misc
.DS_Store
Thumbs.db
```

---

## 4. 환경 변수 설정

### 4.1 .env.example (템플릿 - 커밋됨)

```env
# ===========================================
# GoChat 환경 변수 설정
# ===========================================

# Server
PORT=8080
ENV=development

# PostgreSQL
POSTGRES_USER=gochat
POSTGRES_PASSWORD=gochat
POSTGRES_DB=gochat
DATABASE_URL=postgres://gochat:gochat@postgres:5432/gochat?sslmode=disable

# MongoDB
MONGODB_URI=mongodb://mongodb:27017
MONGODB_DATABASE=gochat

# Redis
REDIS_URL=redis://redis:6379

# JWT
JWT_SECRET=your-256-bit-secret-key-here
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h

# CORS
ALLOWED_ORIGINS=http://localhost:3000
```

### 4.2 .env (로컬 개발 - 커밋 X)

```env
# 로컬 Docker 서비스용 설정
# docker compose up 으로 실행 시 사용

PORT=8080
ENV=development

POSTGRES_USER=gochat
POSTGRES_PASSWORD=gochat
POSTGRES_DB=gochat
DATABASE_URL=postgres://gochat:gochat@localhost:5432/gochat?sslmode=disable

MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=gochat

REDIS_URL=redis://localhost:6379

JWT_SECRET=local-dev-secret-key-32-characters!
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h

ALLOWED_ORIGINS=*
```

### 4.3 .env.production (프로덕션 - 커밋 X)

```env
# 프로덕션 클라우드 서비스 설정
# Railway 배포 시 대시보드에서 설정

PORT=8080
ENV=production

# Neon PostgreSQL
DATABASE_URL=postgres://user:password@ep-xxx.region.aws.neon.tech/gochat?sslmode=require

# MongoDB Atlas
MONGODB_URI=mongodb+srv://user:password@cluster.xxxxx.mongodb.net/gochat?retryWrites=true&w=majority

# Upstash Redis
REDIS_URL=rediss://default:xxxxx@xxx-xxx.upstash.io:6379

# JWT (강력한 시크릿 사용)
JWT_SECRET=your-production-256-bit-secret-key
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h

# CORS (프론트엔드 도메인)
ALLOWED_ORIGINS=https://your-frontend.com
```

---

## 5. Air 설정 (핫 리로드)

### 5.1 .air.toml

```toml
root = "."
tmp_dir = "tmp"

[build]
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ./cmd/server"
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "docs", "migrations"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html"]
  kill_delay = "2s"
  log = "build-errors.log"
  send_interrupt = false
  stop_on_error = true

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  time = false

[misc]
  clean_on_exit = true

[screen]
  clear_on_rebuild = true
```

---

## 6. Makefile

```makefile
.PHONY: help dev dev-d dev-tools db stop clean logs logs-all test test-coverage lint build migrate-up migrate-down migrate-create prod-build prod-test prod-size

# 기본 타겟
help:
	@echo "GoChat Docker 명령어"
	@echo ""
	@echo "개발 환경:"
	@echo "  make dev          - 개발 서버 시작 (포그라운드)"
	@echo "  make dev-d        - 개발 서버 시작 (백그라운드)"
	@echo "  make dev-tools    - 개발 서버 + DB 관리 도구 시작"
	@echo "  make db           - 데이터베이스만 시작"
	@echo "  make stop         - 모든 컨테이너 중지"
	@echo "  make clean        - 컨테이너 및 볼륨 삭제"
	@echo "  make logs         - 앱 로그 확인"
	@echo ""
	@echo "테스트 & 빌드:"
	@echo "  make test         - 테스트 실행"
	@echo "  make lint         - 린터 실행"
	@echo "  make build        - 로컬 빌드"
	@echo ""
	@echo "마이그레이션:"
	@echo "  make migrate-up   - 마이그레이션 적용"
	@echo "  make migrate-down - 마이그레이션 롤백"
	@echo "  make migrate-create name=xxx - 새 마이그레이션 생성"
	@echo ""
	@echo "프로덕션:"
	@echo "  make prod-build   - 프로덕션 이미지 빌드"
	@echo "  make prod-test    - 프로덕션 환경 로컬 테스트"

# ==================== 개발 환경 ====================

# 개발 서버 시작 (포그라운드)
dev:
	docker compose up

# 개발 서버 시작 (백그라운드)
dev-d:
	docker compose up -d

# 개발 서버 + DB 관리 도구 (pgAdmin, mongo-express, redis-commander)
dev-tools:
	docker compose --profile tools up

# 데이터베이스만 시작 (앱은 로컬에서 실행)
db:
	docker compose up postgres mongodb redis -d

# 컨테이너 중지
stop:
	docker compose down

# 컨테이너 및 볼륨 삭제
clean:
	docker compose down -v
	rm -rf tmp/

# 앱 로그 확인
logs:
	docker compose logs -f app

# 모든 로그 확인
logs-all:
	docker compose logs -f

# ==================== 테스트 & 빌드 ====================

# 테스트 실행
test:
	go test -v ./...

# 테스트 (커버리지)
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# 린터 실행
lint:
	golangci-lint run

# 로컬 빌드
build:
	go build -o bin/server ./cmd/server

# ==================== 마이그레이션 ====================

# 마이그레이션 적용
migrate-up:
	migrate -path ./migrations -database "$${DATABASE_URL}" up

# 마이그레이션 롤백
migrate-down:
	migrate -path ./migrations -database "$${DATABASE_URL}" down 1

# 새 마이그레이션 생성
migrate-create:
	@if [ -z "$(name)" ]; then echo "Usage: make migrate-create name=migration_name"; exit 1; fi
	migrate create -ext sql -dir ./migrations -seq $(name)

# ==================== 프로덕션 ====================

# 프로덕션 이미지 빌드
prod-build:
	docker build --target production -t gochat:latest .

# 프로덕션 환경 로컬 테스트 (클라우드 DB 사용)
prod-test:
	docker compose -f docker-compose.yml --env-file .env.production up --build

# 프로덕션 이미지 크기 확인
prod-size:
	docker images gochat:latest --format "{{.Size}}"
```

---

## 7. 실행 가이드

### 7.1 최초 설정

```bash
# 1. 환경 변수 파일 생성
cp .env.example .env

# 2. 개발 서버 시작
make dev

# 3. 서비스 확인
# - 앱: http://localhost:8080
# - 앱 헬스체크: http://localhost:8080/health
```

### 7.2 개발 환경 실행

```bash
# 기본 개발 서버 (앱 + DB)
make dev

# 백그라운드 실행
make dev-d

# DB 관리 도구 포함 실행
make dev-tools
# - pgAdmin: http://localhost:5050
# - Mongo Express: http://localhost:8081
# - Redis Commander: http://localhost:8082

# 로그 확인
make logs
```

### 7.3 데이터베이스만 실행

앱은 IDE에서 직접 실행하고 싶을 때:

```bash
# DB만 Docker로 실행
make db

# 앱은 로컬에서 실행
go run ./cmd/server
```

### 7.4 마이그레이션

```bash
# 마이그레이션 적용
make migrate-up

# 마이그레이션 롤백 (1단계)
make migrate-down

# 새 마이그레이션 생성
make migrate-create name=add_users_table
```

### 7.5 테스트

```bash
# 테스트 실행
make test

# 커버리지 리포트
make test-coverage
```

---

## 8. 프로덕션 배포

### 8.1 클라우드 서비스 설정

#### Neon PostgreSQL
1. https://neon.tech 가입
2. 새 프로젝트 생성
3. Connection String 복사 → `DATABASE_URL`

#### MongoDB Atlas
1. https://mongodb.com/atlas 가입
2. 무료 클러스터 생성 (M0)
3. Database Access에서 사용자 생성
4. Network Access에서 IP 허용 (0.0.0.0/0)
5. Connection String 복사 → `MONGODB_URI`

#### Upstash Redis
1. https://upstash.com 가입
2. 새 Redis 데이터베이스 생성
3. REST URL 또는 Redis URL 복사 → `REDIS_URL`

### 8.2 로컬에서 클라우드 연동 테스트

```bash
# .env.production 파일에 클라우드 URL 설정 후
make prod-test
```

### 8.3 Railway 배포

#### 방법 1: GitHub 연동 (권장)

1. GitHub에 코드 푸시
2. Railway 대시보드에서 "New Project" → "Deploy from GitHub"
3. 저장소 선택
4. 환경 변수 설정:
   ```
   DATABASE_URL=<Neon URL>
   MONGODB_URI=<Atlas URL>
   REDIS_URL=<Upstash URL>
   JWT_SECRET=<강력한 시크릿>
   ENV=production
   ALLOWED_ORIGINS=<프론트엔드 URL>
   ```
5. Deploy

#### 방법 2: Railway CLI

```bash
# Railway CLI 설치
npm install -g @railway/cli

# 로그인
railway login

# 프로젝트 초기화
railway init

# 환경 변수 설정
railway variables set DATABASE_URL="<Neon URL>"
railway variables set MONGODB_URI="<Atlas URL>"
railway variables set REDIS_URL="<Upstash URL>"
railway variables set JWT_SECRET="<시크릿>"
railway variables set ENV="production"

# 배포
railway up
```

#### 방법 3: Docker Compose Import

1. Railway 대시보드에서 프로젝트 생성
2. `docker-compose.yml` 파일을 Canvas에 드래그 앤 드롭
3. 자동으로 서비스 구성됨
4. 환경 변수 수정

### 8.4 Railway 배포 시 주의사항

```go
// main.go에서 PORT 환경 변수 처리
func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    router := gin.Default()
    // ... 라우터 설정

    router.Run(":" + port)
}
```

---

## 9. 환경별 전환 체크리스트

### 로컬 개발 → 프로덕션

| 항목 | 로컬 개발 | 프로덕션 |
|-----|----------|---------|
| PostgreSQL | `postgres:5432` | Neon (`sslmode=require`) |
| MongoDB | `mongodb:27017` | Atlas (`mongodb+srv://`) |
| Redis | `redis:6379` | Upstash (`rediss://`) |
| JWT Secret | 개발용 간단한 값 | 256비트 강력한 시크릿 |
| CORS | `*` 허용 | 특정 도메인만 |
| ENV | `development` | `production` |
| 로깅 | 상세 (debug) | 간략 (info) |
| SSL | 불필요 | 필수 (자동) |

### 연결 문자열 차이

```bash
# 로컬 PostgreSQL
postgres://gochat:gochat@localhost:5432/gochat?sslmode=disable

# Neon PostgreSQL
postgres://user:pass@ep-xxx.region.aws.neon.tech/gochat?sslmode=require

# 로컬 MongoDB (인증 없이 사용)
mongodb://localhost:27017

# MongoDB Atlas
mongodb+srv://user:pass@cluster.xxxxx.mongodb.net/gochat?retryWrites=true&w=majority

# 로컬 Redis
redis://localhost:6379

# Upstash Redis
rediss://default:xxxxx@xxx-xxx.upstash.io:6379
```

---

## 10. 트러블슈팅

### 10.1 컨테이너가 시작되지 않음

```bash
# 로그 확인
docker compose logs app

# 컨테이너 상태 확인
docker compose ps

# 이미지 재빌드
docker compose build --no-cache
```

### 10.2 데이터베이스 연결 실패

```bash
# PostgreSQL 연결 테스트
docker compose exec postgres psql -U gochat -d gochat -c "SELECT 1"

# MongoDB 연결 테스트
docker compose exec mongodb mongosh

# Redis 연결 테스트
docker compose exec redis redis-cli ping
```

### 10.3 볼륨 데이터 초기화

```bash
# 모든 데이터 삭제 (주의!)
make clean

# 특정 볼륨만 삭제
docker volume rm gochat_postgres_data
```

### 10.4 포트 충돌

```bash
# 사용 중인 포트 확인
lsof -i :8080
lsof -i :5432

# 다른 포트 사용
PORT=3000 docker compose up
```

### 10.5 macOS 성능 이슈

Docker Desktop 대신 [OrbStack](https://orbstack.dev/) 사용 권장:
- 파일 시스템 성능 최대 10배 향상
- 메모리 사용량 감소
- 빠른 시작 시간

---

## 11. 관련 문서

- [PRD.md](./PRD.md) - 제품 요구사항 문서
- [TRD.md](./TRD.md) - 기술 요구사항 문서

---

## 12. 참고 자료

- [Docker Compose Best Practices](https://docs.docker.com/compose/how-tos/environment-variables/best-practices/)
- [Docker Production Guide](https://docs.docker.com/compose/how-tos/production/)
- [Railway Dockerfile Deployment](https://docs.railway.com/guides/dockerfiles)
- [Neon PostgreSQL Documentation](https://neon.tech/docs)
- [MongoDB Atlas Documentation](https://www.mongodb.com/docs/atlas/)
- [Upstash Redis Documentation](https://upstash.com/docs/redis)

---

*문서 작성일: 2025-01-09*
