@echo off
rem SPDX-License-Identifier: MIT
rem Copyright (C) 2019-2022 WireGuard LLC. All Rights Reserved.

setlocal enabledelayedexpansion
set BUILDDIR=%~dp0
set PATH=%BUILDDIR%.deps\llvm-mingw-20231128-ucrt-x86_64\bin;%BUILDDIR%.deps\go\bin;%BUILDDIR%.deps;%PATH%
set PATHEXT=.exe
cd /d %BUILDDIR% || exit /b 1

if exist .deps\prepared goto :render
:installdeps
	rmdir /s /q .deps 2> NUL
	mkdir .deps || goto :error
	cd .deps || goto :error
	call :download go.zip https://go.dev/dl/go1.22.0.windows-amd64.zip 78b3158fe3aa358e0b6c9f26ecd338f9a11441e88bc434ae2e9f0ca2b0cc4dd3 || goto :error
	call :download llvm-mingw-20231128-ucrt-x86_64.zip https://github.com/mstorsjo/llvm-mingw/releases/download/20231128/llvm-mingw-20231128-ucrt-x86_64.zip 7a344dafa6942de2c1f4643b3eb5c5ce5317fbab671a887e4d39f326b331798f || goto :error
	rem Mirror of https://imagemagick.org/download/binaries/ImageMagick-7.0.8-42-portable-Q16-x64.zip
	call :download imagemagick.zip https://download.wireguard.com/windows-toolchain/distfiles/ImageMagick-7.0.8-42-portable-Q16-x64.zip 584e069f56456ce7dde40220948ff9568ac810688c892c5dfb7f6db902aa05aa "convert.exe colors.xml delegates.xml" || goto :error
	rem Mirror of https://sourceforge.net/projects/ezwinports/files/make-4.2.1-without-guile-w32-bin.zip
	call :download make.zip https://download.wireguard.com/windows-toolchain/distfiles/make-4.2.1-without-guile-w32-bin.zip 30641be9602712be76212b99df7209f4f8f518ba764cf564262bc9d6e4047cc7 "--strip-components 1 bin" || goto :error
	call :download amneziawg-tools.zip https://github.com/amnezia-vpn/amneziawg-tools/archive/e42813caac0b1846810f20435b699234ef5f7ead.zip 7c1e6097e3e8c278dd90dde33a78a54119f5b1ee143ce9fb553700eb7d1001a3 "--exclude wg-quick --strip-components 1" || goto :error
	call :download wintun.zip https://www.wintun.net/builds/wintun-0.14.1.zip 07c256185d6ee3652e09fa55c0b673e2624b565e02c4b9091c79ca7d2f24ef51 || goto :error
	copy /y NUL prepared > NUL || goto :error
	cd .. || goto :error

:render
	echo [+] Rendering icons
	for %%a in ("ui\icon\*.svg") do convert -background none "%%~fa" -define icon:auto-resize="256,192,128,96,64,48,40,32,24,20,16" -compress zip "%%~dpna.ico" || goto :error

:build
	for /f "tokens=3" %%a in ('findstr /r "Number.*=.*[0-9.]*" .\version\version.go') do set WIREGUARD_VERSION=%%a
	set WIREGUARD_VERSION=%WIREGUARD_VERSION:"=%
	for /f "tokens=1-4" %%a in ("%WIREGUARD_VERSION:.= % 0 0 0") do set WIREGUARD_VERSION_ARRAY=%%a,%%b,%%c,%%d
	set GOOS=windows
	set GOARM=7
	set GOPATH=%BUILDDIR%.deps\gopath
	set GOROOT=%BUILDDIR%.deps\go
	if "%GoGenerate%"=="yes" (
		echo [+] Regenerating files
		go generate ./... || exit /b 1
	)
	call :build_plat x86 i686 386 || goto :error
	call :build_plat amd64 x86_64 amd64 || goto :error
	call :build_plat arm64 aarch64 arm64 || goto :error

:sign
	if exist .\sign.bat call .\sign.bat
	if "%SigningCertificate%"=="" goto :success
	if "%TimestampServer%"=="" goto :success
	echo [+] Signing
        signtool sign /sha1 "%SigningCertificate%" /fd sha256 /tr "%TimestampServer%" /td sha256 /d WireGuard x86\amneziawg.exe x86\awg.exe amd64\amneziawg.exe amd64\awg.exe arm64\amneziawg.exe arm64\awg.exe || goto :error

:success
	echo [+] Success. Launch amneziawg.exe.
	exit /b 0

:download
	echo [+] Downloading %1
	curl -#fLo %1 %2 || exit /b 1
	echo [+] Verifying %1
	for /f %%a in ('CertUtil -hashfile %1 SHA256 ^| findstr /r "^[0-9a-f]*$"') do if not "%%a"=="%~3" exit /b 1
	echo [+] Extracting %1
	tar -xf %1 %~4 || exit /b 1
	echo [+] Cleaning up %1
	del %1 || exit /b 1
	goto :eof

:build_plat
	set GOARCH=%~3
	mkdir %1 >NUL 2>&1
	echo [+] Assembling resources %1
	%~2-w64-mingw32-windres -DWIREGUARD_VERSION_ARRAY=%WIREGUARD_VERSION_ARRAY% -DWIREGUARD_VERSION_STR=%WIREGUARD_VERSION% -i resources.rc -o "resources_%~3.syso" -O coff -c 65001 || exit /b %errorlevel%
	echo [+] Building program %1
	go build -tags load_wgnt_from_rsrc -ldflags="-H windowsgui -s -w" -trimpath -buildvcs=false -v -o "%~1\amneziawg.exe" || exit /b 1
	if not exist "%~1\awg.exe" (
		echo [+] Building command line tools %1
		del .deps\src\*.exe .deps\src\*.o .deps\src\wincompat\*.o .deps\src\wincompat\*.lib 2> NUL
		set LDFLAGS=-s
		make --no-print-directory -C .deps\src PLATFORM=windows CC=%~2-w64-mingw32-gcc WINDRES=%~2-w64-mingw32-windres V=1 RUNSTATEDIR= SYSTEMDUNITDIR= -j%NUMBER_OF_PROCESSORS% || exit /b 1
                move /Y .deps\src\wg.exe "%~1\awg.exe" > NUL || exit /b 1
	)
	if not exist "%~1\wintun.dll" (
		copy /Y ".deps\wintun\bin\%~1\wintun.dll" "%~1\wintun.dll" > NUL || exit /b 1
	)
	goto :eof

:error
	echo [-] Failed with error #%errorlevel%.
	cmd /c exit %errorlevel%
