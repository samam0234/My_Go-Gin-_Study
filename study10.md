# study10. 배포와 기초 정리

## ① 목표

서버를 하나의 실행 파일로 빌드하고, 멀티 스테이지 Docker 이미지로 줄인 뒤, study0부터 여기까지 만든 게시판 API를 배포 환경의 환경 변수와 DB에 연결하는 순서를 정리한다.

## ② 📖 이론

`go build`는 이 폴더의 `main`을 실행 파일로 만든다. 그 파일은 Go가 설치되지 않은 기계에서도 같은 OS와 아키텍처면 실행된다. 소스와 `go` 명령은 빌드 기계에만 있으면 된다.

Docker 멀티 스테이지는 빌드 도구가 있는 이미지에서 컴파일하고, 마지막 이미지에는 실행 파일과 화면 파일만 복사한다. 그래서 이미지가 작아지고 컴파일러가 운영 환경에 남지 않는다. `CGO_ENABLED=0` 이면 SQLite도 순수 Go 드라이버로 붙어 공유 라이브러리가 필요 없다.

도메인은 사람이 외우는 이름이고, DNS가 그 이름을 서버 IP로 바꾼다. HTTPS는 TLS로 그 연결을 암호화한다. 이 학습 서버는 컨테이너 안에서 HTTP `:8080` 을 연다. 인증서와 도메인 연결은 앞단의 역방향 프록시나 호스팅의 부하 분산이 맡는다. 앱 코드가 인증서 파일을 직접 읽지 않아도 된다.

비유하면 빌드는 도시락을 싸는 일이고, 첫 번째 스테이지는 주방이며, 마지막 스테이지는 도시락통이다. 칼과 도마는 통에 넣지 않는다.

## ③ 🛠️ 실기

로컬 빌드:

```powershell
go test ./...
go build -trimpath -ldflags="-s -w" -o bin/api.exe .
.\bin\api.exe
```

`-s -w` 는 디버그 정보를 빼 파일을 줄인다. 문제를 소스와 맞춰 볼 때는 이 플래그 없이 빌드한다.

`project/Dockerfile`

```dockerfile
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates \
    && adduser -D -u 65532 app \
    && mkdir -p /data \
    && chown app:app /data
WORKDIR /app
COPY --from=build /out/api /app/api
COPY --from=build /src/project/web /app/web
COPY --from=build /src/project/openapi.yaml /app/openapi.yaml
ENV HTTP_ADDR=:8080 \
    WEB_DIR=/app/web \
    OPENAPI_PATH=/app/openapi.yaml \
    DB_DSN=file:/data/board.db?_pragma=busy_timeout(5000)
USER app
EXPOSE 8080
ENTRYPOINT ["/app/api"]
```

1. 첫 스테이지 이름은 `build` 다. Go 도구가 있다.
2. 의존성 파일만 먼저 복사하면, 소스만 바뀔 때 다운로드 층을 다시 쓰지 않는다.
3. `CGO_ENABLED=0` 과 `GOOS=linux` 로 리눅스용 정적 바이너리를 만든다.
4. 마지막 스테이지는 Alpine이다. 인증서 묶음만 추가한다. HTTPS로 바깥 모델을 부를 때 필요하다.
5. 전용 사용자 `app`과 쓰기 가능한 `/data` 를 만든다. 루트로 실행하지 않는다.
6. 바이너리, 화면, OpenAPI만 복사한다. `.env` 와 소스 트리는 복사하지 않는다.
7. SQLite 파일은 `/data` 다. 컨테이너를 지워도 남기려면 볼륨을 마운트한다.
8. `USER app` 이후에는 그 사용자가 쓸 수 있는 경로만 써야 한다.

실행:

```powershell
docker build -f project/Dockerfile -t gin-study:local .
docker run --rm -p 8080:8080 -e SEED=1 -e JWT_SECRET=dev-only-change-me-please-32b -e APP_ENV=development gin-study:local
```

기초 구간에서 만든 것:

