package rush

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math"
	"math/rand"
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

//go:embed assets/images
var assetsFS embed.FS

const (
	screenWidth  = 160
	screenHeight = 80

	fontSize = 9
)

// Game State
type GameState int

const (
	StateTitle GameState = iota
	StateCountdown
	StateGame
	StateHelp
	StateAbout
	StateWin
	StateGameOver
	StateNameInput
	StatePause
	StateExitConfirm
	StateHighScores
	StateHighScoresThenGame // 新增：排行榜后自动进入游戏
)

var (
	submarineImage     *ebiten.Image
	titleImage         *ebiten.Image
	gameoverImage      *ebiten.Image
	winImage           *ebiten.Image
	coinImage          *ebiten.Image
	bombImage          *ebiten.Image
	handDrawnFontImage *ebiten.Image
	tunnelWallColor    = color.RGBA{139, 69, 19, 255}  // SaddleBrown
	backgroundColor    = color.RGBA{70, 130, 180, 255} // SteelBlue
)

// 排行榜数据结构
const highScoreFilePath = "highscores.json"

type HighScore struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
}

var highScores [5]HighScore

type Player struct {
	x, y float64
	vy   float64 // Vertical velocity
}

type Tunnel struct {
	x      float64
	topY   float64
	height float64
	width  float64
}

type Collectible struct {
	image *ebiten.Image
	x, y  float64
	w, h  int
}

type Game struct {
	state           GameState
	player          *Player
	tunnels         []*Tunnel
	collectibles    []*Collectible
	distance        int
	score           int
	bombs           int
	isBombing       bool
	bombTimer       int
	countdownTimer  int
	tunnelHeight    float64
	tunnelTopY      float64
	slope           int
	upButtonRect    image.Rectangle
	bombButtonRect  image.Rectangle
	menuChoice      int
	menuButtonRects []image.Rectangle

	// 新增：消息提示
	message           string
	messageTimer      int    // 显示剩余帧数
	nameInput         string // 玩家输入的名字
	gameOverAnimFrame int    // GameOver动画帧
	winAnimFrame      int    // Win动画帧

	// 提示语相关
	tips      []string
	tipTimer  int
	curTipIdx int

	explosionFrame int
	explosionDone  bool

	// 新增：名字输入相关状态
	nameInputCursorX   int                 // 字符网格光标X位置 (0-12)
	nameInputCursorY   int                 // 字符网格光标Y位置 (0-4)
	nameInputPosition  int                 // 当前输入位置 (0-7)
	nameInputCharGrid  [][]string          // 字符网格
	nameInputGridRects [][]image.Rectangle // 字符网格的矩形区域

	// 新增：Erase/End红框高亮timer
	eraseBoxHighlightTimer int
	endBoxHighlightTimer   int

	// 新增：原版道具生成相关变量
	nextItem int // 下一个道具生成距离
	thisItem int // 当前道具生成距离
	itemPt   int // 道具指针
}

const (
	eraseBoxX1 = 110
	eraseBoxY1 = 40
	eraseBoxX2 = 151
	eraseBoxY2 = 58

	endBoxX1 = 110
	endBoxY1 = 58
	endBoxX2 = 151
	endBoxY2 = 76
)

// 全局退出标志
var shouldExitApp bool

// SetExitFlag 设置退出标志
func SetExitFlag(exit bool) {
	shouldExitApp = exit
	log.Printf("Exit flag set to: %v", exit)
}

// ShouldExit 检查是否应该退出
func ShouldExit() bool {
	return shouldExitApp
}

