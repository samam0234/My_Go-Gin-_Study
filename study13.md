# study13. 외부 서비스 연동

## ① 목표

파일 업로드, 메일 대기열, 결제 연습, 구글 로그인 연습을 Gin 라우트에 붙이고, 비밀과 실서비스 키는 환경 변수로만 둔다.

## ② 📖 이론

외부 서비스는 우리 DB와 다른 시스템의 일이다. 학습 서버는 그 시스템을 직접 대체하지 않고, 없을 때도 흐름이 보이게 연습 구현을 둔다.

- 파일: 서버 디스크의 `UPLOAD_DIR`. 허용 확장자는 txt, md, png, jpg, jpeg, webp, pdf. 크기 기본 5MB. 넘으면 413 `PAYLOAD_TOO_LARGE`.
- 메일: 가입과 테스트 요청이 `mail_outbox`에 행을 넣는다. 워커나 `POST /api/mail/test` 가 한 통을 "보냈다"고 표시한다. 연습 구현은 로그에 받는 사람과 제목만 남긴다.
- 결제: `orderId`가 같은 사용자의 중복 키다. 처음은 201 `pending`, 같은 주문 같은 금액은 200, 금액이 다르면 409 `CONFLICT`. `POST /api/payments/{id}/confirm` 이 `paid`로 바꾼다. 카드사와 통신하지 않는 연습 상태기계다.
- 소셜 로그인: 실제 구글은 인가 코드와 PKCE다. `OAUTH_MOCK=1` 이면 구글에 나가지 않고 `oauth.mina@example.com` 사용자에게 우리 JWT를 준다. 실서비스는 `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URL` 이 있을 때만 시작한다. 이메일이 확인되지 않았으면 계정을 만들지 않는다.

비유하면 업로드는 물품 보관소, 메일은 발송 대기 상자, 결제는 영수증 번호, OAuth는 다른 건물에서 확인서를 받아 우리 팔찌로 바꾸는 일이다.

## ③ 🛠️ 실기

업로드는 `internal/httpapi/extra.go` 의 `upload` 다.

```go
ext := strings.ToLower(filepath.Ext(header.Filename))
switch ext {
case ".txt", ".md", ".png", ".jpg", ".jpeg", ".webp", ".pdf":
default:
	writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "입력값을 확인하세요.", map[string][]string{
		"file": {"txt, md, png, jpg, webp, pdf 만 올릴 수 있습니다."},
	})
	return
}
```

1. 확장자를 소문자로 제한한다. HTML이나 실행 파일은 거절한다.
2. 저장 이름은 원본이 아니라 무작위 문자열이다. 경로 탈출을 막는다.
3. `io.LimitReader`로 한도보다 1바이트 더 읽히면 파일을 지우고 413을 준다.
4. 응답은 `{data:{filename,size,url}}`. `url`은 `/uploads/이름` 이다.
5. 다운로드는 `Content-Disposition` 첨부와 `nosniff` 다. 브라우저가 업로드된 내용을 페이지로 실행하지 않게 한다.

메일 테스트:

```http
POST /api/mail/test
Authorization: Bearer <token>
```

202 `{data:{status:"sent",to}}`. 본문의 `to`가 없으면 내 이메일이다.

결제:

```json
{ "orderId": "order-1", "amount": 1200 }
```

`amount`는 1 이상 1000만 이하다.

OAuth 연습:

```http
GET /api/auth/oauth/google/start
```

연습 모드 응답의 `authUrl`은 우리 콜백이다. 그 주소를 다시 GET 하면 로그인과 같은 `{data:{user,accessToken}}` 이 온다. `Accept: text/html` 이면 토큰을 `localStorage`에 넣고 `/` 로 보내는 짧은 HTML을 준다.

실서비스 분기는 `exchangeGoogleHTTP` 다. 토큰 엔드포인트에 `code`와 `code_verifier`를 보내고, userinfo에서 `email_verified`를 확인한다. 시작 주소에는 `code_challenge`와 `S256`을 넣는다. state는 10분 동안만 서버 메모리에 있다.

키가 없고 연습 모드도 아니면 501 `OAUTH_NOT_CONFIGURED` 다.

## ④ 🔗 연동 포인트

모든 쓰기 경로는 study7의 Bearer 뒤에 있다. 업로드 URL은 화면이 나중에 `<img>`나 링크로 쓸 수 있지만, 이 단계에서는 API 응답의 `url`을 확인하는 것으로 충분하다. 결제와 메일은 스펙 v1의 글 경로에 섞지 않고 `/api/payments`, `/api/mail/test` 로 둔다. OAuth가 돌려주는 토큰은 비밀번호 로그인과 같은 `requireAuth`를 통과한다.

## ⑤ 오류 노트

| 증상 | 원인 | 해결 |
| --- | --- | --- |
| 413 | 파일 또는 JSON이 한도를 넘음 | `MAX_UPLOAD_BYTES`와 1MB JSON 한도 |
| 400 file 필드 | 확장자가 목록 밖 | txt, md, 이미지, pdf만 |
| 결제 409 | 같은 orderId에 다른 금액 | 새 orderId 또는 같은 금액으로 재조회 |
| 501 OAuth | 키가 없고 mock이 꺼짐 | 개발은 `OAUTH_MOCK=1` |
| 소셜 로그인 400 | state 만료 또는 코드 불일치 | start부터 다시 |
| 메일이 안 감 | 연습 구현은 로그만 남김 | 로그의 `mail to=` 를 확인. SMTP는 그 다음에 같은 대기열에 연결 |

## ⑥ 직접 해보기

1. 로그인한 뒤 1KB 텍스트 파일을 `file` 필드로 올리고, 응답 URL을 GET 한다.
2. 같은 `orderId`로 결제를 두 번 만들어 두 번째가 200인지 본다.
3. `GET /api/auth/oauth/google/start` 의 `authUrl`을 호출해 `accessToken`이 생기는지 본다.

## ⑦ ☐ 체크리스트

- [ ] 업로드 확장자와 크기 제한이 있다
- [ ] 저장 파일 이름에 원본 경로가 없다
- [ ] 메일은 대기열에 넣고 연습 발송은 제목만 로그에 남긴다
- [ ] 결제는 orderId로 중복을 구분한다
- [ ] OAuth 연습 모드는 외부 호출 없이 JWT를 준다
- [ ] 실서비스 키는 환경 변수이고 Git에 없다
- [ ] 확인되지 않은 이메일로는 계정을 만들지 않는다

## ⑧ 다음 단계

경로가 늘어났으면 study14에서 그 경로를 OpenAPI와 오류 규칙으로 문서화한다.