- study0 모듈과 환경 변수
- study1 첫 응답과 템플릿 이스케이프
- study2 패키지
- study3 구조체와 JSON
- study4 경로와 404
- study5 검증과 중복 제출
- study6 sqlx CRUD와 스펙 v1
- study7 JWT, 해시, CORS
- study8 폴더와 `APP_ENV`
- study9 httptest

## ④ 🔗 연동 포인트

배포 환경은 이미지에 없는 값을 실행 시 넣는다.

| 값 | 연결 |
| --- | --- |
| `JWT_SECRET` | 서명. 예시 값이면 `APP_ENV=production` 에서 시작 실패 |
| `DB_DRIVER`, `DB_DSN` | SQLite 파일이면 볼륨, PostgreSQL이면 study11의 DSN |
| `CORS_ORIGIN` | 실제 화면 출처 |
| `SEED` | 운영에서는 비운다 |
| `WEB_DIR`, `OPENAPI_PATH` | 이미지 안의 `/app/web`, `/app/openapi.yaml` |

배포 후 점검:

1. `GET /health` 가 200 `{"status":"ok"}` 인가
2. `GET /api-docs/openapi.yaml` 이 열리는가
3. 회원가입과 로그인이 201, 200인가
4. 토큰 없이 글 등록이 401인가
5. 목록 `meta.totalPages` 가 숫자인가
6. 응답에 비밀번호 해시가 없는가
7. 컨테이너를 지웠다 켜도 볼륨의 글이 남는가
8. 로그에 요청 본문이 없는가

도메인을 붙인 뒤에는 브라우저 주소가 `https://이름` 이고, 앱은 여전히 컨테이너 포트로 HTTP를 받는다. 프록시가 `X-Forwarded-For` 를 붙여도 이 서버는 신뢰 프록시를 비워 두어 그 헤더를 클라이언트 IP로 믿지 않는다.

## ⑤ 오류 노트

| 증상 | 원인 | 해결 |
| --- | --- | --- |
| 이미지가 매우 큼 | 빌드 스테이지를 그대로 실행 | 마지막 스테이지에 바이너리만 복사 |
| 컨테이너에서 화면이 JSON만 보임 | `WEB_DIR`이 비었거나 복사가 빠짐 | Dockerfile의 `COPY`와 환경 변수 |
| `/data`에 쓸 수 없음 | root로 만든 볼륨을 app이 못 씀 | 이미지에서 `chown app` 한 디렉터리를 마운트 |
| 운영이 바로 죽음 | 예시 JWT 비밀 | 비밀을 바꾸고 `APP_ENV=production` |
| DB가 재시작마다 사라짐 | 파일 시스템이 컨테이너 안 | 볼륨 `boarddata:/data` |
| HTTPS 인증서 오류를 앱이 해결하려 함 | 앱이 직접 443을 염 | 프록시가 TLS를 끝내게 둔다 |

## ⑥ 직접 해보기

1. `go build -o bin/api.exe .` 후 Go를 PATH에서 빼고도 실행 파일이 도는지 본다.
2. `docker build -f project/Dockerfile -t gin-study:local .` 로 이미지 크기를 `docker images`에서 확인한다.
3. 컨테이너로 `/health` 를 호출하고, `SEED` 없이 다시 띄우면 연습 계정이 생기지 않는지 본다.

## ⑦ ☐ 체크리스트

- [ ] Docker 멀티 스테이지로 작은 이미지
- [ ] `go test ./...` 후에 빌드한다
- [ ] 최종 이미지는 비루트 사용자다
- [ ] `.env`를 이미지에 넣지 않는다
- [ ] DB 파일은 볼륨에 있다
- [ ] 배포 후 건강 체크, 로그인, 401, 목록 메타를 확인했다
- [ ] 도메인과 HTTPS는 앞단 프록시의 일이다

## ⑧ 다음 단계

기초만으로 게시판을 처음부터 조립하는 순서는 studyMidpoint에 모아 둔다.