func NewGame() *Game {
	_ = loadHighScores() // 启动时加载排行榜
	g := &Game{}
	g.reset() // reset is called first
	g.state = StateTitle
	// Buttons are initialized once, not on every reset
	g.menuButtonRects = []image.Rectangle{
		image.Rect(122, 8, 122+34, 8+9),   // New Game
		image.Rect(122, 23, 122+34, 23+9), // Help
		image.Rect(122, 38, 122+34, 38+9), // About
		image.Rect(122, 53, 122+34, 53+9), // Exit
	}

	// 初始化名字输入字符网格
	g.nameInputCharGrid = [][]string{
		{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M"},
		{"N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"},
		{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m"},
		{"n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z"},
		{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", " ", "", ""},
	}

	// 初始化字符网格矩形区域
	g.nameInputGridRects = make([][]image.Rectangle, 5)
	for y := 0; y < 5; y++ {
		g.nameInputGridRects[y] = make([]image.Rectangle, 13)
		for x := 0; x < 13; x++ {
			// 跳过第5行的最后两个空位置
			if y == 4 && x >= 11 {
				continue
			}
			// 字符网格位置：从(2,29)开始，每个字符8x8像素
			gridX := 2 + x*8
			gridY := 29 + y*8
			g.nameInputGridRects[y][x] = image.Rect(gridX, gridY, gridX+8, gridY+8)
		}
	}

	return g
}

// drawHandDrawnText 使用手绘字体渲染文本
func drawHandDrawnText(screen *ebiten.Image, str string, x, y int, clr color.Color) {
	if handDrawnFontImage == nil {
		return
	}

	lines := strings.Split(str, "\n")
	for lineIdx, line := range lines {
		lineY := y + lineIdx*10 // 行间距10像素

		for charIdx, char := range line {
			char = char - 4
			if char < 0 {
				char = char + 127
			}

			charX := x + charIdx*8 // 字符间距9像素（8像素字符+1像素间距）
			charY := lineY

			// 计算字符在字体图像中的位置
			fontY := int(char) * 8

			// 创建字符子图像
			charRect := image.Rect(0, fontY, 8, fontY+8)
			charSubImage := handDrawnFontImage.SubImage(charRect)

			// 绘制字符
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(charX), float64(charY))

			// 应用颜色
			op.ColorScale.ScaleWithColor(clr)

			screen.DrawImage(charSubImage.(*ebiten.Image), op)
		}
	}
}

func (g *Game) reset() {
	g.player = &Player{
		x:  screenWidth / 4,
		y:  screenHeight / 2,
		vy: 0,
	}
	g.tunnels = []*Tunnel{}
	g.collectibles = []*Collectible{}
	g.distance = 0
	g.score = 0
	g.bombs = 3
	g.isBombing = false
	g.bombTimer = 0
	g.countdownTimer = 180 // 3 seconds at 60 FPS
	g.tunnelHeight = 50
	g.tunnelTopY = 15
	g.slope = 0
	g.player.y = g.tunnelTopY + g.tunnelHeight/2

	// 新增：初始化原版道具生成变量
	g.nextItem = (rand.Intn(5) + 1) * 32
	g.thisItem = 0
	g.itemPt = 0

	// Define button position and size
	buttonSize := 40
	margin := 5
	g.bombButtonRect = image.Rect(margin, screenHeight-buttonSize-margin, margin+buttonSize, screenHeight-margin)
	g.upButtonRect = image.Rect(screenWidth-buttonSize-margin, screenHeight-buttonSize-margin, screenWidth-margin, screenHeight-margin)

	g.tips = []string{
		"I want a GF!  ", "Be careful~   ", "Take it easy~ ", "A red fish!   ",
		"henhenhahi!   ", "QQ:171290999~ ", "QQ:68862232~  ", "I like NDS!   ",
		"Have no money~", "A diamond!!!  ", "What's this?  ", "Up!Up!Up!!!   ",
		"Foolish man!  ", "I'll come back", "Don't hit me! ", "We'll be eat! ",
		"Sunshine~~~   ", "lalalalala~~  ", "Elephant~~    ", "You big nose! ",
		"A lovely girl~", "Clever Anson~ ", "Handsome JAY~ ", "Take my soul~ ",
		"I love NBA!   ", "A good game~  ", "Good ball!    ", "Lucky!        ",
		"To rush out!  ", "NC_TOOLS!!    ", "I love 6502~  ",
	}
	g.tipTimer = 0
	g.curTipIdx = 0

	// 重置名字输入状态
	g.nameInputCursorX = 0
	g.nameInputCursorY = 0
	g.nameInputPosition = 0
}

func (g *Game) spawnTunnel(x float64) {
	g.tunnels = append(g.tunnels, &Tunnel{
		x:      x,
		topY:   g.tunnelTopY,
		height: g.tunnelHeight,
		width:  10,
	})
}

// updateTitle 处理标题界面输入与菜单选择
func (g *Game) updateTitle() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		g.menuChoice = (g.menuChoice + 1) % 5
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		g.menuChoice--
		if g.menuChoice < 0 {
			g.menuChoice = 4
		}
	}
	// Check for mouse click on menu items
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		for i, r := range g.menuButtonRects {
			if (image.Point{x, y}).In(r) {
				g.menuChoice = i
				if err := g.selectMenuItem(); err != nil {
					return err
				}
			}
		}
	}
	// Check for touch click on menu items
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		x, y := ebiten.TouchPosition(id)
		for i, r := range g.menuButtonRects {
			if (image.Point{x, y}).In(r) {
				g.menuChoice = i
				if err := g.selectMenuItem(); err != nil {
					return err
				}
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if err := g.selectMenuItem(); err != nil {
			return err
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.state = StateExitConfirm
		return nil
	}
	return nil
}

// updateCountdown 处理倒计时逻辑
func (g *Game) updateCountdown() error {
	if g.countdownTimer > 0 {
		g.countdownTimer--
	}
	if g.countdownTimer <= 0 {
		g.state = StateGame
	}
	return nil
}

// updateGame 处理游戏主循环逻辑
func (g *Game) updateGame() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
		g.state = StatePause
		g.showMessage("Paused", 60)
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.state = StateExitConfirm
		return nil
	}
	g.updateGameLogic()
	if g.state == StateGameOver {
		g.explosionFrame = 0
		g.explosionDone = false
	}
	return nil
}

// updateHelp 处理帮助界面输入
func (g *Game) updateHelp() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || len(inpututil.AppendJustPressedTouchIDs(nil)) > 0 {
		g.state = StateTitle
	}
	return nil
}

// updateAbout 处理关于界面输入
func (g *Game) updateAbout() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || len(inpututil.AppendJustPressedTouchIDs(nil)) > 0 {
		g.state = StateTitle
	}
	return nil
}

