# GoChat TRD (Technical Requirements Document)

## 문서 정보

| 항목 | 내용 |
|-----|------|
| 프로젝트명 | GoChat |
| 버전 | 1.0 |
| 작성일 | 2025-01-09 |
| 상태 | Draft |

---

## 1. 시스템 아키텍처

### 1.1 고수준 아키텍처

```
┌─────────────┐     ┌─────────────────────────────────────────┐
│   Client    │     │              GoChat Server              │
│  (Browser)  │────▶│  ┌─────────────────────────────────┐   │
└─────────────┘     │  │         HTTP Router (Gin)        │   │
      │             │  └─────────────────────────────────┘   │
      │ WebSocket   │        │                    │          │
      │             │  ┌─────▼─────┐       ┌─────▼─────┐    │
      │             │  │   Auth    │       │   Chat    │    │
      └────────────▶│  │  Handler  │       │  Handler  │    │
                    │  └─────┬─────┘       └─────┬─────┘    │
                    │        │                    │          │
                    │  ┌─────▼─────┐       ┌─────▼─────┐    │
                    │  │   Auth    │       │   Chat    │    │
                    │  │  Service  │       │  Service  │    │
                    │  └─────┬─────┘       └─────┬─────┘    │
                    │        │                    │          │
                    │  ┌─────▼─────┐       ┌─────▼─────┐    │
                    │  │   Auth    │       │   Chat    │    │
                    │  │   Repo    │       │   Repo    │    │
                    │  └─────┬─────┘       └─────┬─────┘    │
                    └────────┼──────────────────┼───────────┘
                             │                  │
        ┌────────────────────┼──────────────────┼────────────────────┐
        │                    │                  │                    │
   ┌────▼────┐         ┌────▼────┐        ┌────▼────┐         ┌────▼────┐
   │PostgreSQL│         │ MongoDB │        │  Redis  │         │  Redis  │
   │  (Neon)  │         │ (Atlas) │        │(Upstash)│         │ Pub/Sub │
   └──────────┘         └─────────┘        └─────────┘         └─────────┘
   사용자/채널           메시지/로그         세션/캐시           실시간 이벤트
```

### 1.2 계층 구조

```
┌─────────────────────────────────────────────────┐
│                 Presentation Layer               │
│         (HTTP Handlers, WebSocket Handlers)      │
├─────────────────────────────────────────────────┤
│                  Business Layer                  │
│              (Services, Domain Logic)            │
├─────────────────────────────────────────────────┤
│                 Data Access Layer                │
│                  (Repositories)                  │
├─────────────────────────────────────────────────┤
│                Infrastructure Layer              │
│    (Database, Cache, External Services)          │
└─────────────────────────────────────────────────┘
```

---

## 2. 기술 스택

### 2.1 핵심 기술

| 구분 | 기술 | 버전 | 용도 |
|-----|-----|------|------|
| 언어 | Go | 1.24.0 | 메인 개발 언어 |
| 웹 프레임워크 | Gin | v1.11.0 | HTTP 라우팅, 미들웨어 |
| WebSocket | gorilla/websocket | v1.5.3 | 실시간 통신 |
| ORM | GORM | v1.31.1 | PostgreSQL ORM (Generics 지원) |
| MongoDB 드라이버 | mongo-go-driver/v2 | v2.4.1 | MongoDB 연결 |
| Redis 클라이언트 | go-redis | v9.17.2 | Redis 연결 |

### 2.2 인프라

| 서비스 | 제공자 | 무료 티어 제한 |
|-------|-------|---------------|
| PostgreSQL | Neon | 0.5GB 스토리지, 100 CU-hours/월 |
| MongoDB | Atlas | 512MB 스토리지, 공유 vCPU/RAM |
| Redis | Upstash | 500K 명령/월, 256MB 데이터 |
| 배포 | Railway | $5/월 Hobby 플랜 (30일 트라이얼 후) |

### 2.3 개발 도구

| 도구 | 용도 |
|-----|------|
| Docker | 로컬 개발 환경 |
| Docker Compose | 멀티 컨테이너 관리 |
| golang-migrate | DB 마이그레이션 |
| Swagger | API 문서화 |
| Air | Hot reload (개발용) |

