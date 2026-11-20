# study7. 인증과 보안 기초

## ① 목표

회원가입과 로그인으로 JWT를 발급하고, 작성자만 글을 고치게 하며, 비밀번호 해시, CORS, XSS, CSRF를 Gin 서버 기준으로 정리한다.

## ② 📖 이론

인증은 "누구인지"이고 인가는 "그 사람이 이 글을 고칠 수 있는지"다.

- 401 `UNAUTHORIZED`: 토큰이 없거나 서명이 이상하거나 만료됐다.
- 403 `FORBIDDEN`: 누구인지는 아는데 그 글의 작성자가 아니다.
- 409 `EMAIL_TAKEN`: 이메일이 이미 있다.
- 401 `INVALID_CREDENTIALS`: 로그인 실패. 메일이 없는 것과 비밀번호가 틀린 것을 같은 문장으로 답한다.

토큰은 `Authorization: Bearer <accessToken>` 이다. 학습용 수명은 `JWT_TTL` 이고 기본 168시간(7일)이다. 스펙이 말하는 기본 1시간도 `JWT_TTL=1h` 로 줄일 수 있다. 서명 비밀은 `JWT_SECRET` 이고 16자 미만이면 서버가 시작하지 않는다. `APP_ENV=production` 인데 비밀에 `change-me` 이 들어 있으면 시작을 거부한다.

비밀번호는 bcrypt로만 저장한다. 로그와 JSON에 원문이나 `$2` 해시를 넣지 않는다. 없는 이메일도 가짜 해시와 비교해, 응답 속도로 가입 여부를 짐작하기 어렵게 한다.

CORS는 브라우저가 다른 출처의 응답을 읽을지 정하는 규칙이다. `CORS_ORIGIN`에 적힌 출처만 허용하고 `*` 는 버린다. 이 API는 쿠키가 아니라 Bearer 헤더라서 `Allow-Credentials` 는 끈다. 쿠키 세션을 나중에 쓰면 `SameSite`와 CSRF 토큰이 필요해진다. Bearer만 쓰는 동안 다른 사이트의 폼이 우리 글을 지울 수 없다. 브라우저가 다른 출처의 비밀 헤더를 자동으로 붙이지 않기 때문이다.

XSS는 글 본문을 HTML로 해석할 때 생긴다. 화면은 `textContent`만 쓴다. 템플릿도 `html/template`가 이스케이프한다. 업로드 파일은 다운로드 헤더와 `nosniff` 로 브라우저가 내용으로 실행하지 않게 한다.

역할 `admin` 은 남의 댓글과 글을 지울 수 있다. 일반 가입은 `user` 다. 관리자 승격은 공개 API가 아니라 저장소의 `SetRole` 이다.

비유하면 JWT는 입장 팔찌다. 팔찌가 없으면 401, 팔찌는 있는데 남의 사물함이면 403이다. 비밀번호 원문은 팔찌에 적지 않는다.

## ③ 🛠️ 실기

해시와 토큰만 보려면:

```powershell
go test ./demo/step07
```

`demo/step07/main.go` 의 `run`은 `password123`을 해시하고, 맞은 비밀번호와 틀린 비밀번호를 구분한 뒤, 사용자 id 7로 서명한 토큰을 다시 읽어 id가 7인지 본다. 비밀이 다르면 `Parse`가 실패한다.

서버 흐름은 `internal/httpapi/auth.go` 다.