// updateWin 处理胜利界面输入
func (g *Game) updateWin() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || len(inpututil.AppendJustPressedTouchIDs(nil)) > 0 {
		g.state = StateTitle
	}
	return nil
}

func (g *Game) pressBackspace(x, y int) bool {
	if x < eraseBoxX1 || x > eraseBoxX2 || y < eraseBoxY1 || y > eraseBoxY2 {
		return false
	}

	if g.nameInputPosition > 0 {
		g.nameInputPosition--
		if g.nameInputPosition < len(g.nameInput) {
			g.nameInput = g.nameInput[:g.nameInputPosition]
			g.eraseBoxHighlightTimer = 8
		}
	}

	return true
}

func (g *Game) pressEnd(x, y int) bool {
	if x < endBoxX1 || x > endBoxX2 || y < endBoxY1 || y > endBoxY2 {
		return false
	}

	g.endBoxHighlightTimer = 8
	if len(g.nameInput) > 0 {
		g.insertHighScore(g.nameInput, g.score)
		saveHighScores()
		g.state = StateHighScores
	} else {
		g.insertHighScore("Player", g.score)
		saveHighScores()
		g.state = StateHighScores
	}
	return true
}

// updateNameInput 处理玩家名字输入 - 基于原版GetName实现
func (g *Game) updateNameInput() error {
	// 处理方向键导航
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		g.nameInputCursorY--
		if g.nameInputCursorY < 0 {
			g.nameInputCursorY = 4
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		g.nameInputCursorY++
		if g.nameInputCursorY > 4 {
			g.nameInputCursorY = 0
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		g.nameInputCursorX--
		if g.nameInputCursorY == 4 {
			// 第5行只有11个字符（0-9和空格）
			if g.nameInputCursorX < 0 {
				g.nameInputCursorX = 10
			}
		} else {
			if g.nameInputCursorX < 0 {
				g.nameInputCursorX = 12
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		g.nameInputCursorX++
		if g.nameInputCursorY == 4 {
			// 第5行只有11个字符
			if g.nameInputCursorX > 10 {
				g.nameInputCursorX = 0
			}
		} else {
			if g.nameInputCursorX > 12 {
				g.nameInputCursorX = 0
			}
		}
	}

	// 处理鼠标点击
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		// 检查Erase红框
		if g.pressBackspace(x, y) {
			return nil
		}
		// 检查End红框
		if g.pressEnd(x, y) {
			return nil
		}
		g.handleNameInputClick(x, y)
	}

	// 处理触摸输入
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		x, y := ebiten.TouchPosition(id)
		if x >= 110 && x < 150 && y >= 50 && y < 58 {
			g.pressBackspace(x, y)
			return nil
		}
		if x >= 110 && x < 150 && y >= 68 && y < 76 {
			g.pressEnd(x, y)
			return nil
		}
		g.handleNameInputClick(x, y)
	}

	// 处理字符输入（Enter键）
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		g.inputSelectedChar()
	}

	// 处理删除（Backspace键）
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		g.pressBackspace(120, 40)
	}

	// 处理确认输入（空格键结束）
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.pressEnd(120, 60)
	}

	return nil
}

// handleNameInputClick 处理名字输入界面的点击事件
func (g *Game) handleNameInputClick(x, y int) {
	// 检查是否点击了字符网格
	for gridY := 0; gridY < 5; gridY++ {
		for gridX := 0; gridX < 13; gridX++ {
			// 跳过第5行的空位置
			if gridY == 4 && gridX >= 11 {
				continue
			}

			if (image.Point{x, y}).In(g.nameInputGridRects[gridY][gridX]) {
				g.nameInputCursorX = gridX
				g.nameInputCursorY = gridY
				g.inputSelectedChar()
				return
			}
		}
	}
}

// inputSelectedChar 输入当前选中的字符
func (g *Game) inputSelectedChar() {
	if g.nameInputPosition >= 8 {
		return // 最多8个字符
	}

	// 获取当前光标位置的字符
	if g.nameInputCursorY < len(g.nameInputCharGrid) && g.nameInputCursorX < len(g.nameInputCharGrid[g.nameInputCursorY]) {
		char := g.nameInputCharGrid[g.nameInputCursorY][g.nameInputCursorX]
		if char != "" {
			// 如果是空格键，结束输入
			if char == " " {
				if len(g.nameInput) > 0 {
					g.insertHighScore(g.nameInput, g.score)
					saveHighScores()
					g.state = StateHighScores
				} else {
					g.insertHighScore("Player", g.score)
					saveHighScores()
					g.state = StateHighScores
				}
				return
			}

			// 添加字符到输入缓冲区
			if g.nameInputPosition >= len(g.nameInput) {
				g.nameInput += char
			} else {
				// 在指定位置插入字符
				g.nameInput = g.nameInput[:g.nameInputPosition] + char + g.nameInput[g.nameInputPosition:]
			}
			g.nameInputPosition++
		}
	}
}

// updatePause 处理暂停界面输入
func (g *Game) updatePause() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
		g.state = StateGame
		g.showMessage("Resume", 60)
	}
	return nil
}

