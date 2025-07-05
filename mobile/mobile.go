package mobile

import (
	"rush"

	"github.com/hajimehoshi/ebiten/v2/mobile"
)

const (
	screenWidth  = 160
	screenHeight = 80
)

// StartGame 导出函数，供 Java 层在设置好高分目录后启动游戏
//
//export StartGame
func StartGame() {
	mobile.SetGame(rush.NewGame())
}

// ShouldExit 导出函数，供 Android 检查是否需要退出应用
//
//export ShouldExit
func ShouldExit() bool {
	return rush.ShouldExit()
}

// SetExitFlag 导出函数，供游戏内部设置退出标志
//
//export SetExitFlag
func SetExitFlag(exit bool) {
	rush.SetExitFlag(exit)
}

// SetHighScoreDir 导出函数，供 Java 层设置高分存储目录
//
//export SetHighScoreDir
func SetHighScoreDir(path string) {
	rush.SetHighScoreDir(path)
}

// Dummy is a dummy exported function.
//
// gomobile doesn't compile a package that doesn't include any exported function.
// Dummy forces gomobile to compile this package.
func Dummy() {}
