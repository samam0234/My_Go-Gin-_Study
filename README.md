# Go (Gin) 학습

Gin 하나로 게시판 API를 만드는 학습 저장소입니다. 다른 웹 프레임워크와 코드를 섞지 않고, [게시판 공통 API 스펙 v1](https://app.notion.com/p/3d6d7dec5e2e8139adbfd36dd506c65e)의 경로·상태 코드·JSON 모양을 Gin으로 맞춥니다.

노션 페이지: [Go (Gin)](https://app.notion.com/p/40bd7dec5e2e81648a66eed92fc2202f)

## 실행

Go 1.27 이상이 필요합니다. Gin 1.12를 사용합니다.

```powershell
copy .env.example .env
go test ./...
$env:SEED = "1"
go run .
```

브라우저에서 http://localhost:8080 을 엽니다. `SEED=1`이면 연습 계정 `mina@example.com` / `password123` 과 글 하나가 생깁니다. 이 비밀번호는 로컬 학습용입니다.

같은 서버는 `go run ./project/cmd/api` 로도 켜집니다. MCP 표준 입출력 서버는 `go run ./project/cmd/mcp` 입니다.

## 문서

| 파일 | 내용 |
| --- | --- |
| [study0.md](study0.md) | 설치, 모듈, 폴더, 환경 변수 |
| [study1.md](study1.md) | 첫 응답, 템플릿, Go 기본 문법 |
| [study2.md](study2.md) | 패키지로 핸들러 나누기 |
| [study3.md](study3.md) | 구조체, JSON, 파일 저장 |
| [study4.md](study4.md) | 라우팅, 쿼리, 404 |
| [study5.md](study5.md) | 바인딩, 검증, 중복 제출 |
| [study6.md](study6.md) | sqlx와 CRUD |
| [study7.md](study7.md) | JWT, 해시, CORS, XSS |
| [study8.md](study8.md) | 폴더와 설정 분리 |
| [study9.md](study9.md) | httptest, 로그, 자주 나는 오류 |
| [study10.md](study10.md) | 빌드와 Docker |
| [studyMidpoint.md](studyMidpoint.md) | 게시판 서비스 1차 |
| [study11.md](study11.md) | MariaDB, PostgreSQL, MongoDB, GORM |
| [study12.md](study12.md) | 같은 서버의 화면 |
| [study13.md](study13.md) | 업로드, 메일, 결제 연습, OAuth |
| [study14.md](study14.md) | OpenAPI, 페이지네이션, 오류 형식 |
| [study15.md](study15.md) | 캐시, 메일 워커 |
| [study16.md](study16.md) | Compose, GitHub Actions, 백업 |
| [study17.md](study17.md) | 웹소켓 알림 |
| [study18.md](study18.md) | 문서 질문, 스트리밍, MCP |
| [studyFinal.md](studyFinal.md) | 위 내용을 한 서비스로 마무리 |

## 코드 지도

- `demo/step00`~`step07`, `step11gorm`, `step11mongo`: 단계별 작은 프로그램
- `internal/`: 설정, 저장소, HTTP, AI, MCP
- `project/web`: 브라우저 화면
- `project/openapi.yaml`: API 문서
- `project/Dockerfile`, `project/docker-compose.yml`: 이미지와 선택 DB
- `.github/workflows/test.yml`: 테스트와 이미지 빌드

비밀값은 `.env`에만 둡니다. `.env.example`만 저장소에 있습니다.
