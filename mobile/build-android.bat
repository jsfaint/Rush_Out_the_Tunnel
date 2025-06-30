@echo off
echo ========================================
echo Rush Android APK 一键构建脚本
echo ========================================

echo.
echo 步骤 1: 生成 rush.aar...
ebitenmobile bind -target android -javapkg net.emsky.rush -o rush.aar .
if %ERRORLEVEL% NEQ 0 (
    echo 错误: 生成 rush.aar 失败
    pause
    exit /b 1
)

echo.
echo 步骤 2: 复制 rush.aar 到 Android 项目...
if exist "rush.aar" (
    copy "rush.aar" "android\app\libs\"
    echo rush.aar 已复制到 android\app\libs\
) else (
    echo 错误: 未找到 rush.aar 文件
    pause
    exit /b 1
)

echo.
echo 步骤 3: 构建 Android APK...
cd android
call build-apk.bat

echo.
echo ========================================
echo 构建完成！
echo ========================================
echo.
echo 如果构建成功，APK 文件位于:
echo android\app\build\outputs\apk\debug\app-debug.apk
echo.
echo 要安装到设备，请运行:
echo adb install android\app\build\outputs\apk\debug\app-debug.apk
echo.

pause