1. `POST /api/auth/signup` 본문은 `email`, `password`, `nickname`. 성공은 201 `{data:{id,email,nickname}}`. 토큰은 주지 않는다. 화면이 이어서 로그인한다.
2. 이메일이 있으면 409 `EMAIL_TAKEN`.
3. 가입이 되면 환영 메일을 대기열에 넣는다. 실제 발송은 study15의 워커다.
4. `POST /api/auth/login` 성공은 200 `{data:{user, accessToken}}`.
5. `GET /api/auth/me` 는 Bearer가 유효할 때만 200.
6. `POST /api/auth/logout` 은 인증된 요청에 204다. 서버는 토큰 목록을 지우지 않는다. 화면이 `localStorage`의 `accessToken`을 지운다. 스펙이 말하는 쿠키 삭제는 쿠키 방식일 때의 일이다.
7. `requireAuth`는 `Bearer ` 접두어를 확인하고, 서명을 검사하고, 그 id의 사용자가 아직 있을 때만 `authUser`를 컨텍스트에 넣는다.

글 수정과 삭제는 `canEdit`가 작성자 id 또는 `admin` 만 통과시킨다. 다른 사용자는 403이다.

화면은 로그인 응답의 `accessToken`을 `localStorage`에 두고 다음 `fetch`의 `Authorization` 헤더에 붙인다. 같은 출처 화면이라 CORS 사전 요청 없이도 동작한다. 5173 포트의 다른 화면을 붙이면 서버의 `CORS_ORIGIN`이 그 출처여야 한다.

연습 계정은 `SEED=1`일 때 `mina@example.com` / `password123` 이다.

## ④ 🔗 연동 포인트

프론트와 백엔드는 토큰 문자열만 공유한다. 서버는 세션 저장소 없이 서명으로 사용자를 복원한다. 화면이 토큰을 지우면 다음 요청은 401이 되고 목록 조회처럼 공개 API만 남는다. study13의 OAuth도 마지막에는 같은 `accessToken` JSON을 준다. study18의 AI 경로는 이 미들웨어 뒤에 있어 키가 없는 브라우저는 모델을 부르지 못한다.

## ⑤ 오류 노트

| 증상 | 원인 | 해결 |
| --- | --- | --- |
| 맞는 비밀번호인데 401 | 이메일의 대소문자 또는 앞뒤 공백 | 서버가 이메일을 소문자로 저장한다. 입력을 맞춘다 |
| 토큰을 넣었는데 401 | `Bearer` 뒤에 공백이 없거나 토큰이 잘림 | `Authorization: Bearer` 한 칸 |
| 비밀이 짧아 서버가 죽음 | `JWT_SECRET` 16자 미만 | `.env`를 늘린다 |
| 로그인 실패 문구가 메일 존재 여부를 말함 | 404와 401을 나눔 | 둘 다 `INVALID_CREDENTIALS` |
| 글이 HTML로 실행됨 | `innerHTML` | `textContent` |
| 브라우저 CORS 오류 | 출처가 `CORS_ORIGIN`에 없음 | 환경 변수에 화면 출처를 적는다. `*` 는 쓰지 않는다 |
| 운영에서 예시 비밀로 기동 | `change-me` 검사 | 운영 비밀로 교체 |

## ⑥ 직접 해보기

1. 토큰 없이 `POST /api/posts` 가 401인지 본다.
2. 다른 계정으로 그 글을 `PATCH` 해 403인지 본다.
3. 없는 메일과 틀린 비밀번호의 `error.code` 가 같은지 비교한다.

## ⑦ ☐ 체크리스트

- [ ] JWT 미들웨어
- [ ] 회원가입 201, 중복 이메일 409 `EMAIL_TAKEN`
- [ ] 로그인 200에 `accessToken`, 실패는 401 `INVALID_CREDENTIALS`
- [ ] 비밀번호는 bcrypt 해시만 저장하고 응답에 없다
- [ ] 작성자가 아니면 403, 토큰이 없으면 401
- [ ] `CORS_ORIGIN`만 허용하고 `*` 는 버린다
- [ ] 화면은 토큰을 헤더로 보내고, 로그아웃 때 지운다
- [ ] `go test ./demo/step07` 와 인증 시나리오가 통과한다

## ⑧ 다음 단계

인증과 DB가 한 서버에 모였으면 study8에서 폴더와 환경 설정을 역할별로 정리한다.
