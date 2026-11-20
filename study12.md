# study12. 화면 연동

## ① 목표

Gin이 같은 출처에서 HTML, CSS, JS를 주고, 그 화면이 스펙 JSON을 읽어 글자만 안전한 방식으로 표시하게 한다.

## ② 📖 이론

이 저장소의 화면은 또 다른 웹 프레임워크가 아니다. `project/web`의 정적 파일이고, 상태와 폼은 브라우저 기본 기능으로 처리한다.

- 상태: `state` 객체에 토큰, 페이지 번호, 검색어, 열린 글 id가 있다. 새로고침 뒤에도 남길 값은 `localStorage`의 `accessToken` 뿐이다.
- 폼: HTML `form`과 `FormData`. 서버 검증이 최종이고, `required`와 `maxlength`는 명백한 빈칸을 미리 막는다.
- 화면 갱신: `createElement`와 `textContent`. `innerHTML`은 쓰지 않는다.

UI 라이브러리를 나중에 붙이더라도 계약은 같다. `GET /api/posts`의 `data`와 `meta`, 로그인 응답의 `accessToken`, 오류의 `error.code` 다. 화면 구현이 바뀌어도 경로는 유지한다.

비유하면 Gin은 주방이고 정적 파일은 쟁반이다. 쟁반 위의 글씨는 손님이 적어 준 문장을 명령으로 실행하지 않고 그대로 보여 준다.

## ③ 🛠️ 실기

`internal/httpapi/server.go` 가 `WEB_DIR` 또는 `project/web` 을 찾으면 다음을 연다.

- `GET /` → `index.html`
- `GET /app.js`, `GET /app.css`

`project/web/app.js` 의 요청 함수:

```javascript
async function api(path, options) {
  const opts = options || {};
  const headers = { "Content-Type": "application/json" };
  if (state.token) headers.Authorization = "Bearer " + state.token;
  const res = await fetch(path, {
    method: opts.method || "GET",
    headers,
    body: opts.body ? JSON.stringify(opts.body) : undefined,
  });
  if (res.status === 204) return { status: 204 };
  const payload = await res.json().catch(function () { return {}; });
  if (!res.ok) {
    const fields = payload.error && payload.error.fields ? JSON.stringify(payload.error.fields) : "";
    const message = (payload.error && payload.error.message) || "요청에 실패했습니다.";
    throw Object.assign(new Error(fields ? message + " " + fields : message), { status: res.status });
  }
  return { status: res.status, data: payload.data, meta: payload.meta };
}
```

1. 기본 헤더는 JSON이다.
2. 토큰이 있으면 Bearer를 붙인다. 서버의 `requireAuth`와 같다.
3. `fetch`는 같은 출처라 상대 경로면 된다.
4. 204는 본문이 없으므로 JSON을 읽지 않는다. 로그아웃이 이 경우다.
5. 실패 본문이 JSON이 아니면 빈 객체로 두고 메시지를 만든다.
6. `error.message`와 `fields`를 예외 메시지로 올린다. 상태 코드도 예외에 붙인다.
7. 성공은 `data`와 `meta`만 돌려준다. 비밀번호 해시는 서버가 보내지 않고, 화면도 그 키를 그리지 않는다.

목록은 `data`의 `title`과 `author.nickname`으로 버튼을 만든다. 버튼을 누르면 `GET /api/posts/{id}` 와 댓글 목록을 상세 영역에 `textContent`로 넣는다. 등록이 201이면 목록을 다시 읽고 그 `id`의 상세를 연다.

제출 중에는 버튼을 끈다. 실패해도 `finally`에서 다시 켠다.

실행:

```powershell
$env:SEED = "1"
go run .
```

http://localhost:8080 에서 연습 계정으로 로그인하면 상태 줄에 `200 로그인했습니다.` 와 `mina (mina@example.com)` 이 보인다. 시드 글 `첫 글` 을 누르면 본문 `안녕하세요` 가 상세에 나온다.

## ④ 🔗 연동 포인트

화면의 경로 문자열이 스펙과 같아야 한다. `/api/posts`, `/api/auth/login`, `/api/documents/{id}/questions` 다. 다른 포트에서 이 파일을 열고 싶다면 `CORS_ORIGIN`에 그 출처를 넣는다. 같은 프로세스로 제공하면 CORS 없이도 된다. study7의 토큰과 study6의 목록 메타를 이 파일이 소비한다.

## ⑤ 오류 노트

| 증상 | 원인 | 해결 |
| --- | --- | --- |
| 화면이 `{"service":"gin-study"}` 만 보임 | `project/web`을 못 찾음 | 저장소 루트에서 실행하거나 `WEB_DIR`을 지정 |
| 로그인 전에 글쓰기가 401 | 토큰이 없음 | 로그인 후 상태 줄이 200인지 확인 |
| 글이 굵은 태그로 보임 | HTML로 삽입 | `textContent`만 사용 |
| 버튼이 계속 회색 | 예외가 `finally` 밖에서 남 | `withButton`의 `finally`를 유지 |
| 새로고침 후 로그아웃됨 | 토큰을 메모리에만 둠 | `localStorage`에 두고, 지우기는 로그아웃 버튼만 |

## ⑥ 직접 해보기

1. 로그인 뒤 상태 줄의 숫자와 별명 문장을 확인한다.
2. 시드 글을 눌러 제목과 본문이 상세에 따로 보이는지 본다.
3. 검색칸에 `안녕`을 넣고 목록이 줄어드는지 본다. 비우면 다시 전체다.

## ⑦ ☐ 체크리스트

- [ ] `/` 가 `index.html`을 준다
- [ ] 토큰은 Authorization 헤더로만 나간다
- [ ] 목록, 상세, 등록이 `data` 필드를 읽는다
- [ ] 오류 상태 코드가 상태 줄에 남는다
- [ ] 본문은 `textContent`로만 그린다
- [ ] 제출 중 버튼이 비활성화된다

## ⑧ 다음 단계

화면이 API를 읽으면 study13에서 파일, 메일, 결제 연습, 소셜 로그인을 같은 토큰 뒤에 둔다.
