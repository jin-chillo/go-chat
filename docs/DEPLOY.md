# GoChat 배포 가이드

## 문서 정보

| 항목 | 내용 |
|-----|------|
| 프로젝트명 | GoChat |
| 버전 | 1.0 |
| 작성일 | 2026-01-14 |
| 상태 | Final |

---

## 1. 개요

이 문서는 GoChat 프로젝트를 프로덕션 환경에 배포하는 방법을 설명합니다.

### 1.1 배포 아키텍처

```
┌─────────────────────────────────────────────────────────────────┐
│                        Production Environment                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   ┌─────────────┐     ┌─────────────────────────────────────┐   │
│   │   Railway   │     │         External Services            │   │
│   │   (or alt)  │     │                                      │   │
│   │             │     │  ┌───────────┐  ┌────────────────┐  │   │
│   │  ┌───────┐  │     │  │   Neon    │  │ MongoDB Atlas  │  │   │
│   │  │GoChat │◀─┼─────┼─▶│PostgreSQL │  │    (512MB)     │  │   │
│   │  │Server │  │     │  │  (Free)   │  └────────────────┘  │   │
│   │  └───────┘  │     │  └───────────┘          ▲           │   │
│   │      │      │     │        ▲                │           │   │
│   │      │      │     │        │         ┌──────┴───────┐   │   │
│   │      ▼      │     │        │         │   Upstash    │   │   │
│   │   HTTPS     │     │        │         │    Redis     │   │   │
│   │  (auto)     │     │        │         │   (Free)     │   │   │
│   └─────────────┘     │        │         └──────────────┘   │   │
│                       │        │                │           │   │
│                       │        └────────────────┘           │   │
│                       └─────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 포트폴리오 프로젝트 안내

이 프로젝트는 포트폴리오용으로 제작되었으며, 실제 프로덕션 배포 없이도 다음을 통해 배포 역량을 입증합니다:

| 구현 항목 | 설명 |
|----------|------|
| Docker 멀티스테이지 빌드 | 최적화된 프로덕션 이미지 (< 20MB) |
| docker-compose 구성 | 개발/프로덕션 환경 분리 |
| 환경 변수 관리 | 환경별 설정 분리 |
| Health Check | 서비스 상태 모니터링 엔드포인트 |
| Graceful Shutdown | 안전한 서버 종료 처리 |

### 1.3 문서 구분

| 문서 | 내용 |
|------|------|
| **DEPLOY.md** (본 문서) | 프로덕션 배포 방법, 클라우드 서비스 설정 |
| [DOCKER_GUIDE.md](./DOCKER_GUIDE.md) | 로컬 Docker 개발 환경 구성, docker-compose 상세 |

> **참고**: 로컬 개발 환경 구성은 [DOCKER_GUIDE.md](./DOCKER_GUIDE.md)를 참조하세요.

---

## 2. 클라우드 서비스 설정

프로덕션 배포를 위해 아래 무료 클라우드 서비스들을 설정합니다.

### 2.1 Neon PostgreSQL

**무료 티어**: 0.5GB 스토리지, 100 compute hours/월

#### 설정 단계

1. [Neon Console](https://console.neon.tech/) 접속 및 회원가입
2. "New Project" 클릭
3. 프로젝트 설정:
   - Project name: `gochat`
   - Region: `Asia Pacific (Singapore)` 권장
   - Postgres version: `16`
4. "Create project" 클릭
5. Connection string 복사

#### Connection String 형식

```
postgres://[user]:[password]@[host]/[database]?sslmode=require
```

**주의**: Neon은 `sslmode=require` 필수

#### 마이그레이션 실행

```bash
# 프로덕션 DB에 마이그레이션 적용
DATABASE_URL="postgres://..." make migrate-up
```

---

### 2.2 MongoDB Atlas

**무료 티어**: 512MB 스토리지, 공유 클러스터

#### 설정 단계

1. [MongoDB Atlas](https://www.mongodb.com/atlas) 접속 및 회원가입
2. "Build a Cluster" 클릭
3. 클러스터 설정:
   - Cloud Provider: AWS
   - Region: `ap-southeast-1 (Singapore)`
   - Cluster Tier: `M0 Sandbox (Free)`
   - Cluster Name: `gochat-cluster`
4. "Create Cluster" 클릭
5. Database Access 설정:
   - "Add New Database User" 클릭
   - Authentication Method: Password
   - Username/Password 설정
   - Database User Privileges: `Read and write to any database`
6. Network Access 설정:
   - "Add IP Address" 클릭
   - `0.0.0.0/0` 입력 (모든 IP 허용) 또는 Railway IP 지정
7. "Connect" → "Connect your application" → Connection string 복사

#### Connection String 형식

```
mongodb+srv://[user]:[password]@[cluster].mongodb.net/gochat?retryWrites=true&w=majority
```

---

### 2.3 Upstash Redis

**무료 티어**: 500K 명령/월, 256MB 데이터

#### 설정 단계

1. [Upstash Console](https://console.upstash.com/) 접속 및 회원가입
2. "Create Database" 클릭
3. 데이터베이스 설정:
   - Name: `gochat-redis`
   - Type: `Regional`
   - Region: `ap-southeast-1 (Singapore)`
   - TLS: `Enabled` (기본값)
4. "Create" 클릭
5. Details 탭에서 `UPSTASH_REDIS_REST_URL` 복사

#### Connection String 형식

```
rediss://default:[password]@[endpoint]:6379
```

**주의**: Upstash는 TLS 사용 (`rediss://`)

