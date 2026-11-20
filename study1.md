# study1. Hello World와 기본 문법

## ① 목표

Gin으로 HTML 한 장과 JSON `{"message":"pong"}` 을 돌려주고, 그 코드를 읽는 데 필요한 Go 문법(변수, 에러 값, 구조체 태그, 고루틴)을 같은 파일에서 확인한다.

## ② 📖 이론

브라우저가 `GET /` 을 보내면 Gin 엔진이 등록된 경로와 맞추고 핸들러를 부른다. 핸들러는 `*gin.Context`로 헤더, 쿼리, 상태 코드, 본문을 다룬다.

HTML은 `html/template`로 만든다. `{{.Name}}` 자리는 데이터로 바뀌고, `<` 같은 문자는 이스케이프된다. 사용자가 `?name=<b>민아</b>` 를 보내도 굵은 글자가 되지 않고 글자로 보인다. 이것이 템플릿을 쓰는 이유다.

Go에는 예외로 흐름을 끊는 문법이 없다. 함수는 결과와 `error`를 함께 돌려주고, 호출한 쪽이 `if err != nil` 로 처리한다. 0으로 나누기 예제가 그 모양이다.

고루틴은 `go` 뒤에 함수를 붙여 동시에 돌리는 일이다. 채널은 그 함수가 끝난 값을 건네는 통로다. 웹 서버 자체도 연결마다 고루틴을 쓰지만, 처음에는 "일을 맡기고 결과를 기다린다"까지만 보면 된다.

비유하면 핸들러는 카운터이고, 템플릿은 빈칸이 있는 안내문이다. 손님이 안내문에 HTML 태그를 적어도 인쇄기는 그것을 명령이 아니라 문자로 찍는다.

## ③ 🛠️ 실기

파일은 `demo/step01/main.go` 다. 실행은 저장소 루트에서 `go run ./demo/step01` 이다. 브라우저에서 http://localhost:8080/ 과 http://localhost:8080/ping 을 연다. 개발자도구 Network에서 `/ping` 상태 200과 응답 본문을 본다. 코드를 고친 뒤에는 프로세스를 끄고 다시 실행해야 새 응답이 나온다.

```go
func newEngine() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.GET("/", home)
	r.GET("/ping", ping)
	return r
}

func home(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	name := c.DefaultQuery("name", "Gin")
	if err := page.Execute(c.Writer, map[string]any{"Name": name}); err != nil {
		c.String(http.StatusInternalServerError, "템플릿 오류")
	}
}

func ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "pong"})
}

func divide(a, b int) (int, error) {
	if b == 0 {
		return 0, fmt.Errorf("0으로 나눌 수 없음")
	}
	return a / b, nil
}

func waitDone() string {
	ch := make(chan string, 1)
	go func() { ch <- "완료" }()
	return <-ch
}
```

줄별:

1. `newEngine`은 라우터를 만들어 돌려준다. 테스트가 포트를 열지 않고 이 함수만 부른다.
2. `gin.New()`는 미들웨어가 없는 빈 엔진이다.
3. `Logger`는 요청 한 줄을 찍고, `Recovery`는 핸들러가 패닉 나도 프로세스를 살린다.
4. `GET /` 는 `home`에 연결한다.
5. `GET /ping` 은 `ping`에 연결한다.
6. `home`은 HTML 헤더를 먼저 넣는다. 헤더는 본문보다 먼저다.
7. 쿼리 `name`이 없으면 `"Gin"` 이다.
8. 템플릿에 맵을 넘긴다. 실패하면 500 문자열을 보낸다.
9. `ping`은 상태 200과 JSON 객체를 보낸다. `gin.H`는 `map[string]any` 의 별칭이다.
10. `divide`는 몫과 에러 두 값을 돌려준다.
11. 나누는 수가 0이면 에러를 만든다. 몫은 0이어도 호출자가 에러를 보면 그 몫을 쓰지 않는다.
12. `waitDone`은 버퍼 1짜리 채널을 만든다.
13. 고루틴이 `"완료"`를 넣고, 바깥은 그 값을 받을 때까지 기다린다.

템플릿 문자열의 `{{.Name}}` 이 맵 키 `Name`과 연결된다. 테스트 `TestHello`는 `<b>`가 본문에 그대로 있으면 실패하게 되어 있다. 이스케이프가 되면 `&lt;b&gt;` 로 바뀐다.

개발자도구에서 볼 것:

- Network의 Status
- Response 헤더 `Content-Type`
- `/` 는 `text/html`, `/ping` 은 `application/json`

## ④ 🔗 연동 포인트

`newEngine()` 패턴은 이후 모든 단계와 `internal/httpapi` 테스트가 그대로 쓴다. 핸들러를 `http.Handler`로 감싸 `httptest`에 넘기면 브라우저 없이 응답을 확인할 수 있다. study9에서 이 확인을 테스트 파일로 고정한다.

## ⑤ 오류 노트

| 증상 | 원인 | 해결 |
| --- | --- | --- |
| 브라우저가 연결을 거부 | 서버가 다른 포트이거나 프로세스가 종료됨 | 터미널의 `listening` 로그와 주소를 맞춘다 |
| 코드를 고쳤는데 화면이 그대로 | 프로세스를 재시작하지 않음 | Ctrl+C 후 `go run` |
| JSON이 문자열로만 보임 | `c.String`으로 직접 JSON을 씀 | `c.JSON`을 쓴다 |
| 템플릿이 빈 제목 | 맵 키와 `{{.Name}}`의 대소문자가 다름 | 키를 `Name`으로 맞춘다 |
| `declared and not used` | 변수를 만들고 안 씀 | 지우거나 `_` 로 받는다. Go는 이것을 컴파일 오류로 본다 |
| 고루틴 테스트가 끝나지 않음 | 채널에 넣는 쪽이 없음 | 버퍼와 `ch <-` 가 있는지 본다 |

## ⑥ 직접 해보기

1. `/?name=공부` 를 열고 제목이 바뀌는지, 개발자도구 Response에 태그가 실행되지 않는지 본다.
2. `divide(10, 0)` 의 에러 문자열을 테스트에서 출력해 본다.
3. `/ping` 을 두 번 호출하고 터미널 로그에 요청이 두 줄 생기는지 본다.

## ⑦ ☐ 체크리스트

- [ ] `go run ./demo/step01` 후 `/` 와 `/ping` 을 브라우저에서 열었다
- [ ] 개발자도구에서 상태 코드와 Content-Type을 구분했다
- [ ] `{{.Name}}` 이 쿼리 값으로 바뀌고 HTML은 이스케이프된다
- [ ] `if err != nil` 로 나눗셈 실패를 처리했다
- [ ] 고루틴과 채널로 `"완료"`를 받았다
- [ ] `go test ./demo/step01` 이 통과한다

## ⑧ 다음 단계

한 파일에 있던 창구를 study2에서 패키지 둘로 나눈다.