// updateExitConfirm 处理退出确认界面输入
func (g *Game) updateExitConfirm() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyY) {
		// 设置退出标志而不是直接返回 ebiten.Termination
		// 在 Android 中，ebiten.Termination 不会关闭应用
		// 需要通过 MainActivity 来处理退出
		SetExitFlag(true)
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyN) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.state = StateTitle
	}
	return nil
}

// updateHighScores 处理高分榜界面输入
func (g *Game) updateHighScores() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) ||
		len(inpututil.AppendJustPressedTouchIDs(nil)) > 0 {
		g.state = StateTitle
	}
	return nil
}

// updateGameOver 处理游戏结束界面输入和爆炸动画
func (g *Game) updateGameOver() error {
	if !g.explosionDone {
		g.explosionFrame++
		if g.explosionFrame > 30 {
			g.explosionDone = true
		}
		return nil
	}
	if g.isHighScore(g.score) {
		g.nameInput = ""
		g.nameInputCursorX = 0
		g.nameInputCursorY = 0
		g.nameInputPosition = 0
		g.state = StateNameInput
	} else if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) ||
		len(inpututil.AppendJustPressedTouchIDs(nil)) > 0 {
		g.state = StateTitle
	}
	return nil
}

// updateHighScoresThenGame 处理高分榜后自动进入游戏
func (g *Game) updateHighScoresThenGame() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) ||
		len(inpututil.AppendJustPressedTouchIDs(nil)) > 0 {
		g.reset()
		g.state = StateCountdown
		return nil
	}
	return nil
}

// updateGameLogic 保留原有 updateGame 的主逻辑部分
func (g *Game) updateGameLogic() {
	if g.isBombing {
		g.bombTimer--
		if g.bombTimer <= 0 {
			g.isBombing = false
			g.tunnels = []*Tunnel{}
			g.collectibles = []*Collectible{}
		}
		return
	}
	isPressingBomb := inpututil.IsKeyJustPressed(ebiten.KeyX)
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if g.bombButtonRect.Min.X <= x && x < g.bombButtonRect.Max.X && g.bombButtonRect.Min.Y <= y && y < g.bombButtonRect.Max.Y {
			isPressingBomb = true
		}
	}
	if !isPressingBomb {
		for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
			x, y := ebiten.TouchPosition(id)
			if g.bombButtonRect.Min.X <= x && x < g.bombButtonRect.Max.X && g.bombButtonRect.Min.Y <= y && y < g.bombButtonRect.Max.Y {
				isPressingBomb = true
				break
			}
		}
	}
	if isPressingBomb && g.bombs > 0 && !g.isBombing {
		g.bombs--
		g.isBombing = true
		g.bombTimer = 15
		return
	}
	if isPressingBomb && g.bombs == 0 {
	}
	g.distance++
	if g.distance%40 == 0 {
		g.score++
	}
	if g.distance >= 4000 {
		g.winAnimFrame = 0
		g.showMessage("Win", 60)
		g.state = StateWin
		return
	}
	if g.distance%10 == 0 {
		g.slope = rand.Intn(3)
	}
	if g.distance%200 == 0 && g.tunnelHeight > 20 {
		g.tunnelHeight--
	}
	if g.slope == 0 && g.tunnelTopY > 10 {
		g.tunnelTopY--
	}
	if g.slope == 2 && g.tunnelTopY < screenHeight-g.tunnelHeight-10 {
		g.tunnelTopY++
	}

	// 新增：每帧动态生成隧道（与原版一致）
	// 每帧在屏幕右侧生成新的隧道段，使用最新的 tunnelTopY 和 tunnelHeight
	g.spawnTunnel(159)

	// 每帧将所有隧道段左移1像素
	for _, t := range g.tunnels {
		t.x -= 1.0
	}

	// 移除超出屏幕左侧的隧道段
	remainingTunnels := g.tunnels[:0]
	for _, t := range g.tunnels {
		if t.x+t.width > 0 {
			remainingTunnels = append(remainingTunnels, t)
		}
	}
	g.tunnels = remainingTunnels

	// 新增：原版道具生成逻辑
	if g.distance <= 3840 {
		if g.distance-g.thisItem == g.nextItem {
			// 检查是否有空闲的道具槽位
			hasEmptySlot := false
			for _, c := range g.collectibles {
				if c == nil {
					hasEmptySlot = true
					break
				}
			}
			if !hasEmptySlot && len(g.collectibles) < 5 {
				hasEmptySlot = true
			}

			if hasEmptySlot {
				// 生成金币
				coinY := g.tunnelTopY + float64(rand.Intn(int(g.tunnelHeight)-10))
				g.collectibles = append(g.collectibles, &Collectible{
					image: coinImage,
					x:     157,
					y:     coinY,
					w:     coinImage.Bounds().Dx(),
					h:     coinImage.Bounds().Dy(),
				})
			}
			g.thisItem = g.distance
			g.nextItem = (rand.Intn(5) + 1) * 32
		}
	}

	isPressingUp := ebiten.IsKeyPressed(ebiten.KeyUp)
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if g.upButtonRect.Min.X <= x && x < g.upButtonRect.Max.X && g.upButtonRect.Min.Y <= y && y < g.upButtonRect.Max.Y {
			isPressingUp = true
		}
	}
	if !isPressingUp {
		for _, id := range ebiten.TouchIDs() {
			x, y := ebiten.TouchPosition(id)
			if g.upButtonRect.Min.X <= x && x < g.upButtonRect.Max.X && g.upButtonRect.Min.Y <= y && y < g.upButtonRect.Max.Y {
				isPressingUp = true
				break
			}
		}
	}
	if isPressingUp {
		g.player.vy -= 0.2
	} else {
		g.player.vy += 0.1
	}
	g.player.y += g.player.vy
	if g.player.vy > 1.0 {
		g.player.vy = 1.0
	}
	if g.player.vy < -1.0 {
		g.player.vy = -1.0
	}
	for _, c := range g.collectibles {
		c.x -= 1.0
	}

	// 检查玩家是否撞到顶部或底部
	if g.player.y < 0 || int(g.player.y)+4 > screenHeight {
		g.state = StateGameOver
		return
	}

	playerRect := image.Rect(int(g.player.x), int(g.player.y), int(g.player.x)+8, int(g.player.y)+4)
	for _, t := range g.tunnels {
		topRect := image.Rect(int(t.x), 0, int(t.x+t.width), int(t.topY))
		bottomRect := image.Rect(int(t.x), int(t.topY+t.height), int(t.x+t.width), screenHeight)
		if playerRect.Overlaps(topRect) || playerRect.Overlaps(bottomRect) {
			g.state = StateGameOver
			return
		}
	}

	remainingCollectibles := g.collectibles[:0]
	for _, c := range g.collectibles {
		collectibleRect := image.Rect(int(c.x), int(c.y), int(c.x)+c.w, int(c.y)+c.h)
		if playerRect.Overlaps(collectibleRect) {
			g.score += 5
			g.showMessage("获得金币！", 30)
			continue
		} else if c.x+float64(c.w) > 0 {
			remainingCollectibles = append(remainingCollectibles, c)
		}
	}
	g.collectibles = remainingCollectibles
	g.tipTimer++
	if g.tipTimer%200 == 0 {
		g.curTipIdx = rand.Intn(len(g.tips))
		g.showMessage(g.tips[g.curTipIdx], 60)
	}
}

