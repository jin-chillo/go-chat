# GoChat API 테스트 가이드

이 문서는 Postman 또는 Insomnia를 사용하여 GoChat API를 수동 테스트하는 방법을 안내합니다.

## 목차

- [사전 준비](#사전-준비)
- [인증 API](#인증-api)
- [채팅 API](#채팅-api)
- [WebSocket API](#websocket-api)
- [테스트 시나리오](#테스트-시나리오)
- [에러 코드 참조](#에러-코드-참조)

## 사전 준비

### 1. 서버 실행

```bash
# DB 컨테이너 실행
make db

# 마이그레이션 적용
make migrate-up

# 서버 실행 (별도 터미널)
go run ./cmd/server
```

### 2. 환경 변수 설정 (Postman/Insomnia)

| 변수명 | 값 |
|--------|-----|
| `base_url` | `http://localhost:8080` |
| `access_token` | (로그인 후 자동 설정) |
| `refresh_token` | (로그인 후 자동 설정) |

---

## 인증 API

### 1. Health Check

서버 상태를 확인합니다.

| 항목 | 값 |
|------|-----|
| Method | `GET` |
| URL | `{{base_url}}/health` |
| Headers | 없음 |
| Body | 없음 |

**예상 응답 (200 OK):**
```json
{
  "status": "ok"
}
```

---

### 2. 회원가입 (Register)

새 사용자를 등록합니다.

| 항목 | 값 |
|------|-----|
| Method | `POST` |
| URL | `{{base_url}}/api/v1/auth/register` |
| Headers | `Content-Type: application/json` |

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "password123",
  "nickname": "MyNickname"
}
```

**유효성 검사 규칙:**
- `email`: 유효한 이메일 형식, 최대 255자
- `password`: 최소 8자, 최대 72자
- `nickname`: 최소 2자, 최대 50자

**예상 응답 (201 Created):**
```json
{
  "user": {
    "id": "uuid-here",
    "email": "user@example.com",
    "nickname": "MyNickname",
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

**에러 응답:**

| Status | Code | 설명 |
|--------|------|------|
| 400 | AUTH001 | 유효성 검사 실패 |
| 409 | AUTH003 | 이메일 중복 |
| 429 | RATE001 | Rate Limit 초과 (3회/분) |

---

### 3. 로그인 (Login)

사용자 인증 후 토큰을 발급받습니다.

| 항목 | 값 |
|------|-----|
| Method | `POST` |
| URL | `{{base_url}}/api/v1/auth/login` |
| Headers | `Content-Type: application/json` |

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**예상 응답 (200 OK):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "550e8400-e29b-41d4-a716-446655440000",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

> **Postman/Insomnia 팁:** 응답에서 `access_token`과 `refresh_token`을 환경 변수에 자동 저장하세요.

**Postman 스크립트 (Tests 탭):**
```javascript
if (pm.response.code === 200) {
    var jsonData = pm.response.json();
    pm.environment.set("access_token", jsonData.access_token);
    pm.environment.set("refresh_token", jsonData.refresh_token);
}
```

**Insomnia 환경 설정:**
Response → JSONPath로 `$.access_token` 추출

**에러 응답:**

| Status | Code | 설명 |
|--------|------|------|
| 401 | AUTH004 | 이메일 또는 비밀번호 불일치 |
| 429 | RATE001 | Rate Limit 초과 (5회/분) |

---

### 4. 토큰 갱신 (Refresh)

만료된 Access Token을 갱신합니다.

| 항목 | 값 |
|------|-----|
| Method | `POST` |
| URL | `{{base_url}}/api/v1/auth/refresh` |
| Headers | `Content-Type: application/json` |

**Request Body:**
```json
{
  "refresh_token": "{{refresh_token}}"
}
```

**예상 응답 (200 OK):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

**에러 응답:**

| Status | Code | 설명 |
|--------|------|------|
| 400 | AUTH001 | refresh_token 누락 |
| 401 | AUTH006 | 유효하지 않거나 만료된 토큰 |

---

### 5. 로그아웃 (Logout)

현재 Access Token을 무효화합니다.

| 항목 | 값 |
|------|-----|
| Method | `POST` |
| URL | `{{base_url}}/api/v1/auth/logout` |
| Headers | `Authorization: Bearer {{access_token}}` |
| Body | 없음 |

**예상 응답 (200 OK):**
```json
{
  "message": "Successfully logged out"
}
```

**에러 응답:**

| Status | Code | 설명 |
|--------|------|------|
| 401 | AUTH006 | 토큰 없음, 유효하지 않음, 또는 블랙리스트 |

---

## 채팅 API

> **참고:** 모든 채팅 API는 `Authorization: Bearer {{access_token}}` 헤더가 필요합니다.

### 1. 채널 목록 조회 (List Channels)

전체 채널 목록을 조회합니다.

| 항목 | 값 |
|------|-----|
| Method | `GET` |
| URL | `{{base_url}}/api/v1/channels` |
| Headers | `Authorization: Bearer {{access_token}}` |
| Query | `page` (기본: 1), `limit` (기본: 20, 최대: 100) |

**예상 응답 (200 OK):**
```json
{
  "channels": [
    {
      "id": "uuid-here",
      "name": "General",
      "description": "General discussion",
      "owner_id": "owner-uuid",
      "owner": {
        "id": "owner-uuid",
        "nickname": "Admin"
      },
      "member_count": 5,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "limit": 20
}
```

---

### 2. 내 채널 목록 조회 (List My Channels)

현재 사용자가 참여한 채널 목록을 조회합니다.

| 항목 | 값 |
|------|-----|
| Method | `GET` |
| URL | `{{base_url}}/api/v1/channels/my` |
| Headers | `Authorization: Bearer {{access_token}}` |
| Query | `page` (기본: 1), `limit` (기본: 20, 최대: 100) |

**예상 응답:** 채널 목록 조회와 동일한 형식

---

### 3. 채널 생성 (Create Channel)

새 채널을 생성합니다. 생성자는 자동으로 멤버가 됩니다.

| 항목 | 값 |
|------|-----|
| Method | `POST` |
| URL | `{{base_url}}/api/v1/channels` |
| Headers | `Authorization: Bearer {{access_token}}`, `Content-Type: application/json` |

**Request Body:**
```json
{
  "name": "My Channel",
  "description": "Channel description (optional)"
}
```

**유효성 검사 규칙:**
- `name`: 필수, 최소 1자, 최대 100자
- `description`: 선택, 최대 500자

**예상 응답 (201 Created):**
```json
{
  "channel": {
    "id": "uuid-here",
    "name": "My Channel",
    "description": "Channel description",
    "owner_id": "your-uuid",
    "member_count": 1,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

---

### 4. 채널 상세 조회 (Get Channel)

채널 상세 정보와 멤버 목록을 조회합니다.

| 항목 | 값 |
|------|-----|
| Method | `GET` |
| URL | `{{base_url}}/api/v1/channels/:id` |
| Headers | `Authorization: Bearer {{access_token}}` |

**예상 응답 (200 OK):**
```json
{
  "channel": {
    "id": "uuid-here",
    "name": "My Channel",
    "description": "Channel description",
    "owner_id": "owner-uuid",
    "owner": {
      "id": "owner-uuid",
      "nickname": "Admin"
    },
    "members": [
      {
        "id": "user-uuid",
        "nickname": "User1",
        "joined_at": "2024-01-01T00:00:00Z"
      }
    ],
    "member_count": 1,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

**에러 응답:**

| Status | Code | 설명 |
|--------|------|------|
| 404 | CHAT001 | 채널을 찾을 수 없음 |

---

### 5. 채널 수정 (Update Channel)

채널 정보를 수정합니다. 채널 소유자만 가능합니다.

| 항목 | 값 |
|------|-----|
| Method | `PUT` |
| URL | `{{base_url}}/api/v1/channels/:id` |
| Headers | `Authorization: Bearer {{access_token}}`, `Content-Type: application/json` |

**Request Body:**
```json
{
  "name": "Updated Name",
  "description": "Updated description"
}
```

**에러 응답:**

| Status | Code | 설명 |
|--------|------|------|
| 403 | CHAT002 | 채널 소유자가 아님 |
| 404 | CHAT001 | 채널을 찾을 수 없음 |

---

### 6. 채널 삭제 (Delete Channel)

채널을 삭제합니다. 채널 소유자만 가능합니다.

| 항목 | 값 |
|------|-----|
| Method | `DELETE` |
| URL | `{{base_url}}/api/v1/channels/:id` |
| Headers | `Authorization: Bearer {{access_token}}` |

**예상 응답 (200 OK):**
```json
{
  "message": "channel deleted successfully"
}
```

---

### 7. 채널 참여 (Join Channel)

채널에 참여합니다.

| 항목 | 값 |
|------|-----|
| Method | `POST` |
| URL | `{{base_url}}/api/v1/channels/:id/join` |
| Headers | `Authorization: Bearer {{access_token}}` |
| Body | 없음 |

**예상 응답 (200 OK):**
```json
{
  "message": "채널에 참여했습니다"
}
```

**에러 응답:**

| Status | Code | 설명 |
|--------|------|------|
| 404 | CHAT001 | 채널을 찾을 수 없음 |
| 409 | CHAT002 | 이미 채널 멤버임 |

---

### 8. 채널 퇴장 (Leave Channel)

채널에서 퇴장합니다. 채널 소유자는 퇴장할 수 없습니다.

| 항목 | 값 |
|------|-----|
| Method | `POST` |
| URL | `{{base_url}}/api/v1/channels/:id/leave` |
| Headers | `Authorization: Bearer {{access_token}}` |
| Body | 없음 |

**예상 응답 (200 OK):**
```json
{
  "message": "채널에서 퇴장했습니다"
}
```

**에러 응답:**

| Status | Code | 설명 |
|--------|------|------|
| 400 | CHAT002 | 채널 멤버가 아님 |
| 403 | CHAT002 | 채널 소유자는 퇴장 불가 |
| 404 | CHAT001 | 채널을 찾을 수 없음 |

---

### 9. 메시지 조회 (Get Messages)

채널의 메시지 이력을 조회합니다. 커서 기반 페이지네이션을 사용합니다.

| 항목 | 값 |
|------|-----|
| Method | `GET` |
| URL | `{{base_url}}/api/v1/channels/:id/messages` |
| Headers | `Authorization: Bearer {{access_token}}` |
| Query | `before` (RFC3339 timestamp), `limit` (기본: 50, 최대: 100) |

**예상 응답 (200 OK):**
```json
{
  "messages": [
    {
      "id": "mongodb-objectid",
      "channel_id": "channel-uuid",
      "user_id": "user-uuid",
      "nickname": "User1",
      "content": "Hello, world!",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "has_more": true
}
```

**페이지네이션 사용:**
```
# 첫 페이지
GET /api/v1/channels/:id/messages?limit=50

# 다음 페이지 (마지막 메시지의 created_at 사용)
GET /api/v1/channels/:id/messages?before=2024-01-01T00:00:00Z&limit=50
```

---

### 10. 온라인 멤버 조회 (Get Online Members)

채널에서 현재 온라인인 멤버 목록을 조회합니다. (WebSocket 연결 및 채널 구독 상태 기준)

| 항목 | 값 |
|------|-----|
| Method | `GET` |
| URL | `{{base_url}}/api/v1/channels/:id/members/online` |
| Headers | `Authorization: Bearer {{access_token}}` |

**예상 응답 (200 OK):**
```json
{
  "users": [
    {
      "id": "user-uuid",
      "nickname": "User1"
    },
    {
      "id": "user-uuid-2",
      "nickname": "User2"
    }
  ],
  "count": 2
}
```

> **참고:** 온라인 상태는 Redis에 저장되며, WebSocket으로 채널에 구독한 사용자만 표시됩니다.

---

## WebSocket API

### 연결 (Connect)

| 항목 | 값 |
|------|-----|
| URL | `ws://localhost:8080/ws/chat?token={{access_token}}` |
| Protocol | WebSocket |

**연결 예시 (JavaScript):**
```javascript
const token = "your-access-token";
const ws = new WebSocket(`ws://localhost:8080/ws/chat?token=${token}`);

ws.onopen = () => {
  console.log("Connected");
};

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  console.log("Received:", msg);
};

ws.onerror = (error) => {
  console.error("Error:", error);
};
```

**인증 에러 응답:**

| Status | Code | 설명 |
|--------|------|------|
| 401 | AUTH006 | 토큰 없음 |
| 401 | AUTH006 | 유효하지 않은 토큰 |

---

### 메시지 타입

#### 1. 채널 구독 (subscribe)

채널에 구독하여 메시지를 수신합니다.

**Request:**
```json
{
  "type": "subscribe",
  "channel_id": "channel-uuid"
}
```

**Response:**
```json
{
  "type": "subscribed",
  "channel_id": "channel-uuid",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

---

#### 2. 채널 구독 해제 (unsubscribe)

채널 구독을 해제합니다.

**Request:**
```json
{
  "type": "unsubscribe",
  "channel_id": "channel-uuid"
}
```

**Response:**
```json
{
  "type": "unsubscribed",
  "channel_id": "channel-uuid",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

---

#### 3. 메시지 전송 (message)

구독한 채널에 메시지를 전송합니다.

**Request:**
```json
{
  "type": "message",
  "channel_id": "channel-uuid",
  "content": "Hello, everyone!"
}
```

**Broadcast (모든 구독자에게 전송):**
```json
{
  "type": "message",
  "channel_id": "channel-uuid",
  "message": {
    "id": "mongodb-objectid",
    "user_id": "sender-uuid",
    "nickname": "Sender",
    "content": "Hello, everyone!",
    "created_at": "2024-01-01T00:00:00Z"
  },
  "timestamp": "2024-01-01T00:00:00Z"
}
```

---

#### 4. 타이핑 표시 (typing)

타이핑 중임을 알립니다.

**Request:**
```json
{
  "type": "typing",
  "channel_id": "channel-uuid"
}
```

**Broadcast (발신자 제외):**
```json
{
  "type": "typing",
  "channel_id": "channel-uuid",
  "user": {
    "id": "user-uuid",
    "nickname": "User1"
  },
  "timestamp": "2024-01-01T00:00:00Z"
}
```

---

#### 5. 접속 상태 (presence)

사용자가 채널에 접속/퇴장할 때 자동으로 전송됩니다.

**Broadcast:**
```json
{
  "type": "presence",
  "channel_id": "channel-uuid",
  "user": {
    "id": "user-uuid",
    "nickname": "User1"
  },
  "status": "online",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

`status` 값: `online` (접속) 또는 `offline` (퇴장)

---

#### 6. 에러 (error)

요청 처리 중 오류가 발생한 경우 전송됩니다.

```json
{
  "type": "error",
  "error": "not a member of this channel",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

---

## 테스트 시나리오

### 시나리오 1: 인증 기본 플로우

1. **회원가입** → 201 응답 확인
2. **로그인** → 토큰 발급 확인, 환경 변수에 저장
3. **로그아웃** → 200 응답 확인
4. **로그아웃 재시도** → 401 응답 확인 (토큰 블랙리스트)

### 시나리오 2: 토큰 갱신 플로우

1. **로그인** → 토큰 발급
2. **토큰 갱신** → 새 Access Token 발급 확인
3. **새 토큰으로 로그아웃** → 성공 확인

### 시나리오 3: 채팅 기본 플로우

1. **로그인** → 토큰 발급
2. **채널 생성** → 201 응답, 채널 ID 저장
3. **채널 목록 조회** → 생성한 채널 확인
4. **내 채널 목록** → 생성한 채널 확인
5. **채널 상세 조회** → 멤버 목록에 본인 포함 확인
6. **메시지 조회** → 빈 배열 확인

### 시나리오 4: 채널 참여/퇴장 플로우

1. **사용자 A 로그인** → 토큰 발급
2. **채널 생성** → 채널 ID 저장
3. **사용자 B 로그인** → 다른 토큰 발급
4. **사용자 B 채널 참여** → 200 응답
5. **채널 상세 조회** → 멤버 2명 확인
6. **사용자 B 채널 퇴장** → 200 응답
7. **사용자 A 채널 퇴장 시도** → 403 응답 (소유자)

### 시나리오 5: WebSocket 플로우

1. **로그인** → 토큰 발급
2. **채널 생성 및 참여**
3. **WebSocket 연결** (`ws://localhost:8080/ws/chat?token=...`)
4. **채널 구독** → `subscribed` 응답 확인
5. **메시지 전송** → 브로드캐스트 수신 확인
6. **온라인 멤버 조회** → 본인 포함 확인
7. **WebSocket 연결 해제**
8. **온라인 멤버 조회** → 빈 배열 확인

### 시나리오 6: 에러 케이스

1. **중복 이메일로 회원가입** → 409 응답 확인
2. **잘못된 비밀번호로 로그인** → 401 응답 확인
3. **존재하지 않는 채널 조회** → 404 응답 확인
4. **다른 사용자의 채널 수정** → 403 응답 확인
5. **미가입 채널에서 퇴장** → 400 응답 확인
6. **토큰 없이 WebSocket 연결** → 401 응답 확인

### 시나리오 7: Rate Limiting

1. **회원가입 4회 연속 요청** → 4번째에서 429 응답 확인
2. 1분 대기 후 다시 시도 → 성공

---

## Rate Limiting 정책

| 엔드포인트 | 제한 |
|-----------|------|
| POST /api/v1/auth/register | 3회/분 |
| POST /api/v1/auth/login | 5회/분 |

---

## 에러 코드 참조

### 인증 에러

| Code | 설명 |
|------|------|
| AUTH001 | 유효성 검사 실패 |
| AUTH003 | 이메일 중복 |
| AUTH004 | 잘못된 이메일 또는 비밀번호 |
| AUTH006 | 토큰 관련 오류 (없음, 만료, 블랙리스트) |

### 채팅 에러

| Code | 설명 |
|------|------|
| CHAT001 | 채널을 찾을 수 없음, 유효하지 않은 요청 |
| CHAT002 | 권한 관련 오류 (소유자 아님, 멤버 아님, 이미 멤버) |
| CHAT003 | 메시지 관련 오류 |

### 기타 에러

| Code | 설명 |
|------|------|
| RATE001 | Rate Limit 초과 |

---

## Postman Collection 가져오기

아래 JSON을 Postman에서 Import하여 사용할 수 있습니다:

```json
{
  "info": {
    "name": "GoChat API",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "variable": [
    {"key": "base_url", "value": "http://localhost:8080"},
    {"key": "access_token", "value": ""},
    {"key": "refresh_token", "value": ""},
    {"key": "channel_id", "value": ""}
  ],
  "item": [
    {
      "name": "Auth",
      "item": [
        {
          "name": "Health Check",
          "request": {
            "method": "GET",
            "url": "{{base_url}}/health"
          }
        },
        {
          "name": "Register",
          "request": {
            "method": "POST",
            "url": "{{base_url}}/api/v1/auth/register",
            "header": [{"key": "Content-Type", "value": "application/json"}],
            "body": {
              "mode": "raw",
              "raw": "{\"email\": \"test@example.com\", \"password\": \"password123\", \"nickname\": \"TestUser\"}"
            }
          }
        },
        {
          "name": "Login",
          "request": {
            "method": "POST",
            "url": "{{base_url}}/api/v1/auth/login",
            "header": [{"key": "Content-Type", "value": "application/json"}],
            "body": {
              "mode": "raw",
              "raw": "{\"email\": \"test@example.com\", \"password\": \"password123\"}"
            }
          },
          "event": [
            {
              "listen": "test",
              "script": {
                "exec": [
                  "if (pm.response.code === 200) {",
                  "    var jsonData = pm.response.json();",
                  "    pm.collectionVariables.set('access_token', jsonData.access_token);",
                  "    pm.collectionVariables.set('refresh_token', jsonData.refresh_token);",
                  "}"
                ]
              }
            }
          ]
        },
        {
          "name": "Refresh Token",
          "request": {
            "method": "POST",
            "url": "{{base_url}}/api/v1/auth/refresh",
            "header": [{"key": "Content-Type", "value": "application/json"}],
            "body": {
              "mode": "raw",
              "raw": "{\"refresh_token\": \"{{refresh_token}}\"}"
            }
          }
        },
        {
          "name": "Logout",
          "request": {
            "method": "POST",
            "url": "{{base_url}}/api/v1/auth/logout",
            "header": [{"key": "Authorization", "value": "Bearer {{access_token}}"}]
          }
        }
      ]
    },
    {
      "name": "Channels",
      "item": [
        {
          "name": "List Channels",
          "request": {
            "method": "GET",
            "url": "{{base_url}}/api/v1/channels",
            "header": [{"key": "Authorization", "value": "Bearer {{access_token}}"}]
          }
        },
        {
          "name": "List My Channels",
          "request": {
            "method": "GET",
            "url": "{{base_url}}/api/v1/channels/my",
            "header": [{"key": "Authorization", "value": "Bearer {{access_token}}"}]
          }
        },
        {
          "name": "Create Channel",
          "request": {
            "method": "POST",
            "url": "{{base_url}}/api/v1/channels",
            "header": [
              {"key": "Authorization", "value": "Bearer {{access_token}}"},
              {"key": "Content-Type", "value": "application/json"}
            ],
            "body": {
              "mode": "raw",
              "raw": "{\"name\": \"Test Channel\", \"description\": \"A test channel\"}"
            }
          },
          "event": [
            {
              "listen": "test",
              "script": {
                "exec": [
                  "if (pm.response.code === 201) {",
                  "    var jsonData = pm.response.json();",
                  "    pm.collectionVariables.set('channel_id', jsonData.channel.id);",
                  "}"
                ]
              }
            }
          ]
        },
        {
          "name": "Get Channel",
          "request": {
            "method": "GET",
            "url": "{{base_url}}/api/v1/channels/{{channel_id}}",
            "header": [{"key": "Authorization", "value": "Bearer {{access_token}}"}]
          }
        },
        {
          "name": "Update Channel",
          "request": {
            "method": "PUT",
            "url": "{{base_url}}/api/v1/channels/{{channel_id}}",
            "header": [
              {"key": "Authorization", "value": "Bearer {{access_token}}"},
              {"key": "Content-Type", "value": "application/json"}
            ],
            "body": {
              "mode": "raw",
              "raw": "{\"name\": \"Updated Channel\", \"description\": \"Updated description\"}"
            }
          }
        },
        {
          "name": "Delete Channel",
          "request": {
            "method": "DELETE",
            "url": "{{base_url}}/api/v1/channels/{{channel_id}}",
            "header": [{"key": "Authorization", "value": "Bearer {{access_token}}"}]
          }
        },
        {
          "name": "Join Channel",
          "request": {
            "method": "POST",
            "url": "{{base_url}}/api/v1/channels/{{channel_id}}/join",
            "header": [{"key": "Authorization", "value": "Bearer {{access_token}}"}]
          }
        },
        {
          "name": "Leave Channel",
          "request": {
            "method": "POST",
            "url": "{{base_url}}/api/v1/channels/{{channel_id}}/leave",
            "header": [{"key": "Authorization", "value": "Bearer {{access_token}}"}]
          }
        },
        {
          "name": "Get Messages",
          "request": {
            "method": "GET",
            "url": "{{base_url}}/api/v1/channels/{{channel_id}}/messages",
            "header": [{"key": "Authorization", "value": "Bearer {{access_token}}"}]
          }
        },
        {
          "name": "Get Online Members",
          "request": {
            "method": "GET",
            "url": "{{base_url}}/api/v1/channels/{{channel_id}}/members/online",
            "header": [{"key": "Authorization", "value": "Bearer {{access_token}}"}]
          }
        }
      ]
    }
  ]
}
```

---

## 문제 해결

### Rate Limit에 걸린 경우
```bash
# Redis 초기화
docker exec go-chat-redis-1 redis-cli FLUSHALL
```

### 서버가 시작되지 않는 경우
```bash
# 컨테이너 상태 확인
docker compose ps

# 로그 확인
docker compose logs
```
