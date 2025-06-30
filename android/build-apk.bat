@echo off
echo 开始构建 Rush APK...

REM 检查是否存在 rush.aar
if not exist "app\libs\rush.aar" (
    echo 错误: 未找到 rush.aar 文件
    echo 请先运行 ebitenmobile 生成 rush.aar 文件
    echo 命令: ebitenmobile bind -target android -javapkg net.emsky.rush -o rush.aar .
    pause
    exit /b 1
)

echo 清理之前的构建...
call gradlew clean

echo 构建 APK...
call gradlew assembleDebug

if %ERRORLEVEL% EQU 0 (
    echo.
    echo 构建成功！
    echo APK 文件位置: app\build\outputs\apk\debug\app-debug.apk
    echo.
    echo 要安装到设备，请运行:
    echo adb install app\build\outputs\apk\debug\app-debug.apk
) else (
    echo.
    echo 构建失败，请检查错误信息
)

pause