func (g *Game) selectMenuItem() error {
	switch g.menuChoice {
	case 0: // New
		g.reset()
		g.state = StateHighScoresThenGame
	case 1: // Help
		g.state = StateHelp
	case 2: // About
		g.state = StateAbout
	case 3: // Exit
		// 设置退出标志而不是直接返回 ebiten.Termination
		SetExitFlag(true)
		return nil
	}
	return nil
}

func (g *Game) Update() error {
	switch g.state {
	case StateTitle:
		return g.updateTitle()
	case StateCountdown:
		return g.updateCountdown()
	case StateGame:
		return g.updateGame()
	case StateHelp:
		return g.updateHelp()
	case StateAbout:
		return g.updateAbout()
	case StateWin:
		return g.updateWin()
	case StateNameInput:
		return g.updateNameInput()
	case StatePause:
		return g.updatePause()
	case StateExitConfirm:
		return g.updateExitConfirm()
	case StateHighScores:
		return g.updateHighScores()
	case StateGameOver:
		return g.updateGameOver()
	case StateHighScoresThenGame:
		return g.updateHighScoresThenGame()
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	switch g.state {
	case StateTitle:
		g.drawTitle(screen)
	case StateCountdown:
		g.drawCountdown(screen)
	case StateGame:
		g.drawGame(screen)
	case StateHelp:
		g.drawHelp(screen)
	case StateAbout:
		g.drawAbout(screen)
	case StateWin:
		g.drawWin(screen)
	case StateGameOver:
		g.drawGameOver(screen)
	case StateNameInput:
		g.drawNameInput(screen)
	case StatePause:
		g.drawPause(screen)
	case StateExitConfirm:
		g.drawExitConfirm(screen)
	case StateHighScores, StateHighScoresThenGame:
		g.drawHighScores(screen)
	}

	// 消息提示统一绘制
	if g.messageTimer > 0 {
		drawHandDrawnText(screen, g.message, 40, 55, color.RGBA{0, 0, 0, 255})
		g.messageTimer--
	}
}

func (g *Game) drawHighScores(screen *ebiten.Image) {
	screen.Fill(color.White)

	// 显示当前高分榜
	title := "TOP 5 SCORES"
	drawHandDrawnText(screen, title, 30, 10, color.RGBA{0, 0, 0, 255})

	for i, hs := range highScores {
		name := hs.Name
		if name == "" {
			name = "---"
		}
		colorName := color.RGBA{0, 0, 128, 255}
		colorScore := color.RGBA{128, 0, 0, 255}
		if g.score == hs.Score && g.nameInput != "" && name == g.nameInput {
			colorName = color.RGBA{255, 0, 0, 255}
			colorScore = color.RGBA{255, 0, 0, 255}
		}
		scoreStr := fmt.Sprintf("%d", hs.Score)
		drawHandDrawnText(screen, fmt.Sprintf("%d. %s", i+1, name), 30, 20+11*i, colorName)
		drawHandDrawnText(screen, scoreStr, 110, 20+11*i, colorScore)
	}
}

func (g *Game) drawTitle(screen *ebiten.Image) {
	screen.Fill(color.White)

	if titleImage != nil {
		op := &ebiten.DrawImageOptions{}
		screen.DrawImage(titleImage, op)
	} else {
		// 如果标题图像加载失败，显示文本标题
		ebitenutil.DebugPrint(screen, "RUSH OUT THE TUNNEL\n\nAssets failed to load!\nTitle image is nil")
	}

	// Draw menu selector
	// Adjusted Y and Height to better center the highlight on the text.
	selectorY := float64(8 + g.menuChoice*15)
	selectorColor := color.RGBA{R: 70, G: 130, B: 180, A: 128} // Semi-transparent SteelBlue
	ebitenutil.DrawRect(screen, 122, selectorY, 34, 9, selectorColor)
}

func (g *Game) drawCountdown(screen *ebiten.Image) {
	g.drawGameScene(screen)
	g.drawGameHUD(screen)

	countdownNum := int(math.Ceil(float64(g.countdownTimer) / 60.0))
	if countdownNum > 0 {
		var textStr string
		switch countdownNum {
		case 3:
			textStr = "3"
		case 2:
			textStr = "2"
		case 1:
			textStr = "1"
		}

		// Create a temporary offscreen image to draw the text on.
		textImgWidth := 8
		textImgHeight := 16
		textImg := ebiten.NewImage(textImgWidth, textImgHeight)

		drawHandDrawnText(textImg, textStr, 0, 0, color.White)

		// Now, draw the temporary image onto the main screen, centered.
		op := &ebiten.DrawImageOptions{}
		textX := (screenWidth - textImgWidth) / 2
		textY := (screenHeight - textImgHeight) / 2
		op.GeoM.Translate(float64(textX), float64(textY))
		screen.DrawImage(textImg, op)
	}
}

func (g *Game) drawGameScene(screen *ebiten.Image) {
	screen.Fill(backgroundColor)

	for _, t := range g.tunnels {
		ebitenutil.DrawRect(screen, t.x, 0, t.width, t.topY, tunnelWallColor)
		ebitenutil.DrawRect(screen, t.x, t.topY+t.height, t.width, screenHeight-(t.topY+t.height), tunnelWallColor)
	}

	// Draw Collectibles
	for _, c := range g.collectibles {
		if c.image != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(c.x, c.y)
			screen.DrawImage(c.image, op)
		}
	}

	// Draw Player
	if submarineImage != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(g.player.x, g.player.y)
		screen.DrawImage(submarineImage, op)
	} else {
		// 提供更详细的调试信息
		ebitenutil.DebugPrint(screen, "Assets loading failed!\nSubmarine image is nil")
		// 绘制一个简单的矩形作为玩家
		ebitenutil.DrawRect(screen, g.player.x, g.player.y, 8, 4, color.RGBA{255, 255, 0, 255})
	}
}