---

## 3. 프로젝트 구조

```
gochat/
├── cmd/
│   └── server/
│       └── main.go              # 애플리케이션 진입점
├── internal/
│   ├── auth/                    # 인증 도메인
│   │   ├── handler.go           # HTTP 핸들러
│   │   ├── service.go           # 비즈니스 로직
│   │   ├── repository.go        # 데이터 접근 인터페이스
│   │   ├── repository_pg.go     # PostgreSQL 구현
│   │   ├── repository_mongo.go  # MongoDB 구현 (로그)
│   │   └── dto.go               # 요청/응답 DTO
│   ├── chat/                    # 채팅 도메인
│   │   ├── handler.go           # HTTP 핸들러
│   │   ├── websocket.go         # WebSocket 핸들러
│   │   ├── hub.go               # WebSocket 허브 (연결 관리)
│   │   ├── service.go           # 비즈니스 로직
│   │   ├── repository.go        # 데이터 접근 인터페이스
│   │   ├── repository_pg.go     # PostgreSQL 구현
│   │   ├── repository_mongo.go  # MongoDB 구현 (메시지)
│   │   └── dto.go               # 요청/응답 DTO
│   ├── config/                  # 설정 관리
│   │   └── config.go            # 환경 변수 로드
│   ├── database/                # DB 연결
│   │   ├── postgres.go          # PostgreSQL 연결
│   │   └── mongodb.go           # MongoDB 연결
│   ├── cache/                   # 캐시 레이어
│   │   └── redis.go             # Redis 연결 및 유틸
│   ├── middleware/              # HTTP 미들웨어
│   │   ├── auth.go              # JWT 인증 미들웨어
│   │   ├── cors.go              # CORS 설정
│   │   └── logger.go            # 요청 로깅
│   └── model/                   # 공유 모델
│       ├── user.go              # 사용자 모델
│       ├── channel.go           # 채널 모델
│       └── message.go           # 메시지 모델
├── migrations/                  # DB 마이그레이션 파일
│   ├── 000001_create_users.up.sql
│   ├── 000001_create_users.down.sql
│   └── ...
├── docs/                        # 문서
│   ├── PRD.md
│   └── TRD.md
├── docker-compose.yml           # 로컬 개발 환경
├── Dockerfile                   # 프로덕션 빌드
├── .env.example                 # 환경 변수 예시
├── go.mod
├── go.sum
└── README.md
```

---

## 4. API 설계

### 4.1 인증 API

#### POST /api/v1/auth/register
회원가입

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "securePassword123",
  "nickname": "사용자닉네임"
}
```

**Response (201):**
```json
{
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "nickname": "사용자닉네임",
    "created_at": "2025-01-09T10:00:00Z"
  }
}
```

**Errors:**
- `400`: 유효하지 않은 입력
- `409`: 이메일 중복

---

#### POST /api/v1/auth/login
로그인

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "securePassword123"
}
```

**Response (200):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

**Errors:**
- `400`: 유효하지 않은 입력
- `401`: 인증 실패

---

#### POST /api/v1/auth/refresh
토큰 갱신

**Request Body:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Response (200):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

**Errors:**
- `401`: 유효하지 않거나 만료된 리프레시 토큰

---

#### POST /api/v1/auth/logout
로그아웃

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response (200):**
```json
{
  "message": "Successfully logged out"
}
```

---

### 4.2 채팅 API

#### GET /api/v1/channels
채널 목록 조회

**Headers:**
```
Authorization: Bearer <access_token>
```

**Query Parameters:**
- `page` (optional): 페이지 번호 (default: 1)
- `limit` (optional): 페이지당 항목 수 (default: 20)

**Response (200):**
```json
{
  "channels": [
    {
        "user": {
    "id": "uuid",
      "name": "일반",
      "description": "일반 대화 채널",
      "member_count": 15,
      "created_at": "2025-01-09T10:00:00Z"
    }
  ],
  "total": 10,
  "page": 1,
  "limit": 20
}
```

