# studyFinal. Gin 서비스 마무리

## ① 목표

studyMidpoint의 게시판에 study11부터 study18까지를 더해, Gin 프로세스 하나로 DB 선택, 외부 연습 기능, 문서 질문, Docker, 테스트, 보안 점검을 끝낸다.

## 기능 목록

- 스펙 v1: 건강 체크, 회원, 글, 댓글, 페이지, 오류 형식, Bearer JWT
- DB: 기본 SQLite, 환경 변수로 PostgreSQL 또는 MariaDB
- 화면: `/` 에서 로그인, 목록, 상세, 댓글, 문서 질문
- 업로드, 메일 대기열, 결제 연습, OAuth 연습
- OpenAPI `/api-docs`
- 목록 5초 캐시, 메일 워커, 선택적 호출 제한
- 웹소켓 `post.created`
- mock AI, 발췌 근거 답, SSE, 읽기 전용 MCP
- 멀티 스테이지 이미지, Compose, GitHub Actions

## 연동 구성도

```text
브라우저 http://localhost:8080/
  |  HTML / JS
  |  Bearer JWT
  v
Gin
  |-- store  SQLite 파일  또는  Postgres / MariaDB
  |-- auth   bcrypt + JWT_SECRET
  |-- cache  메모리 5초, 쓰기 시 삭제
  |-- mail   mail_outbox → 로그 워커
  |-- hub    /ws 알림
  |-- ai     mock 또는 ANTHROPIC_API_KEY / OLLAMA_URL
  |-- mcp    MCP_TOKEN, 읽기 도구만
  v
project/openapi.yaml
```

모델 제공자와 MCP 클라이언트는 브라우저에 키를 받지 않는다. 브라우저가 아는 비밀은 자신의 액세스 토큰뿐이다.

## 구현 순서

1. studyMidpoint까지 회원과 글이 `go test ./internal/httpapi -run TestBoardFlow` 를 통과하게 둔다.
2. `DB_DRIVER` 분기를 넣고 SQLite로 다시 테스트를 통과시킨다. Postgres와 MariaDB는 DSN이 있을 때만 기동 확인한다. study11.
3. `project/web`을 루트에 연결하고 로그인, 상세, 목록 갱신을 브라우저에서 본다. study12.
4. 업로드 한도, 메일 테스트, 결제 orderId, OAuth mock을 Bearer 뒤에 둔다. study13.
5. `project/openapi.yaml`과 오류 코드를 라우터와 맞춘다. study14.
6. 목록 캐시와 메일 워커를 테스트로 고정한다. study15.
7. Dockerfile과 Compose, Actions를 추가한다. 이미지에 `.env`를 넣지 않는다. study16.
8. `/ws` 알림 테스트를 통과시킨다. study17.
9. 문서 질문과 MCP를 mock으로 통과시킨다. 실키는 환경 변수로만 바꾼다. study18.
10. 아래 배포 체크리스트를 로컬에서 수행하고 회고를 적는다.

## 배포 체크리스트

- [ ] `go test ./...` 통과
- [ ] `.env`가 Git 상태에 없다
- [ ] `JWT_SECRET`과 `MCP_TOKEN`이 예시 문자열이 아니다
- [ ] `APP_ENV=production` 에서 예시 비밀로는 시작하지 않는다
- [ ] `SEED`가 운영에 없다
- [ ] `GET /health` 200
- [ ] `GET /api-docs/openapi.yaml` 이 열린다
- [ ] 로그인 200, 틀린 비밀번호 401 `INVALID_CREDENTIALS`
- [ ] 토큰 없이 글 등록 401, 남의 글 수정 403
- [ ] 목록 `meta`와 상세 `body`가 스펙과 같다
- [ ] 응답과 로그에 비밀번호 해시가 없다
- [ ] 화면은 `textContent`만 쓴다
- [ ] `CORS_ORIGIN`이 실제 출처이고 `*` 가 아니다
- [ ] DB 볼륨 또는 덤프 백업 경로를 적었다
- [ ] 컨테이너 사용자가 root가 아니다
- [ ] AI 키가 프론트 번들과 로그에 없다
- [ ] MCP HTTP는 토큰 없이 401이고 도구에 삭제가 없다
- [ ] Actions 테스트 잡이 푸시 후 녹색이다