func (g *Game) drawGameHUD(screen *ebiten.Image) {
	// Draw HUD text (score)
	scoreText := fmt.Sprintf("SCORE: %d", g.score)
	drawHandDrawnText(screen, scoreText, 5, 5, color.White)

	// Draw bombs
	for i := 0; i < g.bombs; i++ {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(screenWidth-15-i*8), 5)
		screen.DrawImage(bombImage, op)
	}

	// Draw the virtual up button
	buttonColor := color.RGBA{100, 100, 100, 128} // Semi-transparent grey
	ebitenutil.DrawRect(screen, float64(g.upButtonRect.Min.X), float64(g.upButtonRect.Min.Y), float64(g.upButtonRect.Dx()), float64(g.upButtonRect.Dy()), buttonColor)

	// Draw the virtual bomb button
	ebitenutil.DrawRect(screen, float64(g.bombButtonRect.Min.X), float64(g.bombButtonRect.Min.Y), float64(g.bombButtonRect.Dx()), float64(g.bombButtonRect.Dy()), buttonColor)
	if bombImage != nil {
		op := &ebiten.DrawImageOptions{}
		iconW, iconH := bombImage.Size()
		buttonW := g.bombButtonRect.Dx()
		buttonH := g.bombButtonRect.Dy()
		op.GeoM.Translate(float64(g.bombButtonRect.Min.X+(buttonW-iconW)/2), float64(g.bombButtonRect.Min.Y+(buttonH-iconH)/2))
		screen.DrawImage(bombImage, op)
	}
}

