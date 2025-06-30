# Rush Android 项目

这是 "冲出海底隧道" 游戏的 Android 版本。

## 项目结构

```
android/
├── app/                          # 应用模块
│   ├── libs/                     # 第三方库目录
│   │   └── rush.aar             # Ebiten 生成的 Android 库
│   ├── src/main/
│   │   ├── java/net/emsky/rush/ # Java 源代码
│   │   ├── res/                 # 资源文件
│   │   └── AndroidManifest.xml  # 应用清单
│   └── build.gradle             # 应用级构建配置
├── build.gradle                 # 项目级构建配置
├── settings.gradle              # 项目设置
├── gradle.properties            # Gradle 属性
├── build-apk.bat               # 构建脚本
└── README.md                   # 本文件
```

## 构建步骤

### 1. 生成 rush.aar

首先需要从 Go 项目生成 Android 库文件：

```bash
cd mobile
ebitenmobile bind -target android -javapkg net.emsky.rush -o rush.aar .
```

### 2. 复制 rush.aar

将生成的 `rush.aar` 文件复制到 `android/app/libs/` 目录：

```bash
copy mobile\rush.aar android\app\libs\
```

### 3. 构建 APK

在 `android` 目录下运行构建脚本：

```bash
cd android
build-apk.bat
```

或者手动运行：

```bash
cd android
gradlew assembleDebug
```

### 4. 安装 APK

构建成功后，APK 文件位于 `app/build/outputs/apk/debug/app-debug.apk`。

使用 ADB 安装到设备：

```bash
adb install app\build\outputs\apk\debug\app-debug.apk
```

## 开发环境要求

- Android Studio 或 Android SDK
- Java 8 或更高版本
- Gradle 8.0 或更高版本
- Android SDK API 21 或更高版本

## 注意事项

1. 游戏设置为横屏模式
2. 应用会请求存储权限以保存游戏数据
3. 如果遇到 native 库问题，请检查 `jniLibs` 目录结构
4. MainActivity 中的类名和方法名可能需要根据实际的 rush.aar 内容进行调整

## 故障排除

### 常见问题

1. **找不到 rush.aar**

   - 确保已运行 ebitenmobile 命令生成 AAR 文件
   - 检查文件路径是否正确

2. **Native 库错误**

   - 检查 `jniLibs` 目录是否包含正确的 `.so` 文件
   - 确保 AAR 文件包含所有必要的 native 库

3. **构建失败**
   - 检查 Android SDK 版本
   - 确保 Gradle 版本兼容
   - 查看详细的错误日志

## 自定义配置

### 修改应用信息

编辑 `app/build.gradle` 中的以下字段：

- `applicationId`: 应用包名
- `versionCode`: 版本号
- `versionName`: 版本名称

### 修改权限

编辑 `app/src/main/AndroidManifest.xml` 添加或移除权限。

### 修改主题

编辑 `app/src/main/res/values/themes.xml` 自定义应用主题。