---

#### POST /api/v1/channels
채널 생성

**Headers:**
```
Authorization: Bearer <access_token>
```

**Request Body:**
```json
{
  "name": "새로운 채널",
  "description": "채널 설명"
}
```

**Response (201):**
```json
{
    "user": {
    "id": "uuid",
  "name": "새로운 채널",
  "description": "채널 설명",
  "owner_id": "user-uuid",
  "created_at": "2025-01-09T10:00:00Z"
  }
}
```

---

#### GET /api/v1/channels/my
내 채널 목록 조회

**Headers:**
```
Authorization: Bearer <access_token>
```

**Query Parameters:**
- `page` (optional): 페이지 번호 (default: 1)
- `limit` (optional): 페이지당 항목 수 (default: 20)

**Response (200):**
```json
{
  "channels": [
    {
        "user": {
    "id": "uuid",
      "name": "일반",
      "description": "일반 대화 채널",
      "member_count": 15,
      "created_at": "2025-01-09T10:00:00Z"
    }
  ],
  "total": 5,
  "page": 1,
  "limit": 20
}
```

---

#### GET /api/v1/channels/:id
채널 상세 조회

**Response (200):**
```json
{
    "user": {
    "id": "uuid",
  "name": "일반",
  "description": "일반 대화 채널",
  "owner_id": "user-uuid",
  "member_count": 15,
  "members": [
    {
        "user": {
    "id": "uuid",
      "nickname": "사용자",
      "is_online": true
    }
  ],
  "created_at": "2025-01-09T10:00:00Z"
  }
}
```

---

#### PUT /api/v1/channels/:id
채널 수정

**Headers:**
```
Authorization: Bearer <access_token>
```

**Request Body:**
```json
{
  "name": "수정된 채널명",
  "description": "수정된 설명"
}
```

**Response (200):**
```json
{
  "channel": {
      "user": {
    "id": "uuid",
    "name": "수정된 채널명",
    "description": "수정된 설명",
    "owner_id": "user-uuid",
    "member_count": 15,
    "created_at": "2025-01-09T10:00:00Z",
    "updated_at": "2025-01-09T11:00:00Z"
  }
}
```

**Errors:**
- `400`: 유효하지 않은 입력
- `403`: 권한 없음 (소유자가 아님)
- `404`: 채널 없음

---

#### DELETE /api/v1/channels/:id
채널 삭제

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response (200):**
```json
{
  "message": "channel deleted successfully"
}
```

**Errors:**
- `403`: 권한 없음 (소유자가 아님)
- `404`: 채널 없음

---

#### POST /api/v1/channels/:id/join
채널 참여

**Response (200):**
```json
{
  "message": "채널에 참여했습니다"
}
```

---

#### POST /api/v1/channels/:id/leave
채널 퇴장

**Response (200):**
```json
{
  "message": "채널에서 퇴장했습니다"
}
```

---

#### GET /api/v1/channels/:id/messages
메시지 이력 조회

**Query Parameters:**
- `before` (optional): 이 시간 이전의 메시지 (커서 페이지네이션)
- `limit` (optional): 조회할 메시지 수 (default: 50, max: 100)

**Response (200):**
```json
{
  "messages": [
    {
      "id": "message-uuid",
      "channel_id": "channel-uuid",
      "user_id": "user-uuid",
      "nickname": "사용자",
      "content": "안녕하세요!",
      "created_at": "2025-01-09T10:30:00Z"
    }
  ],
  "has_more": true
}
```

---

#### GET /api/v1/channels/:id/members/online
온라인 멤버 조회

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response (200):**
```json
{
  "users": [
    {
      "id": "user-uuid",
      "nickname": "사용자"
    }
  ],
  "count": 1
}
```

**Errors:**
- `404`: 채널 없음

---

### 4.3 WebSocket API

#### WS /ws/chat
WebSocket 연결

**Connection:**
```
ws://host/ws/chat?token=<access_token>    (개발 환경)
wss://host/ws/chat?token=<access_token>   (프로덕션 환경, HTTPS 필수)
```

**Client → Server Messages:**

