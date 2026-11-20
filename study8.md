# study8. 구조화와 재사용

## ① 목표

한 파일에 모여 있던 서버를 설정, 저장소, HTTP, 인증으로 나누고, 개발과 운영이 같은 코드에서 환경 변수만 다르게 읽게 한다.

## ② 📖 이론

기초 예제는 `demo/step0N` 처럼 일부러 짧다. 게시판 전체를 그 방식으로 두면 회원가입을 고칠 때 SQL이 옆에 있고, 비밀 키가 핸들러 옆에 있다. 같은 일을 두 번 적게 되면 한쪽만 고쳐 상태가 어긋난다.

이 저장소의 나눔:

| 폴더 | 하는 일 | 하지 않는 일 |
| --- | --- | --- |
| `internal/config` | 환경 변수를 `Config`로 모음 | SQL, HTTP |
| `internal/store` | 테이블과 쿼리 | 상태 코드 |
| `internal/auth` | 해시와 JWT | 라우터 |
| `internal/httpapi` | 상태 코드, JSON, 미들웨어 | 비밀의 기본값을 코드에 숨기지 않음 |
| `internal/app` | 열기, 마이그레이션, 시드, 종료 | 개별 경로 |
| `project/web` | 브라우저 | DB |

`internal`은 다른 모듈이 import 하지 못한다. 앱 전용 코드라는 표시다.

개발과 운영의 차이는 파일이 아니라 `APP_ENV` 다.

- `development`: Gin 디버그 로그, OAuth 연습 모드를 기본으로 켤 수 있음
- `test`: 접근 로그를 끄고 `gin.TestMode`
- `production`: `gin.ReleaseMode`, `JWT_SECRET`에 `change-me` 이 있으면 시작 거부

같은 `Config.Validate`가 세 환경을 검사한다. 포트, DB, CORS, AI, 업로드 한도는 모두 환경 변수다. `.env`는 개발 PC에만 있고, 운영은 배포 시스템의 비밀 저장소에 같은 키를 넣는다.

중복을 줄인 예:

- 오류 JSON은 `writeError` 한 곳
- 글 JSON은 `postListJSON`, `postDetailJSON`
- 로그인 토큰 발급은 `issueToken`
- DB 자리 표시자는 `Rebind`가 SQLite의 `?` 와 PostgreSQL의 `$1` 을 바꾼다

비유하면 폴더는 주방의 도마다. 불 조절(config)과 재료 창고(store)와 접시 담기(httpapi)를 한 도마에서 하면 비밀번호가 샐러드에 섞인다.

## ③ 🛠️ 실기

서버 입구 `internal/app/run.go` 의 순서는 고정이다.

```go
cfg := config.Load()
if err := cfg.Validate(); err != nil {
	return err
}
st, err := store.Open(cfg.Driver, cfg.DSN)
if err := st.Migrate(context.Background()); err != nil {
	return err
}
srv := httpapi.New(cfg, st)
if cfg.Seed {
	if err := srv.Seed(context.Background()); err != nil {
		return err
	}
}
```

1. `.env`가 있으면 읽고, 이미 있는 환경 변수는 덮어쓰지 않는다.
2. 비밀 길이와 운영용 예시 값을 검사한다. 여기서 실패하면 DB를 열지 않는다.
3. 드라이버와 DSN으로 연결한다.
4. 테이블이 없으면 만든다. 있으면 그대로 둔다.
5. HTTP 서버는 설정과 저장소만 받는다. 핸들러가 `os.Getenv`를 직접 부르지 않는다.
6. `SEED=1`일 때만 연습 데이터를 넣는다. 테스트는 이 값을 주지 않아 빈 DB로 시작한다.

`httpapi.New`는 환경에 따라 Gin 모드를 정하고, `project/web` 과 `project/openapi.yaml` 이 있을 때만 화면과 문서 경로를 연다. 테스트는 패키지 디렉터리에서 돌기 때문에 그 파일이 없어도 API 테스트는 된다.

개발에서 쓰는 값과 운영에서 바꿀 값:

| 키 | 개발 | 운영에서 바꿀 것 |
| --- | --- | --- |
| `APP_ENV` | development | production |
| `HTTP_ADDR` | :8080 | 컨테이너가 노출할 주소 |
| `DB_DSN` | 로컬 파일 | 운영 DB 주소 |
| `JWT_SECRET` | 예시 문자열 | 긴 무작위 값 |
| `CORS_ORIGIN` | http://localhost:5173 | 실제 화면 출처 |
| `SEED` | 1 또는 비움 | 비움 |
| `AI_PROVIDER` | mock | 쓸 공급자만, 키는 비밀로 |

`go fmt ./...` 로 포맷을 맞추고, 안 쓰는 값은 지운다. 공통 함수가 한 곳에 있으면 상태 코드 문구를 바꿀 때도 그 함수만 고친다.

## ④ 🔗 연동 포인트

study10의 Docker 이미지는 같은 `Config`를 환경 변수로 받는다. 이미지 안에 `.env` 파일을 넣지 않는다. study16의 Compose는 `environment`로 개발용 값을 넘기고, 운영 워크플로는 저장소 밖의 비밀을 같은 키에 주입한다. DB 종류를 바꿀 때도 핸들러 서명은 유지되고 `DB_DRIVER`만 바뀐다.

## ⑤ 오류 노트

| 증상 | 원인 | 해결 |
| --- | --- | --- |
| 테스트가 개발 DB 파일을 지움 | 테스트가 `board.db`를 열음 | 테스트는 `t.TempDir`의 DSN을 쓴다 |
| 운영에서 디버그 라우트 목록이 찍힘 | `APP_ENV`가 development | production으로 둔다 |
| 설정을 핸들러마다 읽음 | `os.Getenv`가 흩어짐 | `Config` 필드를 넘긴다 |
| import cycle | `store`가 `httpapi`를 import | 의존은 httpapi → store 한 방향 |
| `.env`가 운영 컨테이너에 없음 | 로컬 파일에만 의존 | 배포 환경 변수로 같은 키를 넣는다 |

## ⑥ 직접 해보기

1. `APP_ENV=test` 로 서버를 잠깐 켜고 접근 로그가 없는지 본다. 끝나면 development로 되돌린다.
2. `JWT_SECRET`을 10자로 두고 시작이 거부되는지 확인한 뒤 긴 값으로 되돌린다.
3. `internal/httpapi`에서 `os.Getenv`를 검색해 핸들러가 환경을 직접 읽지 않는지 본다.

## ⑦ ☐ 체크리스트

- [ ] 설정, 저장, 인증, HTTP, 입구가 폴더로 나뉘어 있다
- [ ] 핸들러가 `Config`와 `Store`만 받는다
- [ ] 개발, 테스트, 운영이 `APP_ENV`로 갈린다
- [ ] 비밀과 연습 데이터 주입은 환경 변수다
- [ ] 오류 JSON과 글 JSON을 한 곳에서 만든다
- [ ] `go test ./...` 가 개발용 `board.db`에 의존하지 않는다

## ⑧ 다음 단계

구조가 갈라졌으면 study9에서 그 경계를 `httptest`와 로그로 확인한다.
