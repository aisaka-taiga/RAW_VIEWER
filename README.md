# Photo Viewer MVP

Go 기반 gRPC 서버와 Electron 데스크톱 앱으로 이루어진 사진 뷰어 MVP입니다.

## 구성

- `cmd/photoflowd`: Go gRPC 서버
- `proto/photo_engine.proto`: gRPC API 정의
- `apps/desktop`: Electron 데스크톱 앱

## 실행 전 준비

- Go 1.23.5 이상
- Node.js와 npm
- `exiftool`
- `protoc`
- `protoc-gen-go`, `protoc-gen-go-grpc`

`exiftool`이 PATH에 없으면 서버 실행 시 `--exiftool-path`에 전체 경로를 넣어야 합니다.

## 1. proto 코드 생성

`proto/photo_engine.proto`를 수정했다면 프로젝트 루트에서 아래 스크립트를 실행합니다.

```powershell
.\scripts\gen-proto.ps1
```

생성 결과는 `gen/` 아래에 들어갑니다.

## 2. Go gRPC 서버 실행

서버는 기본적으로 `127.0.0.1:50051`에서 대기합니다. Electron 앱도 이 주소로 고정 연결하므로, 서버와 앱은 같은 주소를 사용해야 합니다.

기본값은 아래와 같습니다.

- 사진 폴더: `C:\Photos`
- 캐시 폴더: `data/thumbs`
- SQLite 파일: `data/app.db`

프로젝트 루트에서 서버를 실행합니다.

```powershell
cd "<프로젝트 루트>"
$env:GOCACHE = Join-Path $PWD ".gocache"
$env:GOMODCACHE = Join-Path $PWD ".gomodcache"
& "C:\Program Files\Go\bin\go.exe" run ./cmd/photoflowd `
  --library-root "C:\Photos" `
  --cache-dir "data/thumbs" `
  --exiftool-path "C:\exiftool-13.57_64\exiftool(-k).exe" `
  --sqlite-path "data/app.db" `
  --listen "127.0.0.1:50051"
```

`--library-root`는 실제 사진이 있는 폴더로 바꿔 주세요. `--exiftool-path`도 본인 PC의 실제 경로로 바꿔 주세요. PATH에 등록돼 있으면 `exiftool`만 써도 됩니다.

## 3. Electron 앱 실행

별도 터미널에서 데스크톱 앱을 실행합니다.

```powershell
cd "<프로젝트 루트>\apps\desktop"
npm install
npm start
```

서버가 먼저 실행 중이어야 앱이 정상 동작합니다.

PowerShell에서 `npm`이 `npm.ps1`로 실행되며 막히면 아래 중 하나를 사용합니다.

```powershell
npm.cmd install
npm.cmd start
```

또는 현재 사용자 범위에서 한 번만 정책을 완화할 수 있습니다.

```powershell
Set-ExecutionPolicy -Scope CurrentUser RemoteSigned
```

## 실행 순서 요약

1. 필요하면 `.\scripts\gen-proto.ps1`로 proto를 다시 생성합니다.
2. Go 서버를 실행합니다.
3. Electron 앱을 실행합니다.
4. 앱에서 사진 폴더를 스캔합니다.
