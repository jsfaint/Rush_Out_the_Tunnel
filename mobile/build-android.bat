@echo off
echo ========================================
echo Rush Android APK һ�������ű�
echo ========================================

echo.
echo ���� 1: ���� rush.aar...
ebitenmobile bind -target android -javapkg net.emsky.rush -o rush.aar .
if %ERRORLEVEL% NEQ 0 (
    echo ����: ���� rush.aar ʧ��
    pause
    exit /b 1
)

echo.
echo ���� 2: ���� rush.aar �� Android ��Ŀ...
if exist "rush.aar" (
    copy "rush.aar" "android\app\libs\"
    echo rush.aar �Ѹ��Ƶ� android\app\libs\
) else (
    echo ����: δ�ҵ� rush.aar �ļ�
    pause
    exit /b 1
)

echo.
echo ���� 3: ���� Android APK...
cd android
call build-apk.bat

echo.
echo ========================================
echo ������ɣ�
echo ========================================
echo.
echo ��������ɹ���APK �ļ�λ��:
echo android\app\build\outputs\apk\debug\app-debug.apk
echo.
echo Ҫ��װ���豸��������:
echo adb install android\app\build\outputs\apk\debug\app-debug.apk
echo.

pause