```json
// 메시지 전송
{
  "type": "message",
  "channel_id": "channel-uuid",
  "content": "안녕하세요!"
}

// 타이핑 표시
{
  "type": "typing",
  "channel_id": "channel-uuid"
}

// 채널 구독
{
  "type": "subscribe",
  "channel_id": "channel-uuid"
}

// 채널 구독 해제
{
  "type": "unsubscribe",
  "channel_id": "channel-uuid"
}
```

**Server → Client Messages:**

```json
// 채널 구독 확인
{
  "type": "subscribed",
  "channel_id": "channel-uuid",
  "timestamp": "2025-01-09T10:30:00Z"
}

// 채널 구독 해제 확인
{
  "type": "unsubscribed",
  "channel_id": "channel-uuid",
  "timestamp": "2025-01-09T10:30:00Z"
}

// 새 메시지
{
  "type": "message",
  "channel_id": "channel-uuid",
  "message": {
    "id": "message-uuid",
    "user_id": "user-uuid",
    "nickname": "사용자",
    "content": "안녕하세요!",
    "created_at": "2025-01-09T10:30:00Z"
  },
  "timestamp": "2025-01-09T10:30:00Z"
}

// 타이핑 표시
{
  "type": "typing",
  "channel_id": "channel-uuid",
  "user": {
    "id": "user-uuid",
    "nickname": "사용자"
  },
  "timestamp": "2025-01-09T10:30:00Z"
}

// 사용자 온라인 상태 변경
{
  "type": "presence",
  "channel_id": "channel-uuid",
  "user": {
    "id": "user-uuid",
    "nickname": "사용자"
  },
  "status": "online",  // or "offline"
  "timestamp": "2025-01-09T10:30:00Z"
}

// 에러
{
  "type": "error",
  "error": "에러 메시지"
}
```

---

## 5. 데이터 모델

### 5.1 PostgreSQL 스키마

#### users 테이블
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    nickname VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
```

#### refresh_tokens 테이블
```sql
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
```

#### channels 테이블
```sql
CREATE TABLE channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_channels_owner_id ON channels(owner_id);
```

#### channel_members 테이블
```sql
CREATE TABLE channel_members (
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (channel_id, user_id)
);

CREATE INDEX idx_channel_members_user_id ON channel_members(user_id);
```

### 5.2 MongoDB 스키마

#### messages 컬렉션
```javascript
{
  "_id": ObjectId("..."),
  "channel_id": "uuid-string",
  "user_id": "uuid-string",
  "nickname": "사용자닉네임",
  "content": "메시지 내용",
  "created_at": ISODate("2025-01-09T10:30:00Z")
}

// Indexes
db.messages.createIndex({ "channel_id": 1, "created_at": -1 })
```

#### login_history 컬렉션
```javascript
{
  "_id": ObjectId("..."),
  "user_id": "uuid-string",
  "ip_address": "192.168.1.1",
  "user_agent": "Mozilla/5.0...",
  "success": true,
  "created_at": ISODate("2025-01-09T10:00:00Z")
}

// Indexes
db.login_history.createIndex({ "user_id": 1, "created_at": -1 })
db.login_history.createIndex({ "created_at": 1 }, { expireAfterSeconds: 7776000 }) // 90일 후 자동 삭제
```

### 5.3 Redis 구조

```
# 세션 데이터
session:{session_id} -> JSON { user_id, created_at, ip }
TTL: 7일

# 토큰 블랙리스트
blacklist:{jti} -> 1
TTL: access_token 남은 유효기간

# 온라인 상태
online:{user_id} -> timestamp
TTL: 5분 (heartbeat로 갱신)

# 채널별 온라인 사용자
channel:{channel_id}:online -> SET { user_id1, user_id2, ... }

