# study0. 환경 설정

## ① 목표

Go 1.27과 Gin 1.12로 이 저장소의 모듈을 준비하고, 비밀값은 `.env`에만 둔 채 `go run .` 이 서버를 띄우게 한다.

## ② 📖 이론

Gin은 Go 표준 라이브러리 `net/http` 위에 얹은 웹 라우터다. 요청이 들어오면 엔진이 경로를 고르고, 미들웨어를 지나 핸들러 함수를 호출한다. Go 프로그램은 컴파일되면 실행 파일 하나가 된다. 서버에 올릴 때도 그 파일과 환경 변수면 충분하다.

이 단계에서 쓸 말:

| 용어 | 뜻 |
| --- | --- |
| 모듈 | `go.mod`가 있는 폴더. 의존성 이름의 단위 |
| 패키지 | 같은 폴더의 Go 파일. 함수를 다른 폴더에 공개하려면 이름이 대문자로 시작한다 |
| 핸들러 | `func(c *gin.Context)` 처럼 요청 하나를 처리하는 함수 |
| 미들웨어 | 핸들러 앞뒤에서 공통 일을 하는 함수. 로그, 복구, CORS |
| 환경 변수 | 포트, DB 주소, 비밀키처럼 코드 밖에 두는 값 |

비유하면 모듈은 작업실이고, 패키지는 서랍이며, 핸들러는 창구 직원이다. 환경 변수는 작업실 문에 붙여 둔 오늘의 비밀번호다. 비밀번호를 소스에 적으면 저장소를 복사하는 사람 모두가 알게 된다.

선행으로 구조체, 에러 값, 인터페이스, 고루틴을 조금 알고 있으면 다음 파일이 빨리 읽힌다. 없는 부분은 study1에서 게시판에 필요한 만큼만 다시 본다.

## ③ 🛠️ 실기

### 설치와 버전

1. https://go.dev/dl/ 에서 Windows amd64 설치 파일을 받아 설치한다. 이 저장소는 Go 1.27.4로 확인하고 Gin v1.12.0을 쓴다.
2. 새 터미널에서 아래를 실행한다.

```powershell
go version
go env GOVERSION
go env GOPATH
go env GOROOT
```

`go version`이 `go1.27` 이상으로 보여야 한다. `go`를 못 찾으면 설치 경로 `C:\Program Files\Go\bin`을 PATH에 넣는다.

### 모듈과 Gin

이미 이 폴더에는 모듈이 있다. 빈 폴더에서 다시 만든다면:

```powershell
go mod init github.com/samam0234/my-go-gin-study
go get github.com/gin-gonic/gin@v1.12.0
```

`go.mod`의 `module` 줄이 다른 패키지를 import 할 때 쓰는 접두어다. `go get`은 라이브러리를 받아 `go.mod`와 `go.sum`에 적는다. Gin은 실행 파일이 아니므로 `go install github.com/gin-gonic/gin`으로 깔지 않는다.

### 폴더

```text
main.go                     서버 입구. internal/app 을 호출
internal/config             환경 변수
internal/store              DB
internal/httpapi            라우터와 핸들러
internal/auth               비밀번호 해시와 JWT
internal/ai                 문서 발췌와 모델 호출
internal/mcp                MCP JSON-RPC
demo/step00 ~ step07        기초 단계별 작은 프로그램
demo/step11gorm             GORM
demo/step11mongo            MongoDB
project/web                 브라우저 화면
project/openapi.yaml        API 문서
project/Dockerfile          멀티 스테이지 이미지
project/cmd/mcp             MCP 표준 입출력 입구
```

`internal` 폴더는 이 모듈 바깥에서 import 할 수 없다. 학습용 공개 라이브러리가 아니라 이 서버의 속이다.

### 개발 서버

```powershell
go run ./demo/step00
go run .
```

`step00`은 버전과 `APP_ENV`, 비밀키 설정 여부만 출력하고 끝난다. `go run .` 은 `:8080`에서 요청을 기다린다. 코드가 바뀌면 프로세스를 멈추고 다시 실행한다. Go 기본 도구에는 저장 즉시 재시작이 없다. Cursor에서 저장 후 터미널에서 다시 `go run .` 하면 된다.

### 에디터

Cursor 또는 VS Code에 공식 Go 확장을 설치한다. 확장 ID는 `golang.go` 다. 저장 시 `gofmt`가 돌아가게 두고, 안 쓰는 import는 컴파일 오류이므로 확장이 지워 주는 것을 그대로 쓴다.

