@echo off
REM a script in PATH that sets up go windows 7 backport;
REM feel free to not bother with this
call win7go

REM msbuild the aurora.dll
msbuild Aurora\Aurora.slnx /p:Configuration=Release

REM ensure goversioninfo is installed ..
go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest

REM setup application manifest
go generate

REM build with subsystem:windows
go build -ldflags="-H windowsgui"
