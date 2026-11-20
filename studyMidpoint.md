# studyMidpoint. 게시판 서비스

## ① 목표

study0부터 study10까지로 Gin 서버 하나에 회원가입, 로그인, 글과 댓글 CRUD를 만들고 로컬에서 한 번 실행한다.

## 기능 목록

- `GET /health`
- 회원가입, 로그인, 내 정보, 로그아웃
- 글 목록(검색, 페이지, 정렬), 상세, 등록, 수정, 삭제
- 댓글 목록, 등록, 삭제
- 작성자 또는 관리자만 수정과 삭제
- SQLite, JWT, bcrypt
- 브라우저 화면 `/` 에서 위 흐름을 클릭

결제, 문서 질문, 웹소켓, Docker 자동 배포는 study11 이후와 studyFinal에서 더한다. 여기의 완료 기준은 스펙 v1의 회원과 글이다.

## 폴더 구조

```text
main.go
internal/config/config.go
internal/auth/password.go
internal/auth/token.go
internal/store/store.go
internal/store/ddl.go
internal/httpapi/server.go
internal/httpapi/auth.go
internal/httpapi/posts.go
project/web/index.html
project/web/app.js
.env.example
```

## 단계별 구현 순서

1. `go mod init` 과 Gin, sqlx, sqlite, jwt, bcrypt, godotenv를 가져온다. study0.
2. `config.Load`로 포트, DSN, JWT 비밀, CORS를 읽는다. study8.
3. `store.Open`과 `Migrate`로 `users`, `posts`, `comments` 를 만든다. study6.
4. `GET /health` 와 `NoRoute` JSON을 둔다. study4.
5. 회원가입과 로그인을 만들고 응답에서 해시를 뺀다. study7.
6. `requireAuth`를 글 등록, 수정, 삭제, 댓글 등록에 건다.
7. 목록은 `data`와 `meta`, 등록은 201과 `Location`, 삭제는 204. study6.
8. 검증 실패는 400 `VALIDATION_ERROR`. study5.
9. `project/web`에서 로그인과 목록, 등록, 상세, 댓글을 `fetch`와 `textContent`로 잇는다.
10. `go test ./internal/httpapi -run TestBoardFlow` 로 위 순서를 고정한다.
11. `SEED=1` 로 `go run .` 하고 http://localhost:8080 에서 연습 계정으로 한 번 클릭한다.

## DB, API 연동 구성도

```text
브라우저 /
  |  fetch /api , Authorization Bearer
  v
Gin :8080
  |  config.Load  ←  .env / 환경 변수
  |  requireAuth  ←  JWT_SECRET
  v
sqlx + SQLite (DB_DSN)
  users / posts / comments
```

화면과 API는 한 프로세스다. 다른 포트의 화면을 붙이면 `CORS_ORIGIN`만 그 출처로 바꾼다. API 경로는 스펙 v1과 같다.

## 1차 배포 체크리스트

- [ ] `go test ./...` 통과
- [ ] `.env`가 Git에 없다
- [ ] `JWT_SECRET`이 예시 문자열이 아니다
- [ ] `SEED`는 운영에서 비운다
- [ ] `GET /health` 200
- [ ] 연습 계정 로그인 후 글 등록 201
- [ ] 다른 계정 수정 403
- [ ] 토큰 삭제 후 등록 401
- [ ] SQLite 파일을 볼륨 또는 백업 경로에 둔다
- [ ] 응답에 비밀번호 해시가 없다

## 회고

기초 구간의 핵심은 경로와 상태 코드를 스펙에 고정한 것이다. Gin의 편리는 바인딩과 미들웨어이고, 데이터와 권한은 우리가 `store`와 `requireAuth`에 명확히 적어야 한다. 파일 하나만 더 커지기 전에 폴더를 나눈 덕분에 테스트가 임시 DB만으로 같은 핸들러를 호출할 수 있다.

막힌 지점은 주로 401과 403을 섞는 일, nil 슬라이스가 `null`이 되는 일, Windows 경로의 역슬래시다. 오류 노트에 적어 두고 study11에서 DB만 바꿀 때 핸들러를 다시 열지 않아도 되게 둔다.

## ② 📖 이론

작은 서비스는 새 프레임워크가 아니라 지금까지의 핸들러를 한 프로세스에 두는 일이다. 사용자는 화면만 보고, 화면은 스펙의 JSON만 본다.

## ③ 🛠️ 실기

위 구현 순서대로 저장소가 이미 연결되어 있다. 확인 명령:

```powershell
copy .env.example .env
$env:SEED = "1"
go test ./internal/httpapi -run TestBoardFlow
go run .
```

연습 계정 `mina@example.com` / `password123` 으로 로그인하고 글을 하나 남긴다.

## ④ 🔗 연동 포인트

이 서비스의 `posts` 테이블과 JWT가 study11의 다른 DB, study13의 업로드, study18의 문서 질문과 같은 사용자 id를 쓴다. 경로를 바꾸지 않으면 화면 코드를 유지한 채 뒤만 바꿀 수 있다.

## ⑤ 오류 노트

| 증상 | 원인 | 해결 |
| --- | --- | --- |
| 화면은 뜨는데 목록이 실패 | API 포트가 다름 | 화면과 API를 이 서버 하나에 둔다 |
| 시드 계정이 없음 | `SEED`가 비어 있음 | 개발에서만 `SEED=1` |
| 테스트가 시드 글에 의존 | 운영 DB를 테스트가 염 | 임시 파일만 사용 |

## ⑥ 직접 해보기

1. 시드 없이 회원가입부터 댓글까지 브라우저로 진행한다.
2. 두 계정을 만들어 서로의 글 수정을 403으로 확인한다.
3. `TestBoardFlow` 를 읽고 화면에서 한 단계와 테스트 한 줄을 대응시킨다.

## ⑦ ☐ 체크리스트

- [ ] 기능 목록의 회원, 글, 댓글이 스펙 경로와 같다
- [ ] 폴더가 config, store, auth, httpapi, web로 나뉜다
- [ ] 구성도처럼 브라우저가 Gin만 호출한다
- [ ] 1차 배포 체크리스트를 로컬에서 한 번 수행했다
- [ ] 회고에 막힌 오류를 한 줄 적었다

## ⑧ 다음 단계

같은 서비스를 유지한 채 study11에서 DB 드라이버를 바꿔 본다.
