# study9. 테스트와 디버깅

## ① 목표

핸들러를 `httptest`로 호출하고, 실패하면 터미널 로그의 요청 ID로 원인을 찾으며, 이 서버에서 자주 만나는 오류를 상태 코드와 연결한다.

## ② 📖 이론

단위 테스트는 함수 하나를 호출한다. `divide(1, 0)` 이 에러를 돌려주는지, `auth.Parse`가 서명을 거절하는지가 그 예다.

요청 테스트는 서버 포트를 열지 않는다. `httptest.NewRequest`로 요청을 만들고 `httptest.NewRecorder`에 응답을 받는다. Gin 엔진이 `http.Handler` 이므로 `ServeHTTP`만 부르면 된다. `gin.SetMode(gin.TestMode)` 는 디버그 출력을 줄인다.

DB가 필요한 테스트는 임시 폴더의 SQLite 파일을 연다. 외부 API는 테스트에서 호출하지 않는다. AI는 `AI_PROVIDER=mock` 이고, OAuth는 `OAUTH_MOCK=true` 이며, Mongo 예제는 `MONGO_URL`이 없으면 연결을 건너뛴다. 이것이 가짜(mock)로 바꾸는 방법이다. 인터페이스를 비우기보다, 설정으로 네트워크가 없는 구현을 고른다.

로그는 접근 한 줄과 서버 오류 한 줄이다. 둘 다 `rid=` 요청 ID를 붙인다. 응답 JSON의 `error.requestId` 와 같은 값이다. 본문과 비밀번호는 로그에 넣지 않는다.

비유하면 테스트는 손님이 오기 전에 창구 대본을 읽어 보는 일이고, 요청 ID는 영수증 번호다. 손님이 "방금 거절당했다"고 하면 번호로 로그 한 줄을 찾는다.

## ③ 🛠️ 실기

```powershell
go test ./...
go test ./internal/httpapi -count=1
```

`internal/httpapi/server_test.go` 가 요청 테스트다. 도우미 `newServer`는 임시 SQLite와 테스트 비밀로 `httpapi.New`를 만든다. `do`는 메서드, 경로, 토큰, JSON 본문을 넣어 기록기를 돌려준다.

시나리오 `TestBoardFlow`가 확인하는 것:

1. `GET /health` 가 200 `{"status":"ok"}`
2. 회원가입 201, 본문에 `password123`과 `$2` 해시가 없음
3. 중복 이메일은 409 `EMAIL_TAKEN`
4. 없는 메일과 틀린 비밀번호가 둘 다 401이고 코드가 같음
5. 토큰 없이 `GET /api/auth/me` 는 401, 있으면 200
6. 로그아웃 204
7. 빈 제목은 400 `VALIDATION_ERROR`
8. 글 생성 201, `Location: /api/posts/1`
9. 같은 Idempotency-Key 는 본문이 같고 `Idempotency-Replayed: true`
10. 목록 캐시가 처음 `MISS`, 같은 쿼리는 `HIT`
11. 잘못된 `sort`와 글자 id는 400
12. `limit=100` 은 meta에서 50
13. 다른 사용자의 수정은 403, 작성자 수정은 200
14. 댓글 201, 관리자 삭제 204
15. 글을 지우면 댓글 삭제는 404
16. 없는 경로는 404 `NOT_FOUND`

`TestDocumentAIAndMCP`는 mock 모델과 MCP 토큰을 본다. `TestUploadPaymentOAuthQuota`는 한도를 1로 낮춰 429 `AI_QUOTA_EXCEEDED` 를 본다. `TestRateLimitAndSocket`은 분당 2회로 429를 보고, 웹소켓으로 `post.created` 를 받는다.

디버그 순서:

1. 테스트 이름과 상태 코드를 읽는다.
2. `rec.Body.String()` 에 찍힌 `requestId`를 복사한다.
3. 서버를 직접 띄운 경우 터미널에서 `rid=` 같은 값을 찾는다.
4. 500이면 로그의 `err=` 만 보고, 그 문장을 클라이언트에 붙여 넣지 않는다.
5. `dlv`를 쓸 때는 핸들러 첫 줄에 멈추고 `c.Request.URL.Path` 와 컨텍스트의 사용자를 본다. 확장이 없으면 `log.Printf` 한 줄을 넣고 테스트를 돌린 뒤 지운다.

