@echo off
:: UTF-8
chcp 65001 > nul
cd %0/../

set OUTPUT=Setup_ExampleApp_v1.1
set INPUT=linux-installer
set VERSION=
:: Emulate the make cmdline for the linux builder on windows, namely parameters:
::    C:\> make VERSION=1.1 OUTPUT=Setup_ExampleApp_v1.1
:: Windows separates cmdline parameters on "=" as well as space... o_O
:: Which leads to the following code to set VERSION or OUTPUT:
set "%1=%2" > nul
set "%3=%4" > nul

set DATA_SRC_DIR=data
set RESOURCE_SRC_DIR=resources
set DATA_DIST_DIR=data_compressed


call :replace_version
call :zip_data
call :rice_append

echo Done
echo -- %OUTPUT%

pause
goto :eof


:replace_version
    if "%VERSION%"=="" goto :eof
    echo Setting VERSION to %VERSION%...
    :: Powershell 5.1:
    powershell -nologo -noprofile ^
        -command ^
        "(gc %RESOURCE_SRC_DIR%/config.yml) -replace '  version: [\d.-]+$', '  version: %VERSION%' | &{[String]::Join(""`n"", @($input))} | Out-File -NoNewLine -Encoding UTF8NoBOM %RESOURCE_SRC_DIR%/config.yml"
    :: Powershell 6.0:
    :: "(gc %RESOURCE_SRC_DIR%/config.yml) -replace '  version: [\d.-]+$', '  version: %VERSION%' | &{[String]::Join(""`n"", @($input))} | Out-File -NoNewLine -Encoding UTF8NoBOM %RESOURCE_SRC_DIR%/config.yml"
goto :eof

:zip_data
    echo Zipping data...
    del /s /q %DATA_DIST_DIR% > nul
    mkdir %DATA_DIST_DIR% 2> nul
    powershell -nologo -noprofile ^
        -command ^
        "& {Add-Type -A 'System.IO.Compression.FileSystem'; [IO.Compression.ZipFile]::CreateFromDirectory( '%DATA_SRC_DIR%', '%DATA_DIST_DIR%/data.zip'); }"
goto :eof

:rice_append
    echo Appending data to installer...
    copy /y "%INPUT%" "%OUTPUT%" > nul
    rice append --exec="%OUTPUT%"
goto :eof