# Pub/Sub 채널
chat:{channel_id} -> 메시지 발행
```

---

## 6. 인증 플로우

### 6.1 JWT 구조

**Access Token (15분):**
```json
{
  "header": {
    "alg": "HS256",
    "typ": "JWT"
  },
  "payload": {
    "sub": "user-uuid",
    "email": "user@example.com",
    "nickname": "사용자",
    "jti": "unique-token-id",
    "iat": 1704700800,
    "exp": 1704701700
  }
}
```

**Refresh Token (7일):**
```json
{
  "payload": {
    "sub": "user-uuid",
    "jti": "unique-token-id",
    "type": "refresh",
    "iat": 1704700800,
    "exp": 1705305600
  }
}
```

### 6.2 인증 시퀀스

```
┌──────┐          ┌──────┐          ┌──────┐          ┌──────┐
│Client│          │Server│          │ Redis│          │  DB  │
└──┬───┘          └──┬───┘          └──┬───┘          └──┬───┘
   │   POST /login   │                 │                 │
   │────────────────▶│                 │                 │
   │                 │ 사용자 조회      │                 │
   │                 │─────────────────────────────────▶│
   │                 │◀─────────────────────────────────│
   │                 │                 │                 │
   │                 │ 세션 저장        │                 │
   │                 │────────────────▶│                 │
   │                 │                 │                 │
   │  access_token   │                 │                 │
   │  refresh_token  │                 │                 │
   │◀────────────────│                 │                 │
   │                 │                 │                 │
   │  GET /api (JWT) │                 │                 │
   │────────────────▶│                 │                 │
   │                 │ 블랙리스트 확인  │                 │
   │                 │────────────────▶│                 │
   │                 │◀────────────────│                 │
   │                 │                 │                 │
   │    Response     │                 │                 │
   │◀────────────────│                 │                 │
```

---

## 7. WebSocket 아키텍처

### 7.1 Hub 패턴

```go
type Hub struct {
    // 채널별 연결된 클라이언트
    channels map[string]map[*Client]bool

    // 클라이언트 등록
    register chan *Client

    // 클라이언트 해제
    unregister chan *Client

    // 브로드캐스트 메시지
    broadcast chan *Message
}
```

### 7.2 연결 흐름

```
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│  Client  │     │   Hub    │     │  Redis   │     │  Other   │
│          │     │          │     │  Pub/Sub │     │ Servers  │
└────┬─────┘     └────┬─────┘     └────┬─────┘     └────┬─────┘
     │                │                │                │
     │  WS Connect    │                │                │
     │───────────────▶│                │                │
     │                │                │                │
     │  Register      │                │                │
     │◀───────────────│                │                │
     │                │                │                │
     │  Subscribe     │                │                │
     │───────────────▶│                │                │
     │                │  Subscribe     │                │
     │                │───────────────▶│                │
     │                │                │                │
     │  Send Message  │                │                │
     │───────────────▶│                │                │
     │                │   Publish      │                │
     │                │───────────────▶│                │
     │                │                │   Broadcast    │
     │                │                │───────────────▶│
     │   Broadcast    │◀───────────────│                │
     │◀───────────────│                │                │
