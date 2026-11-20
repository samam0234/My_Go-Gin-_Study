# study14. API 설계

## ① 목표

게시판 공통 API 스펙 v1을 Gin 경로와 OpenAPI 파일로 고정하고, 페이지와 오류 본문을 그 규칙대로 유지한다.

## ② 📖 이론

REST에서 경로는 명사다. `/api/posts`, `/api/posts/{id}`, `/api/posts/{id}/comments`. 동사 경로 `/api/getPosts` 는 쓰지 않는다. 수정은 PUT 여러 개가 아니라 PATCH 하나다. 삭제는 본문 없는 204다.

성공 본문은 한 건과 목록이 다르다.

```json
{ "data": { "id": 1, "title": "첫 글", "author": { "id": 3, "nickname": "mina" }, "commentCount": 2, "createdAt": "2026-11-01T09:00:00Z" } }
```

```json
{ "data": [], "meta": { "page": 1, "limit": 10, "total": 0, "totalPages": 0, "q": "" } }
```

필드 이름은 camelCase다. 시각은 UTC `RFC3339` 다. 비밀번호는 어떤 `data`에도 없다.

오류는 항상 `error` 객체다.

| 상태 | code | 언제 |
| --- | --- | --- |
| 400 | VALIDATION_ERROR | 필드 검증. `fields` 포함 |
| 400 | BAD_REQUEST | JSON 문법, 글자 id |
| 401 | UNAUTHORIZED | 토큰 없음·위조·만료 |
| 401 | INVALID_CREDENTIALS | 로그인 실패 |
| 403 | FORBIDDEN | 남의 글 수정·삭제 |
| 404 | NOT_FOUND | 없는 자원, 없는 경로 |
| 409 | EMAIL_TAKEN | 중복 이메일 |
| 413 | PAYLOAD_TOO_LARGE | 크기 초과 |
| 429 | RATE_LIMITED | 분당 호출 초과. 0이면 끔 |
| 500 | INTERNAL | 서버 오류. 원인은 로그 |

버전은 URL에 `/v1`을 넣지 않는다. 스펙이 `/api`를 현재 판으로 쓰기 때문이다. 나중에 깨지는 변경이 생기면 `/api`를 유지한 채 응답 필드를 더하는 쪽을 먼저 고르고, 필드를 지우거나 이름을 바꿀 때만 다음 경로를 논의한다.

문서 주소는 `/api-docs` 와 `/api-docs/openapi.yaml` 이다. `project/openapi.yaml`이 핵심 경로를 담는다. 결제와 문서 질문 같은 확장은 라우터에는 있지만 스펙 표의 필수 경로와 구분한다.

비유하면 스펙은 주문서 양식이다. 주방이 Gin이어도 주문서의 칸 이름은 바꾸지 않는다.

## ③ 🛠️ 실기

OpenAPI의 공통 오류는 `components.responses.Error` 한 곳이다. 보안 스키마 이름은 `bearer` 다.

```yaml
bearer:
  type: http
  scheme: bearer
  bearerFormat: JWT
```

서버가 맞추는 헤더:

- 요청과 응답 `Content-Type: application/json; charset=utf-8` (Gin `c.JSON`)
- 응답 `X-Request-Id`. 요청에 있으면 80자 안에서 그대로 돌려준다.
- 등록 성공 `Location: /api/posts/{id}`

페이지 계산은 `internal/httpapi/posts.go` 에 있다. `total`이 0이면 `totalPages`도 0이다. 그 외에는 `(total + limit - 1) / limit` 이다.

`RATE_LIMIT_PER_MIN`이 양수면 IP마다 분당 그 횟수를 넘길 때 429다. `/health`와 `OPTIONS`는 세지 않는다. 학습 기본값은 0이라 꺼져 있다.

문서 확인:

```powershell
go run .
# 다른 터미널
curl http://localhost:8080/api-docs/openapi.yaml
```

브라우저의 `/api-docs` 는 파일 링크와 핵심 경로 목록이다.

## ④ 🔗 연동 포인트

study12 화면은 이 파일의 경로와 `data`/`meta`/`error`만 안다. study6의 SQL이 바뀌어도 YAML의 칸이 같으면 화면을 고치지 않는다. 요청 ID는 study9 로그와 같은 값이다. 배포 환경의 `CORS_ORIGIN`은 문서에 적힌 개발 출처 `http://localhost:5173` 대신 운영 출처를 넣는다.

## ⑤ 오류 노트

| 증상 | 원인 | 해결 |
| --- | --- | --- |
| 목록이 배열만 오고 meta가 없음 | 핸들러가 스펙 이전 모양 | `listBytes`로 `data`와 `meta`를 같이 보낸다 |
| 오류가 최상위에 `message`만 있음 | `writeError`를 안 거침 | `error.code`, `error.message`, `error.requestId` |
| 삭제 응답에 `{}` | `c.JSON(204, ...)` | `c.Status(204)` |
| OpenAPI와 라우터가 다름 | 경로를 코드만 고침 | `project/openapi.yaml`의 path를 같이 고친다 |
| page가 0부터라고 가정 | 다른 스택의 습관 | 이 스펙은 1부터 |

## ⑥ 직접 해보기

1. `/api-docs/openapi.yaml`에서 `/api/posts`의 쿼리 이름과 코드의 `listQuery` 태그를 대조한다.
2. 없는 경로의 JSON에 `requestId`가 있는지 보고, 같은 값을 요청 헤더로 넣어 그대로 돌아오는지 본다.
3. `limit=100`의 meta.limit이 50인지 확인한다.

## ⑦ ☐ 체크리스트

- [ ] 경로는 복수 명사와 `/api` 다
- [ ] 성공은 `data`, 목록은 `meta`를 포함한다
- [ ] 오류 표의 code를 코드가 그대로 쓴다
- [ ] 페이지는 1부터이고 limit 상한은 50이다
- [ ] OpenAPI가 `/api-docs/openapi.yaml` 로 제공된다
- [ ] Bearer 보안 스키마가 문서에 있다
- [ ] 등록은 `Location` 헤더를 준다

## ⑧ 다음 단계

문서와 동작이 맞으면 study15에서 같은 목록 응답을 짧게 캐시하고 메일을 백그라운드에서 비운다.
