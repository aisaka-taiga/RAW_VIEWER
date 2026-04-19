# Photo Viewer MVP

Go gRPC 엔진과 Electron 데스크톱 셸로 구성된 사진 뷰어 MVP입니다.

## 현재 구조

- `cmd/photoflowd`: Go gRPC 서버
- `proto/photo_engine.proto`: gRPC API 정의
- `apps/desktop`: Electron 데스크톱 앱

## proto 생성

`protoc`가 설치되어 있으면 아래 명령으로 Go 코드를 생성할 수 있습니다.

```powershell
.\scripts\gen-proto.ps1
```

생성 결과는 `gen/` 아래에 만들어집니다.

## Electron 데스크톱 앱

Electron 셸은 `apps/desktop`에 있습니다.

의존성 설치:

```powershell
cd apps\desktop
npm install
```

앱 실행:

```powershell
npm start
```

메인 프로세스는 `127.0.0.1:50051`의 Go gRPC 백엔드에 연결합니다.
따라서 먼저 `photoflowd`를 실행해야 합니다.

## 다음 단계

gRPC API를 Electron 렌더러에 연결해 실제 사진 목록과 썸네일 그리드를 표시하고, Go 쪽에는 대량 인덱싱과 프리로드 작업을 더 붙입니다.


cd "C:\Users\KURISU\Documents\New project"
$env:GOCACHE=Join-Path $PWD '.gocache'
$env:GOMODCACHE=Join-Path $PWD '.gomodcache'
& 'C:\Program Files\Go\bin\go.exe' run .\cmd\photoflowd `
  --library-root "C:\workspace\photos" `
  --cache-dir "C:\Users\KURISU\Documents\New project\data\thumbs" `
  --exiftool-path "C:\workspace\exiftool-13.55_64\exiftool(-k).exe" `
  --sqlite-path "C:\Users\KURISU\Documents\New project\data\app.db" `
  --listen "127.0.0.1:50051"
