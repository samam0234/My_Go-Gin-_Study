# study16. 운영과 CI

## ① 목표

Docker Compose로 서버를 띄우고, GitHub Actions가 테스트와 이미지 빌드를 하게 하며, 로그와 백업을 체크리스트로 확인한다.

## ② 📖 이론

로컬 `go run` 은 개발용이다. 같은 커밋을 다른 기계에서 돌리려면 이미지와 환경 변수가 필요하다. Compose는 그 이미지와 선택 DB를 한 파일에 적는다. Actions는 푸시할 때마다 `go test ./...` 와 `docker build` 를 실행한다. 배포 버튼을 대신 눌러 주는 단계까지 가되, 레지스트리 비밀번호는 저장소에 두지 않는다.

로그는 요청 한 줄과 오류 한 줄이다. 모니터링의 최소 단위는 `/health` 가 실패할 때 알 수 있는 것이다. 백업의 최소 단위는 SQLite 파일 또는 SQL 덤프다.

비유하면 Actions는 출발 전 점검표이고, Compose는 그 점검표를 통과한 상자를 책상 위에 올리는 일이다.

## ③ 🛠️ 실기

`.github/workflows/test.yml`

- `push`, `pull_request` 마다 실행
- Go 1.27.4로 `go test ./...`
- `docker build -f project/Dockerfile -t gin-study:local .`

로컬 Compose:

```powershell
docker compose -f project/docker-compose.yml up --build
```

기본 서비스 `api`는 SQLite 볼륨 `boarddata`를 `/data`에 마운트한다. `SEED=1`은 이 개발 Compose에만 있다. 운영으로 복사할 때는 그 줄을 지우고 `JWT_SECRET`을 바꾼다.

PostgreSQL만 따로:

```powershell
docker compose -f project/docker-compose.yml --profile postgres up -d postgres
```

MariaDB는 `--profile mariadb` 다. API를 그 DB에 붙일 때는 `DB_DRIVER`와 `DB_DSN`을 서비스 환경에 덮어쓴다. 프로필 없이 올리면 SQLite만 돈다.

### 배포 체크리스트

- [ ] `go test ./...` 가 로컬에서 통과했다
- [ ] Actions의 test 잡이 푸시 후 녹색이다
- [ ] 이미지 빌드 잡이 성공했다
- [ ] 컨테이너 사용자가 root가 아니다
- [ ] `.env`와 업로드 파일이 이미지에 없다
- [ ] `JWT_SECRET`, `MCP_TOKEN`이 예시 값이 아니다
- [ ] `GET /health` 가 200이다
- [ ] 회원가입 없이 토큰 없는 글 등록이 401이다
- [ ] 로그에 비밀번호와 토큰 원문이 없다
- [ ] SQLite면 `/data/board.db` 를 볼륨으로 복사해 백업한다
- [ ] PostgreSQL이면 `pg_dump` 한 파일을 남긴다
- [ ] 백업 파일을 다른 폴더에 풀어 `/health` 가 되는지 한 번 본다
- [ ] `SEED`가 운영 환경에 없다
- [ ] `CORS_ORIGIN`이 실제 화면 출처다

백업 예:

```powershell
docker compose -f project/docker-compose.yml cp api:/data/board.db ./backup-board.db
```

볼륨을 직접 쓰는 경우에는 컨테이너를 멈춘 뒤 볼륨의 db 파일을 복사한다. 복사 중 쓰기가 겹치면 파일이 깨질 수 있으니, 학습 환경에서는 서버를 잠시 멈추고 복사한다.

로그를 볼 때:

```powershell
docker compose -f project/docker-compose.yml logs api
```

`rid=` 로 오류 응답의 `requestId`를 찾는다. 상태 코드만으로는 500의 원인이 안 보인다.

## ④ 🔗 연동 포인트

study10의 Dockerfile이 이 워크플로의 빌드 대상이다. study11의 Postgres 프로필은 같은 Compose 파일의 `profiles` 다. 환경 변수 키는 `internal/config`와 `.env.example`과 Compose의 `environment`가 같은 이름이다. 한 곳만 이름을 바꾸면 컨테이너는 기본값으로 떨어진다.

## ⑤ 오류 노트

| 증상 | 원인 | 해결 |
| --- | --- | --- |
| Actions가 Go를 못 맞춤 | `go-version`이 `go.mod`보다 낮음 | 1.27.4 |
| 빌드가 의존성을 못 받음 | 사설 모듈 또는 네트워크 | `go.sum`이 커밋됐는지 확인 |
| Compose가 맥락을 못 찾음 | Dockerfile 경로 | `-f project/docker-compose.yml` 을 저장소 루트에서 실행. context는 `..` |
| 재시작 후 글이 없음 | 볼륨 없이 컨테이너 안 파일 | `boarddata` 마운트 |
| 백업 파일이 열리지 않음 | 서버가 쓰는 중에 복사 | 잠시 멈추고 복사 |
| 로그에 토큰이 보임 | 헤더를 그대로 출력 | 접근 로그는 메서드, 경로, 상태만 |

## ⑥ 직접 해보기

1. `docker compose -f project/docker-compose.yml config` 로 환경 변수 이름이 `.env.example`과 같은지 본다.
2. 워크플로 YAML의 `go test` 줄과 로컬 명령을 같게 맞춘다.
3. 백업 체크리스트에서 오늘 할 수 있는 항목만 체크하고, Docker가 없는 항목은 다음에 남긴다.

## ⑦ ☐ 체크리스트

- [ ] GitHub Actions가 테스트와 이미지 빌드를 한다
- [ ] Compose 기본값은 SQLite 볼륨이다
- [ ] Postgres와 MariaDB는 프로필로만 켜진다
- [ ] 운영 비밀이 YAML에 예시 값으로 고정되어 있지 않다
- [ ] `/health`와 401을 배포 후에 확인한다
- [ ] DB 백업 파일 위치를 한 줄로 적었다
- [ ] 로그에서 요청 ID로 500을 찾을 수 있다

## ⑧ 다음 단계

상자가 뜨면 study17에서 그 서버에 웹소켓 알림을 붙인다.
