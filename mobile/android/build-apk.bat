@echo off
echo ��ʼ���� Rush APK...

REM ����Ƿ���� rush.aar
if not exist "app\libs\rush.aar" (
    echo ����: δ�ҵ� rush.aar �ļ�
    echo �������� ebitenmobile ���� rush.aar �ļ�
    echo ����: ebitenmobile bind -target android -javapkg net.emsky.rush -o rush.aar .
    pause
    exit /b 1
)

echo ����֮ǰ�Ĺ���...
call gradlew clean

echo ���� APK...
call gradlew assembleDebug

if %ERRORLEVEL% EQU 0 (
    echo.
    echo �����ɹ���
    echo APK �ļ�λ��: app\build\outputs\apk\debug\app-debug.apk
    echo.
    echo Ҫ��װ���豸��������:
    echo adb install app\build\outputs\apk\debug\app-debug.apk
) else (
    echo.
    echo ����ʧ�ܣ����������Ϣ
)

pause