```

---

## 8. 보안 요구사항

### 8.1 인증 보안
- 비밀번호: bcrypt 해싱 (cost=12)
- JWT 서명: HMAC-SHA256
- 토큰 저장: HttpOnly Cookie 또는 메모리 (클라이언트)
- HTTPS 강제 (프로덕션)

### 8.2 입력 검증
```go
type RegisterRequest struct {
    Email    string `json:"email" binding:"required,email,max=255"`
    Password string `json:"password" binding:"required,min=8,max=72"`
    Nickname string `json:"nickname" binding:"required,min=2,max=50"`
}
```

### 8.3 Rate Limiting
| 엔드포인트 | 제한 |
|-----------|------|
| POST /api/v1/auth/login | 5회/분 (IP 기준) |
| POST /api/v1/auth/register | 3회/분 (IP 기준) |
| 기타 API | 100회/분 (사용자 기준) |
| WebSocket 메시지 | 30회/분 (사용자 기준) |

### 8.4 CORS 설정
```go
cors.Config{
    AllowOrigins:     []string{"https://your-frontend.com"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders:     []string{"Authorization", "Content-Type"},
    AllowCredentials: true,
    MaxAge:           12 * time.Hour,
}
```

---

## 9. 성능 요구사항

### 9.1 응답 시간 목표
| 작업 | 목표 |
|-----|------|
| API 응답 (P95) | < 100ms |
| DB 쿼리 | < 50ms |
| WebSocket 메시지 전달 | < 50ms |

### 9.2 처리량 목표
| 메트릭 | 목표 (무료 티어 기준) |
|-------|---------------------|
| 동시 WebSocket 연결 | 100+ |
| API 요청/초 | 50+ |
| 메시지/초 | 100+ |

### 9.3 리소스 제한
| 리소스 | 제한 |
|-------|------|
| PostgreSQL 커넥션 풀 | 10 |
| MongoDB 커넥션 풀 | 10 |
| Redis 커넥션 | 5 |

---

## 10. 배포 전략

### 10.1 환경 변수
```env
# Server
PORT=8080
ENV=production

# PostgreSQL (Neon)
DATABASE_URL=postgres://user:pass@host/db?sslmode=require

# MongoDB (Atlas)
MONGODB_URI=mongodb+srv://user:pass@cluster.mongodb.net/gochat

# Redis (Upstash)
REDIS_URL=rediss://default:token@host:6379

# JWT
JWT_SECRET=your-256-bit-secret
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h

# CORS
ALLOWED_ORIGINS=https://your-frontend.com
```

### 10.2 Docker 설정
```dockerfile
# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Run stage
FROM alpine:3.21
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
```

### 10.3 Railway 배포
- GitHub 연동 자동 배포
- 환경 변수 설정
- Health check: `GET /health`
- 자동 HTTPS

---

## 11. 모니터링 및 로깅

### 11.1 Health Check
```json
GET /health

{
  "status": "healthy",
  "timestamp": "2025-01-09T10:00:00Z",
  "services": {
    "postgresql": "connected",
    "mongodb": "connected",
    "redis": "connected"
  }
}
```

### 11.2 로깅 형식
```json
{
  "level": "info",
  "timestamp": "2025-01-09T10:00:00Z",
  "method": "POST",
  "path": "/api/v1/auth/login",
  "status": 200,
  "latency": "45ms",
  "ip": "192.168.1.1",
  "user_id": "uuid"
}
```

---

## 12. 테스트 전략

### 12.1 테스트 유형
| 유형 | 도구 | 커버리지 목표 |
|-----|-----|-------------|
| 단위 테스트 | Go testing | 70% |
| 통합 테스트 | testcontainers-go | 주요 플로우 |
| API 테스트 | httptest | 모든 엔드포인트 |

### 12.2 테스트 실행
```bash
# 단위 테스트
go test ./...

# 커버리지
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## 13. 의존성 목록

### 13.1 Go 모듈
```go
require (
    github.com/gin-gonic/gin v1.11.0
    github.com/gorilla/websocket v1.5.3
    gorm.io/gorm v1.31.1
    gorm.io/driver/postgres v1.6.0
    go.mongodb.org/mongo-driver/v2 v2.4.1
    github.com/redis/go-redis/v9 v9.17.2
    github.com/golang-jwt/jwt/v5 v5.3.0
    golang.org/x/crypto v0.47.0
    github.com/caarlos0/env/v11 v11.3.1
    github.com/google/uuid v1.6.0
)
```

---

## 14. 부록

### 14.1 에러 코드
| 코드 | 설명 |
|-----|------|
| AUTH001 | 이메일 형식 오류 |
| AUTH002 | 비밀번호 조건 미충족 |
| AUTH003 | 이메일 중복 |
| AUTH004 | 인증 실패 |
| AUTH005 | 토큰 만료 |
| AUTH006 | 토큰 무효 |
| CHAT001 | 채널 없음 |
| CHAT002 | 권한 없음 |
| CHAT003 | WebSocket 연결 실패 |

### 14.2 관련 문서
- [PRD.md](./PRD.md) - 제품 요구사항 문서
- [DOCKER_GUIDE.md](./DOCKER_GUIDE.md) - Docker 개발 및 배포 가이드
- [project-overview.md](../project-overview.md) - 프로젝트 개요

---

*문서 작성일: 2025-01-09 (업데이트)*
