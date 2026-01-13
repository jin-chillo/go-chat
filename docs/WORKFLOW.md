# 작업 플로우

Linear 이슈 기반 Git Flow 작업 방식을 따릅니다.

## 브랜치 전략

```
main (프로덕션)
  └── dev (개발 통합)
        └── feature/GO-X-desc (기능 개발)
```

- **main**: 프로덕션 배포용 (안정 버전)
- **dev**: 개발 통합 브랜치 (모든 feature PR의 타겟)
- **feature/\***: 개별 기능 개발 브랜치

> 참고: GitHub 기본 브랜치는 `main`으로 설정되어 있지만, 일상적인 개발 작업은 `dev` 브랜치를 기반으로 합니다.

## 1. 작업 시작

```bash
# dev 브랜치에서 feature 브랜치 생성
git checkout dev
git pull origin dev
git checkout -b feature/GO-{번호}-{설명}
```

- Linear 이슈 상태: `Backlog` → `In Progress`

## 2. 개발 및 커밋

```bash
# 커밋 메시지 형식
git commit -m "feat(GO-{번호}): 작업 내용 요약

- 상세 변경사항 1
- 상세 변경사항 2

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

## 3. PR 생성

```bash
git push -u origin feature/GO-{번호}-{설명}
gh pr create --base dev --title "feat(GO-{번호}): 제목" --body "..."
```

- Linear 이슈 상태: `In Progress` → `In Review`
- Linear 이슈에 PR 링크 첨부

## 4. 리뷰 및 머지

```bash
# 리뷰 확인
gh pr diff {PR번호}

# 머지 (브랜치 자동 삭제)
gh pr merge {PR번호} --merge --delete-branch
```

- Linear 이슈 상태: `In Review` → `Done`

---

## 브랜치 네이밍

| 타입 | 형식 | 예시 |
|-----|------|------|
| 기능 | `feature/GO-{번호}-{설명}` | `feature/GO-7-config` |
| 버그 | `fix/GO-{번호}-{설명}` | `fix/GO-15-login-error` |
| 리팩토링 | `refactor/GO-{번호}-{설명}` | `refactor/GO-20-auth` |

---

## 커밋 컨벤션

### 타입

| 타입 | 설명 |
|-----|------|
| `feat` | 새 기능 |
| `fix` | 버그 수정 |
| `refactor` | 리팩토링 |
| `docs` | 문서 |
| `chore` | 설정, 빌드 |
| `test` | 테스트 |

### 형식

```
{타입}(GO-{번호}): 제목 (한국어)

- 변경사항 1
- 변경사항 2

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
```

---

## 플로우 다이어그램

```
[Linear: Backlog]
       ↓
git checkout -b feature/GO-X-desc
       ↓
[Linear: In Progress]
       ↓
코드 작성 → git commit
       ↓
git push → gh pr create
       ↓
[Linear: In Review]
       ↓
gh pr merge --delete-branch
       ↓
[Linear: Done]
```
