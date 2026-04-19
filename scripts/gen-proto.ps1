$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$proto = Join-Path $root 'proto\photo_engine.proto'
$outDir = Join-Path $root 'gen'

$protoc = "C:\workspace\protoc-25.9-win64\bin\protoc.exe"
if (-not (Test-Path $protoc)) {
  $protoc = "protoc"
  if (-not (Get-Command $protoc -ErrorAction SilentlyContinue)) {
    throw "protoc not found at $protoc or on PATH. Install protoc, then rerun this script."
  }
}

& $protoc "--proto_path=$root\proto" "--go_out=$outDir" "--go_opt=paths=source_relative" "--go-grpc_out=$outDir" "--go-grpc_opt=paths=source_relative" $proto
