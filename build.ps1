$env:GOOS = "linux"
$env:CGO_ENABLED = "0"
$env:GOARCH = "amd64"
go build -o output/main main.go
~\Go\Bin\build-lambda-zip.exe -output output/main.zip output/main