func (g *Game) drawGame(screen *ebiten.Image) {
	g.drawGameScene(screen)
	g.drawGameHUD(screen)

	// Draw bomb flash effect
	if g.isBombing {
		screen.Fill(color.White)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func loadImage(path string) *ebiten.Image {
	file, err := assetsFS.Open("assets/images/" + path)
	if err != nil {
		log.Fatalf("failed to open asset %s: %v", path, err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		log.Fatalf("failed to decode image %s: %v", path, err)
	}

	return ebiten.NewImageFromImage(img)
}

func LoadAssets() error {
	// 加载潜艇图像
	submarineBytes, err := assetsFS.ReadFile("assets/images/submarine.png")
	if err != nil {
		return err
	}
	submarineImage, _, err = ebitenutil.NewImageFromReader(bytes.NewReader(submarineBytes))
	if err != nil {
		return err
	}

	// 加载标题图像
	titleBytes, err := assetsFS.ReadFile("assets/images/title.png")
	if err != nil {
		return err
	}
	titleImage, _, err = ebitenutil.NewImageFromReader(bytes.NewReader(titleBytes))
	if err != nil {
		return err
	}

	// 加载游戏结束图像
	gameoverBytes, err := assetsFS.ReadFile("assets/images/gameover.png")
	if err != nil {
		return err
	}
	gameoverImage, _, err = ebitenutil.NewImageFromReader(bytes.NewReader(gameoverBytes))
	if err != nil {
		return err
	}

	// 加载胜利图像
	winBytes, err := assetsFS.ReadFile("assets/images/win.png")
	if err != nil {
		return err
	}
	winImage, _, err = ebitenutil.NewImageFromReader(bytes.NewReader(winBytes))
	if err != nil {
		return err
	}

	// 加载金币图像
	coinBytes, err := assetsFS.ReadFile("assets/images/coin.png")
	if err != nil {
		return err
	}
	coinImage, _, err = ebitenutil.NewImageFromReader(bytes.NewReader(coinBytes))
	if err != nil {
		return err
	}

	// 加载炸弹图像
	bombBytes, err := assetsFS.ReadFile("assets/images/bomb.png")
	if err != nil {
		return err
	}
	bombImage, _, err = ebitenutil.NewImageFromReader(bytes.NewReader(bombBytes))
	if err != nil {
		return err
	}

	// 加载手绘字体图像
	handDrawnFontBytes, err := assetsFS.ReadFile("assets/images/handdrawn_font.png")
	if err != nil {
		return err
	}
	handDrawnFontImage, _, err = ebitenutil.NewImageFromReader(bytes.NewReader(handDrawnFontBytes))
	if err != nil {
		return err
	}

	return nil
}

// 排行榜读写
func loadHighScores() error {
	file, err := os.Open(highScoreFilePath)
	if err != nil {
		// 文件不存在则初始化为空
		for i := range highScores {
			highScores[i] = HighScore{"", 0}
		}
		return nil
	}
	defer file.Close()
	dec := json.NewDecoder(file)
	var loaded []HighScore
	if err := dec.Decode(&loaded); err != nil {
		return err
	}
	for i := range highScores {
		if i < len(loaded) {
			highScores[i] = loaded[i]
		} else {
			highScores[i] = HighScore{"", 0}
		}
	}
	return nil
}

func saveHighScores() error {
	file, err := os.Create(highScoreFilePath)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	return enc.Encode(highScores[:])
}

// 消息提示方法
func (g *Game) showMessage(msg string, duration int) {
	g.message = msg
	g.messageTimer = duration
}

func (g *Game) insertHighScore(name string, score int) {
	// 插入新分数并排序，保留前5名
	inserted := false
	for i := 0; i < len(highScores); i++ {
		if !inserted && score > highScores[i].Score {
			// 后移低分
			copy(highScores[i+1:], highScores[i:len(highScores)-1])
			highScores[i] = HighScore{name, score}
			inserted = true
			break
		}
	}
}

func (g *Game) isHighScore(score int) bool {
	return score > highScores[len(highScores)-1].Score
}

// DrawHelp 绘制帮助界面
func (g *Game) drawHelp(screen *ebiten.Image) {
	screen.Fill(color.White)
	helpText := `Help about the game
Hold [UP] to go up
Release to go down
[Z] Pause the game
[X] Launch the bomb
[ESC] Exit game
Coin Increase score
(:  Have fun!  :)
`
	drawHandDrawnText(screen, helpText, 1, 1, color.Black)
}

// DrawAbout 绘制关于界面
func (g *Game) drawAbout(screen *ebiten.Image) {
	screen.Fill(color.White)
	aboutText := `Rush out the Tunnel
For WQX Lava 12K
Version: 1.0
Design : Anson
Program: Jay
Created: 6/15/2005
Welcome to:
www.emsky.net
`
	drawHandDrawnText(screen, aboutText, 1, 1, color.Black)
}

// DrawWin 绘制胜利界面动画
func (g *Game) drawWin(screen *ebiten.Image) {
	screen.Fill(color.White)
	msg := "You Win"
	letters := g.winAnimFrame/15 + 1
	if letters > len(msg) {
		letters = len(msg)
	}
	drawHandDrawnText(screen, msg[:letters], 50, 40, color.RGBA{0, 128, 0, 255})
	if g.winAnimFrame < len(msg)*15 {
		g.winAnimFrame++
	}
}

// DrawGameOver 绘制游戏结束界面和爆炸动画
func (g *Game) drawGameOver(screen *ebiten.Image) {
	screen.Fill(color.White)
	if !g.explosionDone {
		cx := int(g.player.x) + 4
		cy := int(g.player.y) + 2
		for r := 2; r < g.explosionFrame*2; r += 4 {
			col := color.RGBA{uint8(255 - r*4), uint8(128 + r*2), 0, 255}
			for a := 0.0; a < 2*math.Pi; a += 0.2 {
				x := cx + int(float64(r)*math.Cos(a))
				y := cy + int(float64(r)*math.Sin(a))
				if x >= 0 && x < screenWidth && y >= 0 && y < screenHeight {
					screen.Set(x, y, col)
				}
			}
		}
		return
	}
	// 爆炸动画结束后显示gameover.png
	if gameoverImage != nil {
		op := &ebiten.DrawImageOptions{}
		imgW := gameoverImage.Bounds().Dx()
		imgH := gameoverImage.Bounds().Dy()
		op.GeoM.Translate(float64((screenWidth-imgW)/2), float64((screenHeight-imgH)/2))
		screen.DrawImage(gameoverImage, op)
	}
}

// DrawNameInput 绘制名字输入界面 - 基于原版GetName实现
func (g *Game) drawNameInput(screen *ebiten.Image) {
	screen.Fill(color.White)

	// 绘制标题
	drawHandDrawnText(screen, "Your Name", 2, 5, color.Black)

	// 绘制字符网格
	for y := 0; y < 5; y++ {
		for x := 0; x < 13; x++ {
			// 跳过第5行的空位置
			if y == 4 && x >= 11 {
				continue
			}

			if y < len(g.nameInputCharGrid) && x < len(g.nameInputCharGrid[y]) {
				char := g.nameInputCharGrid[y][x]
				if char != "" {
					// 特殊处理空格字符显示
					displayChar := char
					if char == " " {
						displayChar = "Spc"
					}

					// 绘制字符
					gridX := 2 + x*8
					gridY := 29 + y*8
					drawHandDrawnText(screen, displayChar, gridX, gridY, color.Black)
				}
			}
		}
	}

	// 绘制操作说明（右侧）
	drawHandDrawnText(screen, "Arrow", 110, 5, color.RGBA{128, 128, 128, 255})
	drawHandDrawnText(screen, "Select", 110, 14, color.Black)
	drawHandDrawnText(screen, "CR", 110, 23, color.RGBA{128, 128, 128, 255})
	drawHandDrawnText(screen, "Input", 110, 32, color.Black)
	drawHandDrawnText(screen, "BS", 110, 41, color.RGBA{128, 128, 128, 255})
	drawHandDrawnText(screen, "Erase", 110, 50, color.Black)
	drawHandDrawnText(screen, "Spc", 110, 59, color.RGBA{128, 128, 128, 255})
	drawHandDrawnText(screen, "End", 110, 68, color.Black)

	// 高亮绘制
	if g.eraseBoxHighlightTimer > 0 {
		ebitenutil.DrawRect(screen, eraseBoxX1, eraseBoxY1, eraseBoxX2-eraseBoxX1, eraseBoxY2-eraseBoxY1, color.RGBA{255, 0, 0, 64})
	}
	if g.endBoxHighlightTimer > 0 {
		ebitenutil.DrawRect(screen, endBoxX1, endBoxY1, endBoxX2-endBoxX1, endBoxY2-endBoxY1, color.RGBA{255, 0, 0, 64})
	}
	// 每帧递减timer
	if g.eraseBoxHighlightTimer > 0 {
		g.eraseBoxHighlightTimer--
	}
	if g.endBoxHighlightTimer > 0 {
		g.endBoxHighlightTimer--
	}

	// 绘制当前输入的名字
	displayName := g.nameInput
	if g.nameInputPosition < len(g.nameInput) {
		// 在光标位置插入下划线
		displayName = g.nameInput[:g.nameInputPosition] + "_" + g.nameInput[g.nameInputPosition:]
	} else {
		displayName = g.nameInput + "_"
	}
	drawHandDrawnText(screen, displayName, 2, 15, color.RGBA{0, 0, 255, 255})

	// 绘制选择框高亮
	if g.nameInputCursorY < len(g.nameInputGridRects) && g.nameInputCursorX < len(g.nameInputGridRects[g.nameInputCursorY]) {
		rect := g.nameInputGridRects[g.nameInputCursorY][g.nameInputCursorX]

		// 特殊处理第5行的空格键（跨越多个字符宽度）
		if g.nameInputCursorY == 4 && g.nameInputCursorX == 10 {
			// 空格键跨越3个字符宽度
			ebitenutil.DrawRect(screen, float64(rect.Min.X), float64(rect.Min.Y), 24, 8, color.RGBA{0, 0, 255, 64})
		} else {
			ebitenutil.DrawRect(screen, float64(rect.Min.X), float64(rect.Min.Y), 8, 8, color.RGBA{0, 0, 255, 64})
		}
	}
}

// DrawPause 绘制暂停界面
func (g *Game) drawPause(screen *ebiten.Image) {
	screen.Fill(color.White)
	drawHandDrawnText(screen, "Paused", 60, 40, color.RGBA{255, 0, 0, 255})
}

// DrawExitConfirm 绘制退出确认界面
func (g *Game) drawExitConfirm(screen *ebiten.Image) {
	screen.Fill(color.White)
	drawHandDrawnText(screen, "Exit game? Y/N", 40, 40, color.RGBA{255, 0, 0, 255})
}