## 최종 회고

이 서비스에서 Gin이 담당한 일은 경로를 함수에 연결하고, JSON을 구조체에 담으며, 미들웨어로 로그와 복구와 CORS를 공통 처리한 것이다. 스펙의 상태 코드, SQL, 권한, 파일 크기, 모델 키는 프레임워크가 대신 정해 주지 않아서 `store`와 `writeError`에 직접 적었다.

유지할 결정:

- 본 서버 DB는 SQL만. MongoDB와 GORM은 비교용 예제로 분리한다.
- 검증 실패는 400 `VALIDATION_ERROR` 로 통일한다.
- 모델과 게시판 쓰기 도구를 MCP에 열지 않는다.
- 화면은 같은 출처의 정적 파일로 두고 경로 계약을 지킨다.

다음에 볼 것:

- 액세스 토큰 수명을 1시간으로 줄이고 리프레시 토큰을 해시로 저장한다.
- 메일 로그 구현을 SMTP로 바꿀 때 대기열 테이블은 유지한다.
- 프로세스를 둘 이상 띄우면 메모리 캐시와 웹소켓 허브를 밖으로 뺀다.

로컬에서 확인한 흐름은 연습 계정 로그인, 시드 글 상세(`첫 글`, `안녕하세요`), 새 글 `학습 메모`의 목록 반영, 문서 질문의 발췌 답이다.

## ② 📖 이론

마무리는 새 기능의 추가가 아니라, 앞 단계의 경로와 환경 변수와 테스트가 한 프로세스에서 동시에 맞는지 보는 일이다.

## ③ 🛠️ 실기

```powershell
copy .env.example .env
go test ./...
$env:SEED = "1"
go run .
```

브라우저 http://localhost:8080 과 `GET /health`, `GET /api-docs/openapi.yaml` 을 연다. Docker가 있으면 `docker compose -f project/docker-compose.yml up --build` 로 같은 포트를 한 번 더 확인한다.

## ④ 🔗 연동 포인트

구성도의 화살표가 환경 변수 이름과 같다. `DB_DSN`은 SQL, `JWT_SECRET`은 로그인, `MCP_TOKEN`은 도구, `ANTHROPIC_API_KEY`는 모델이다. 하나를 비우면 그 화살표만 연습 구현이나 시작 거부로 바뀌고 나머지 경로는 남는다.

## ⑤ 오류 노트

| 증상 | 원인 | 해결 |
| --- | --- | --- |
| 테스트는 되는데 화면만 실패 | 루트가 아닌 폴더에서 실행 | 저장소 루트에서 `go run .` |
| 운영 시작이 즉시 실패 | 예시 비밀 또는 짧은 비밀 | 체크리스트의 비밀 항목 |
| AI만 503 | 제공자를 바꿨는데 서버가 없음 | `AI_PROVIDER=mock` 으로 확인 후 키를 넣는다 |
| 백업 후 글이 0 | 빈 파일 경로를 복원 | `/data/board.db` 인지 확인 |

## ⑥ 직접 해보기

1. 체크리스트를 위에서부터 표시하고, 오늘 못 한 Docker 항목은 비워 둔다.
2. `AI_PROVIDER`를 mock과 다른 값으로 바꿨다가 mock으로 되돌려 글 CRUD가 영향 없는지 본다.
3. 회고의 "다음에 볼 것" 중 하나를 오류 노트에 실험 계획으로 한 줄 남긴다.

## ⑦ ☐ 체크리스트

- [ ] 기능 목록이 이 저장소 경로와 같다
- [ ] 구성도의 비밀이 환경 변수다
- [ ] 구현 순서 1의 게시판 테스트가 통과한다
- [ ] 구현 순서 9의 AI, MCP 테스트가 통과한다
- [ ] 배포 체크리스트의 로컬 항목을 수행했다
- [ ] 최종 회고에 유지할 결정을 남겼다

## ⑧ 다음 단계

이 저장소 밖에서는 같은 스펙을 유지한 채 토큰 수명, SMTP, 여러 프로세스만 하나씩 바꾼다. 경로와 오류 코드는 그대로 둔다.
