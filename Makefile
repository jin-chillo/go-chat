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
