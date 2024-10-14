
Set-StrictMode -Version Latest

$scriptPath = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location (Join-Path $scriptPath '..')

$binDir = "bin"
if (-not (Test-Path -Path $binDir)) {
  New-Item -ItemType Directory -Path $binDir
}

$gitCommit = git rev-list -1 HEAD
$gitTag = git rev-list --tags --max-count=1

# Add an array of common OSes and architectures
$targets = @(
  # "darwin:amd64",
  # "darwin:arm64",
  # "linux:amd64",
  # "linux:arm64",
  # "linux:386",
  # "freebsd:amd64",
  # "freebsd:arm64",
  "windows:amd64"
)

Get-ChildItem -Directory "cmd" | ForEach-Object {
  $b = $_.Name
  foreach ($target in $targets) {
    $os_arch = $target -split ":"
    $os = $os_arch[0]
    $arch = $os_arch[1]

    $binaryName = "$b-$os-$arch"
    if ($os -eq "windows") {
      $binaryName += ".exe"
    }

    Write-Host -NoNewline "Building $b for $os/$arch..."

    if (-not [string]::IsNullOrEmpty($gitTag)) {
      $gitVersion = git describe --tags $gitTag
      $env:GOOS = $os
      $env:GOARCH = $arch
      & go build -mod=vendor -ldflags="-s -w -X main.GitCommit=$gitCommit -X main.GitVersion=$gitVersion" -o "bin/$binaryName" -a "cmd/$b/main.go"
    }
    else {
      $env:GOOS = $os
      $env:GOARCH = $arch
      & go build -mod=vendor -ldflags="-s -w -X main.GitCommit=$gitCommit" -o "bin/$binaryName" -a "cmd/$b/main.go"
    }

    Write-Host "done"
  }
}