자주 나는 오류:

| # | 증상 | 원인 | 해결 |
| --- | --- | --- | --- |
| 1 | `bind: address already in use` | 포트 점유 | 프로세스를 끄거나 `HTTP_ADDR` 변경 |
| 2 | 401인데 토큰을 넣었다고 생각 | `Bearer` 접두어 누락 | 헤더 문자열 확인 |
| 3 | 400 `BAD_REQUEST` | JSON 문법, 빈 본문, 글자 id | 본문과 경로를 분리해 본다 |
| 4 | 400 `VALIDATION_ERROR` | 태그 실패 | `fields`의 키를 고친다 |
| 5 | 403 | 작성자가 아님 | 로그인 계정과 `author.id` 비교 |
| 6 | 404 | 없는 글, 남의 문서, 없는 경로 | `error.message`와 경로 |
| 7 | 409 `EMAIL_TAKEN` | 같은 이메일 | 다른 메일 또는 로그인 |
| 8 | 413 | 본문 또는 업로드 초과 | 크기 제한을 본다 |
| 9 | 429 | 호출 제한 또는 AI 한도 | `RATE_LIMIT_PER_MIN`, `AI_DAILY_TOKEN_LIMIT` |
| 10 | 500 `INTERNAL` | DB, 예상 못 한 오류 | `rid`로 서버 로그만 확인 |

## ④ 🔗 연동 포인트

테스트의 mock은 화면과 배포에서도 같은 스위치다. `AI_PROVIDER=mock` 이면 모델 서버가 없어도 `/api/ai/chat` 이 200이다. `OAUTH_MOCK=1` 이면 구글에 나가지 않고 연습 계정을 만든다. DB를 통째로 가짜 객체로 바꿀 필요는 없다. 임시 SQLite가 쿼리까지 검사하기 때문이다. 외부 HTTP가 꼭 필요하면 `exchangeGoogle` 함수 값을 테스트에서 바꿀 수 있게 `Server`에 필드가 있다.

## ⑤ 오류 노트

| 증상 | 원인 | 해결 |
| --- | --- | --- |
| 테스트가 서로 영향을 줌 | 같은 `board.db` | 테스트마다 임시 파일 |
| `go test`가 포트를 점유 | 테스트 안에서 `Run` | `ServeHTTP`만 사용 |
| 캐시 때문에 두 번째 단언이 이상함 | 같은 쿼리 | `X-Cache`를 기대하거나 쿼리를 바꾼다 |
| 로그에 비밀번호 | 요청 본문을 통째로 출력 | 상태 코드와 경로만 남긴다 |
| 요청 ID가 없음 | 미들웨어 순서 | `requestID`를 라우터보다 앞선다 |

## ⑥ 직접 해보기

1. `go test ./internal/httpapi -run TestBoardFlow -v` 로 시나리오 이름을 본다.
2. 일부러 기대를 400에서 200으로 바꿔 실패 메시지를 읽고 되돌린다.
3. 서버를 켜고 없는 경로를 호출한 뒤, 응답 `requestId`가 터미널 `rid`와 같은지 본다.

## ⑦ ☐ 체크리스트

- [ ] httptest로 핸들러 테스트
- [ ] 테스트 DB는 임시 SQLite다
- [ ] AI와 OAuth는 네트워크 없이 mock으로 돈다
- [ ] 401, 403, 404, 409, 400을 테스트가 구분한다
- [ ] 500의 상세는 로그에만 있고 응답 코드는 `INTERNAL` 이다
- [ ] 요청 ID로 응답과 로그를 연결할 수 있다
- [ ] `go test ./...` 가 통과한다

## ⑧ 다음 단계

테스트로 확인한 서버를 study10에서 실행 파일과 작은 이미지로 만든다.