### .gitignore 와 .env

저장소에 올리면 안 되는 것:

```gitignore
.env
*.db
uploads/
bin/
coverage.out
*.exe
```

`.env.example`은 키 이름만 보여 주는 예시라서 올려도 된다. 복사 후 비밀만 바꾼다.

```powershell
copy .env.example .env
```

예시 안의 `JWT_SECRET=dev-only-change-me-please-32b` 는 로컬용이다. 운영에서는 다른 긴 값으로 바꾼다. `SEED=1`이면 연습 계정과 글 하나를 만든다.

### Git

```powershell
git init
git add .
git status
```

`.env`가 목록에 있으면 `.gitignore`가 적용되기 전에 이미 추적된 것이다. `git rm --cached .env` 로 추적만 끊는다. 원격은 이 저장소용 `https://github.com/samam0234/My_Go-Gin-_Study.git` 이다.

`demo/step00/main.go`는 환경을 읽고 비밀 값은 출력하지 않는다.

```go
func run() error {
	_ = godotenv.Load(".env")
	fmt.Println("go", runtime.Version())
	fmt.Println("module github.com/samam0234/my-go-gin-study")
	fmt.Println("APP_ENV", getenv("APP_ENV", "development"))
	if os.Getenv("JWT_SECRET") == "" {
		fmt.Println("JWT_SECRET 설정됨? 아니오")
	} else {
		fmt.Println("JWT_SECRET 설정됨? 예")
	}
	return nil
}
```

1. `.env`가 있으면 그 값을 프로세스 환경에 넣는다. 없어도 오류로 끝내지 않는다.
2. 컴파일된 Go 런타임 버전을 출력한다.
3. 이 모듈 이름을 출력해, 지금 선 폴더가 맞는지 확인한다.
4. `APP_ENV`가 없으면 `development`라고 말한다.
5. 비밀 문자열 자체는 찍지 않고, 있는지 여부만 말한다.

## ④ 🔗 연동 포인트

이 단계의 산출물은 `go.mod`, `.gitignore`, `.env` 다. study1의 `go run ./demo/step01` 이 같은 모듈 안에서 Gin을 찾고, study6의 SQLite 파일 이름도 `DB_DSN`에서 온다. Git 저장소는 이후 GitHub Actions(study16)가 `go test ./...` 를 돌리는 기준이 된다.

## ⑤ 오류 노트

| 증상 | 원인 | 해결 |
| --- | --- | --- |
| `go`를 인식하지 못함 | PATH에 `Go\bin`이 없음 | 설치 경로를 PATH에 넣고 터미널을 다시 연다 |
| `gin` 패키지를 못 찾음 | `go.sum`이 없거나 모듈 밖에서 실행 | 저장소 루트에서 `go mod tidy` |
| `go: module ... requires go >= 1.27` | 설치된 Go가 오래됨 | 1.27 이상으로 올린다 |
| `:8080` bind 오류 | 이미 서버가 떠 있음 | 기존 프로세스를 끄거나 `HTTP_ADDR=:8081` |
| `.env`를 바꿨는데 반영이 안 됨 | 프로세스가 시작할 때만 읽음 | 서버를 다시 실행한다 |
| 비밀이 `git status`에 보임 | `.env`가 무시 목록 밖에 있음 | `.gitignore` 확인 후 캐시에서 제거 |

## ⑥ 직접 해보기

1. `go version`과 `go env GOPROXY` 결과를 메모한다.
2. `.env`의 `HTTP_ADDR`를 `:8090`으로 바꾸고 `go run .` 후 주소를 확인한다. 끝나면 `:8080`으로 되돌린다.
3. `go run ./demo/step00` 출력에서 JWT 설정 여부가 `예` 또는 `아니오`로만 보이는지 확인한다.

## ⑦ ☐ 체크리스트

- [ ] `go version`이 1.27 이상이다
- [ ] `go.mod`의 모듈 경로를 읽고 import 접두어와 연결했다
- [ ] Go 모듈 프로젝트, Gin 라우터
- [ ] `.env.example`을 `.env`로 복사했고 `.env`는 Git에 없다
- [ ] `go run ./demo/step00` 이 비밀 값 없이 끝난다
- [ ] 에디터 Go 확장이 저장 시 포맷한다
- [ ] `git status`에 `*.db`, `uploads/`, `.env`가 없다

## ⑧ 다음 단계

모듈이 서 있으면 study1에서 Gin이 첫 HTML과 JSON을 돌려주게 한다.