---

## 3. Railway 배포

### 3.1 사전 준비

- GitHub 계정 및 저장소
- Railway 계정 ([railway.app](https://railway.app))
- 클라우드 서비스 Connection Strings (섹션 2 참조)

### 3.2 GitHub 연동 배포 (권장)

#### 단계별 가이드

1. **Railway 로그인**
   - GitHub 계정으로 로그인

2. **새 프로젝트 생성**
   - Dashboard → "New Project"
   - "Deploy from GitHub repo" 선택
   - `go-chat` 저장소 선택

3. **환경 변수 설정**
   - Project → Settings → Variables
   - 다음 변수들 추가:

   ```env
   # Server
   PORT=8080
   GIN_MODE=release

   # PostgreSQL (Neon) - 실제 Connection String으로 교체 필수!
   DATABASE_URL=postgres://user:pass@ep-xxx.neon.tech/gochat?sslmode=require

   # MongoDB (Atlas) - 실제 Connection String으로 교체 필수!
   MONGODB_URI=mongodb+srv://user:pass@cluster.mongodb.net/gochat?retryWrites=true&w=majority
   MONGODB_DATABASE=gochat

   # Redis (Upstash) - 실제 Connection String으로 교체 필수!
   REDIS_URL=rediss://default:xxx@xxx.upstash.io:6379

   # JWT - 반드시 강력한 시크릿으로 교체 필수! (최소 32자 랜덤 문자열)
   JWT_SECRET=your-production-256-bit-secret-key-here
   JWT_ACCESS_EXPIRY=15m
   JWT_REFRESH_EXPIRY=168h

   # bcrypt
   BCRYPT_COST=12
   ```

   > **주의**: 위 예시의 `user:pass`, `xxx` 등은 플레이스홀더입니다. 섹션 2에서 생성한 실제 값으로 교체하세요.

4. **배포 설정 확인**
   - Build Command: 자동 감지 (Dockerfile 사용)
   - Start Command: 자동 감지
   - Health Check Path: `/health`

5. **배포 실행**
   - "Deploy" 클릭
   - 빌드 로그 확인

6. **도메인 설정**
   - Settings → Domains
   - Railway 제공 도메인 사용 또는 커스텀 도메인 연결

### 3.3 Railway CLI 배포

```bash
# Railway CLI 설치
npm install -g @railway/cli

# 로그인
railway login

# 프로젝트 연결 (기존 프로젝트) 또는 생성
railway link
# 또는
railway init

# 환경 변수 설정
railway variables set DATABASE_URL="postgres://..."
railway variables set MONGODB_URI="mongodb+srv://..."
railway variables set REDIS_URL="rediss://..."
railway variables set JWT_SECRET="your-secret"
railway variables set GIN_MODE="release"

# 배포
railway up

# 로그 확인
railway logs
```

### 3.4 배포 확인

```bash
# Health Check
curl https://your-app.railway.app/health

# 예상 응답
{
  "status": "healthy",
  "timestamp": "2026-01-14T10:00:00Z",
  "services": {
    "postgresql": "connected",
    "mongodb": "connected",
    "redis": "connected"
  }
}
```

---

## 4. 대안 플랫폼

Railway 외에도 다음 플랫폼에서 배포 가능합니다.

### 4.1 Render

**장점**: 무료 티어 넉넉함, 자동 HTTPS, GitHub 연동

**무료 티어 제한**: 750시간/월, 15분 비활성 시 슬립

```yaml
# render.yaml
services:
  - type: web
    name: gochat
    env: docker
    dockerfilePath: ./Dockerfile
    dockerContext: .
    envVars:
      - key: DATABASE_URL
        sync: false
      - key: MONGODB_URI
        sync: false
      - key: REDIS_URL
        sync: false
      - key: JWT_SECRET
        generateValue: true
    healthCheckPath: /health
```

### 4.2 Fly.io

**장점**: 글로벌 엣지 배포, 빠른 콜드 스타트

**무료 티어 제한**: 3개 공유 VM, 160GB 아웃바운드 트래픽/월

```bash
# Fly.io CLI 설치
curl -L https://fly.io/install.sh | sh

# 로그인
fly auth login

# 앱 생성 및 배포
fly launch

# 환경 변수 설정
fly secrets set DATABASE_URL="postgres://..."
fly secrets set MONGODB_URI="mongodb+srv://..."
fly secrets set REDIS_URL="rediss://..."
fly secrets set JWT_SECRET="your-secret"

# 배포
fly deploy
```

### 4.3 플랫폼 비교

| 플랫폼 | 무료 티어 | 장점 | 단점 |
|-------|----------|------|------|
| Railway | $5 크레딧/월 | 쉬운 설정, GitHub 연동 | 무료 티어 제한적 |
| Render | 750시간/월 | 넉넉한 무료 티어 | 비활성 시 슬립 |
| Fly.io | 3개 VM | 글로벌 배포, 빠른 시작 | 설정 복잡 |

---

## 5. 환경별 체크리스트

### 5.1 로컬 개발 → 프로덕션 전환

| 항목 | 로컬 개발 | 프로덕션 |
|-----|----------|---------|
| GIN_MODE | `debug` | `release` |
| DATABASE_URL | `localhost:5432` | Neon (`sslmode=require`) |
| MONGODB_URI | `localhost:27017` | Atlas (`mongodb+srv://`) |
| REDIS_URL | `localhost:6379` | Upstash (`rediss://`) |
| JWT_SECRET | 개발용 간단한 값 | 256비트 강력한 시크릿 |
| BCRYPT_COST | 12 | 12 (동일) |
| CORS | `*` | 프론트엔드 도메인만 |

### 5.2 보안 체크리스트

- [ ] JWT_SECRET은 최소 32자 이상의 랜덤 문자열 사용
- [ ] 환경 변수에 민감 정보 직접 노출 금지
- [ ] CORS 설정에서 허용 도메인 명시
- [ ] Rate Limiting 활성화 확인
- [ ] HTTPS 강제 (플랫폼 자동 제공)

### 5.3 배포 전 확인사항

```bash
# 로컬에서 프로덕션 설정 테스트
make prod-test

# 빌드 테스트
make prod-build

# 이미지 크기 확인 (< 30MB 권장)
make prod-size

# 테스트 실행
make test
```

---

## 6. 트러블슈팅

### 6.1 데이터베이스 연결 실패

**증상**: `connection refused` 또는 `timeout` 에러

**해결**:
1. Connection string 형식 확인
2. SSL 설정 확인 (Neon: `sslmode=require`, Upstash: `rediss://`)
3. IP 화이트리스트 확인 (Atlas: Network Access)

### 6.2 빌드 실패

**증상**: Docker 빌드 중 에러

**해결**:
```bash
# 로컬에서 빌드 테스트
docker build --target production -t gochat:test .

# 캐시 없이 재빌드
docker build --no-cache --target production -t gochat:test .
```

### 6.3 Health Check 실패

**증상**: 배포 후 서비스 시작되지 않음

**해결**:
1. 로그 확인: `railway logs` 또는 플랫폼 대시보드
2. 환경 변수 누락 확인
3. 포트 설정 확인 (`PORT=8080`)

---

## 7. 관련 문서

- [DOCKER_GUIDE.md](./DOCKER_GUIDE.md) - Docker 개발 환경 상세 가이드
- [TRD.md](./TRD.md) - 기술 요구사항 문서
- [PRD.md](./PRD.md) - 제품 요구사항 문서

---

*문서 작성일: 2026-01-14*
