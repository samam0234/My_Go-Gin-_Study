# study2. 핵심 구조

## ① 목표

Gin 앱을 엔진과 기능 패키지로 나누고, 패키지 함수를 호출해 데이터를 넘기는 방식으로 `/health` 와 `/memos` 를 만든다.

## ② 📖 이론

이 프레임워크의 기본 단위는 엔진, 라우트 그룹, 핸들러, 미들웨어다. 한 파일에 다 넣으면 게시판이 커질 때 건강 체크와 글 로직이 엉킨다. Go에서는 폴더가 곧 패키지다.

- `main` 패키지만 실행 파일이 된다.
- 다른 폴더의 이름은 소문자 패키지 이름으로 부른다.
- 바깥에서 쓰려면 함수 이름이 대문자다. `Register`는 공개, `register`는 그 패키지 안에서만 보인다.
- 인터페이스는 `implements` 키워드가 없다. 필요한 메서드만 있으면 그 인터페이스로 쓸 수 있다. 저장소가 나중에 `FindAll() ([]Post, error)` 를 만족하면 핸들러는 구현체를 몰라도 된다.

데이터를 넘기는 방법은 세 가지다.

1. 함수 인자. `memo.Register(r, items)` 처럼 엔진과 슬라이스를 넘긴다.
2. 핸들러가 요청에서 읽기. 쿼리, 경로, JSON.
3. 반환 값. `(Post, error)`.

비유하면 `main`은 건물 로비이고 `health`, `memo`는 각 창구다. 로비는 창구를 열기만 하고, 창구 안의 문구는 그 패키지가 정한다.

## ③ 🛠️ 실기

`demo/step02` 를 실행한다.

```powershell
go test ./demo/step02
go run ./demo/step02
```

`health/health.go`

```go
package health

func Register(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
}
```

1. 패키지 이름은 폴더 이름과 같게 `health` 다.
2. `Register`는 대문자라 `main`이 부를 수 있다.
3. 엔진에 `GET /health` 를 등록한다.
4. 본문은 스펙의 건강 체크와 같이 `{"status":"ok"}` 다. 이 경로는 `/api` 접두어가 없다.

`memo/memo.go`

```go
type Item struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
}

func Register(r *gin.Engine, items []Item) {
	r.GET("/memos", func(c *gin.Context) {
		c.JSON(200, gin.H{"data": items})
	})
}
```

1. `Item`은 메모 한 건이다.
2. `` `json:"id"` `` 태그는 JSON 이름을 소문자 `id`로 고정한다. 태그가 없으면 공개 필드 이름 `ID`가 그대로 나간다.
3. `items`는 호출자가 넘긴 슬라이스다. 핸들러는 그 슬라이스를 닫아서 기억한다.
4. 응답은 `{"data":[...]}` 로 감싼다. 나중 스펙의 성공 본문과 같은 습관이다.

`main.go`

```go
func newEngine() *gin.Engine {
	r := gin.New()
	health.Register(r)
	memo.Register(r, []memo.Item{{ID: 1, Text: "패키지로 나눈 메모"}})
	return r
}
```

1. 빈 엔진을 만든다.
2. 건강 체크 패키지에 엔진만 넘긴다.
3. 메모 패키지에는 엔진과 데이터 한 건을 넘긴다. 패키지끼리 직접 부르지 않고, `main`이 연결한다.
4. 테스트와 `main`이 같은 엔진 함수를 쓴다.

실제 서버 `internal/httpapi/server.go` 의 `routes()` 도 같은 구조다. 엔진을 만들고 미들웨어를 얹은 뒤 경로를 메서드에 연결한다. 다만 패키지를 더 쪼개 인증, 글, AI, 부가 기능 파일로 나눴다.

## ④ 🔗 연동 포인트

`health.Register`의 JSON은 공통 스펙 `GET /health` 의 본문이다. `memo.Register`에 넘긴 슬라이스는 study3에서 파일로 읽고, study6에서 DB 조회로 바뀐다. 핸들러 서명은 그대로 두고 데이터 출처만 바꾸면 된다.

## ⑤ 오류 노트

| 증상 | 원인 | 해결 |
| --- | --- | --- |
| `cannot refer to unexported name` | 함수가 소문자 | 다른 패키지에서 부를 이름은 대문자로 |
| import 사이클 | `health`가 `main`을 import | 의존은 `main` → 기능 패키지 한 방향으로만 |
| JSON에 `ID`가 대문자 | 구조체 태그 누락 | `json:"id"` |
| `/health`가 404 | `Register`를 호출하지 않음 | `main`에서 엔진을 넘겼는지 확인 |
| 데이터가 비어 있음 | nil 슬라이스를 JSON으로 보냄 | 빈 슬라이스 `[]Item{}` 를 넘긴다. nil은 `null`이 된다 |

## ⑥ 직접 해보기

1. `memo.Item`에 `Done bool \`json:"done"\`` 를 추가하고 응답에 필드가 생기는지 본다.
2. `health` 패키지에 `GET /health` 가 두 번 등록되면 어떻게 되는지 실험하고 한 번만 두도록 되돌린다.
3. `go test ./demo/step02` 로 두 경로의 본문을 확인한다.

## ⑦ ☐ 체크리스트

- [ ] 엔진, 핸들러, 패키지의 역할을 한 문장씩 말했다
- [ ] 대문자 함수만 다른 패키지에서 호출했다
- [ ] `main`이 슬라이스를 `memo.Register`에 넘겼다
- [ ] `json` 태그로 필드 이름을 정했다
- [ ] `/health` 본문이 `{"status":"ok"}` 다
- [ ] `go test ./demo/step02` 가 통과한다

## ⑧ 다음 단계

메모리 슬라이스를 study3에서 JSON 파일과 구조체 값으로 옮긴다.
