# 冲出海底隧道（Rush Out the Tunnel）项目详细介绍

## 项目简介

“冲出海底隧道”是一个复刻自 20 多年前的经典游戏的开源项目。该项目由 jsfaint 维护，采用 Go 语言开发，使用 Ebiten 作为游戏引擎，并支持 Android 和 WebAssembly（wasm）平台。游戏尽可能还原了原作的玩法与美术资源，同时针对现代设备（如手机）增加了触摸操作支持。

## 核心玩法

- 玩家控制角色穿越海底隧道，躲避障碍并收集道具以提升分数。
- 游戏包含爆炸动画、胜利动画、排行榜等多种界面和反馈效果。
- 道具收集、碰撞检测、分数统计等核心机制均有实现。
- 支持暂停、帮助、关于、游戏结束、输入玩家名字等丰富界面和交互。

## 主要功能与界面

- **游戏主循环与逻辑**：包括标题界面、倒计时、主玩法、暂停、排行榜、游戏结束、胜利等状态管理。
- **输入支持**：键盘操作（如方向键、Z 键暂停、X 键释放炸弹等），移动端支持触摸操作。
- **排行榜功能**：分数自动保存和载入，玩家可输入自己的名字。
- **提示与消息**：游戏过程中会定时显示提示语和消息。
- **帮助和关于界面**：内置操作说明和项目信息，方便玩家快速上手。

## 部分源码亮点

- 使用 Ebiten 游戏引擎实现跨平台渲染。
- drawHandDrawnText 实现手绘风格文字渲染，提升复古氛围。
- drawWin、drawGameOver、drawHighScores、drawCountdown 等函数分别负责不同游戏状态下的动画和界面绘制。
- 名字输入界面包含字符网格、光标高亮、操作说明等细致交互。
- 高分榜存储路径可在 Android 设备上自定义，确保不同平台兼容性。

## Android 移植结构与构建

- Android 版本通过 EbitenMobile 生成 Go 库（rush.aar），并集成到 Java 项目中。
- 主活动类（MainActivity）负责初始化 Ebiten 游戏和高分榜目录。
- 构建步骤包括生成 aar、复制到 libs、运行构建脚本生成 APK、安装 APK。
- 支持自定义配置、故障排查说明，开发环境要求详见 mobile/android/README.md。

## 典型操作说明（摘自源码 Help 界面）

```
Hold [UP] to go up
Release to go down
[Z] Pause the game
[X] Launch the bomb
[ESC] Exit game
Coin Increase score
(:  Have fun!  :)
```

## 关于界面信息

```
Rush out the Tunnel
For WQX Lava 12K
Version: 1.0
Design : Anson
Program: Jay
Created: 6/15/2005
Welcome to:
www.emsky.net
```

## 技术栈与特色

- 主体代码：Go
- 图形渲染：Ebiten
- 支持平台：PC、Android、WebAssembly
- 美术资源：采用原游戏素材
- 适配移动端：触摸操作、排行榜存储路径定制

## 结语

本项目是对经典游戏的现代复刻，兼容多平台，玩法贴近原作，并针对移动设备做了适配改进。欢迎在 GitHub 参与开发与反馈！
