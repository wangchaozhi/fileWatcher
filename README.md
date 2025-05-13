编译到 Windows 64位 (amd64)
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -o fiberweb-windows-amd64.exe
输出文件：fiberweb-windows-amd64.exe，可以在 64 位 Windows 系统上运行。
编译到 Linux 64位 (amd64)


$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o fiberweb-linux-amd64
输出文件：fiberweb-linux-amd64，适用于 64 位 Linux 系统。
编译到 macOS 64位 (amd64)


$env:GOOS = "darwin"
$env:GOARCH = "amd64"
go build -o fiberweb-darwin-amd64
输出文件：fiberweb-darwin-amd64，适用于 Intel 架构的 macOS。
编译到 Linux ARM64


$env:GOOS = "linux"
$env:GOARCH = "arm64"
go build -o fiberweb-linux-arm64
输出文件：fiberweb-linux-arm64，适用于 ARM64 架构的 Linux（如 Raspberry Pi）。