# GoChat

JWT 인증과 WebSocket 실시간 채팅 기능을 갖춘 Go 백엔드 서버

## 기술 스택

| 구분 | 기술 | 버전 | 서비스 |
|-----|-----|------|--------|
| 언어 | Go | 1.23+ | - |
| 웹 프레임워크 | Gin | v1.10+ | - |
| ORM | GORM | v1.30+ | - |
| 관계형 DB | PostgreSQL | 16 | Neon |
| 문서 DB | MongoDB | 7 | MongoDB Atlas |
| 캐시 | Redis | 7 | Upstash |
| 배포 | Docker | - | Railway |

## 핵심 기능

### 인증 시스템
- JWT 기반 인증 (액세스 토큰 15분, 리프레시 토큰 7일)
- bcrypt 비밀번호 해싱
- 토큰 블랙리스트 (Redis)
- Rate Limiting

### 채팅 시스템
- WebSocket 실시간 통신
- 채널 기반 그룹 채팅
- 메시지 저장 및 조회 (MongoDB)
- 온라인 상태 관리

## 프로젝트 구조

```
gochat/
├── cmd/server/main.go      # 엔트리포인트
├── internal/
│   ├── auth/               # 인증 도메인
│   ├── chat/               # 채팅 도메인
│   ├── config/             # 설정 관리
│   ├── database/           # DB 연결
│   ├── cache/              # Redis 연결
│   ├── middleware/         # 미들웨어
│   └── model/              # 데이터 모델
├── migrations/             # DB 마이그레이션
└── docker-compose.yml
```

## 시작하기

### 요구사항
- Go 1.23+
- Docker & Docker Compose

### 로컬 개발
```bash
# 의존성 설치
go mod download

# 로컬 DB 실행 (Docker)
docker-compose up -d

# 서버 실행
go run cmd/server/main.go

# 테스트
go test ./...
```

### 환경변수
```bash
# .env.example 참조
DATABASE_URL=postgres://user:pass@localhost:5432/gochat?sslmode=disable
MONGODB_URI=mongodb://localhost:27017/gochat
REDIS_URL=redis://localhost:6379
JWT_SECRET=your-secret-key
PORT=8080
```

## API 엔드포인트

### 인증
| Method | Endpoint | 설명 |
|--------|----------|------|
| POST | /api/v1/auth/register | 회원가입 |
| POST | /api/v1/auth/login | 로그인 |
| POST | /api/v1/auth/logout | 로그아웃 |
| POST | /api/v1/auth/refresh | 토큰 갱신 |

### 채팅
| Method | Endpoint | 설명 |
|--------|----------|------|
| GET | /api/v1/channels | 채널 목록 |
| POST | /api/v1/channels | 채널 생성 |
| GET | /api/v1/channels/:id | 채널 상세 |
| GET | /api/v1/channels/:id/messages | 메시지 조회 |
| WS | /ws | WebSocket 연결 |

## 라이선스

MIT License
