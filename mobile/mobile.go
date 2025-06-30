package mobile

import (
	"rush"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/mobile"
)

const (
	screenWidth  = 160
	screenHeight = 80
)

func init() {
	// yourgame.Game must implement ebiten.Game interface.
	// For more details, see
	// * https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2#Game

	rush.LoadAssets()

	ebiten.SetWindowSize(screenWidth*5, screenHeight*5)
	ebiten.SetWindowTitle("Rush Out the Tunnel")
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

// Dummy is a dummy exported function.
//
// gomobile doesn't compile a package that doesn't include any exported function.
// Dummy forces gomobile to compile this package.
func Dummy() {}
