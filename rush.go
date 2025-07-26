package rush

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math"
	"math/rand"
	"os"
	"runtime"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

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
	tunnelWallColor = color.RGBA{139, 69, 19, 255}  // SaddleBrown
	backgroundColor = color.RGBA{70, 130, 180, 255} // SteelBlue
)

// 排行榜数据结构
const highScoreFilePath = "highscores.json"

type HighScore struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
}

var highScores [5]HighScore
var highScoreStorage HighScoreStorage

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

	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		os.Exit(0)
	}
}

// ShouldExit 检查是否应该退出
func ShouldExit() bool {
	return shouldExitApp
}

func loadAssets() error {
	// 使用新的资源管理器预加载所有资源
	rm := GetResourceManager()
	return rm.PreloadResources()
}

func NewGame() *Game {
	loadAssets()
	ebiten.SetWindowSize(screenWidth*5, screenHeight*5)
	ebiten.SetWindowTitle("Rush Out the Tunnel")

	highScoreStorage = NewHighScoreStorage()
	g := &Game{}
	_ = g.loadHighScores() // 启动时加载排行榜
	g.reset()              // reset is called first
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
	rm := GetResourceManager()
	handDrawnFontImage := rm.GetResource(ResourceHandDrawnFont)
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
	if isKeyJustPressed(ebiten.KeyDown) {
		g.menuChoice = (g.menuChoice + 1) % 5
	}
	if isKeyJustPressed(ebiten.KeyUp) {
		g.menuChoice--
		if g.menuChoice < 0 {
			g.menuChoice = 4
		}
	}
	// Check for mouse click on menu items
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		for i, r := range g.menuButtonRects {
			if isMouseInRect(r) {
				g.menuChoice = i
				if err := g.selectMenuItem(); err != nil {
					return err
				}
			}
		}
	}
	// Check for touch click on menu items
	for i, r := range g.menuButtonRects {
		if isTouchInRect(r) {
			g.menuChoice = i
			if err := g.selectMenuItem(); err != nil {
				return err
			}
		}
	}
	if isKeyJustPressed(ebiten.KeyEnter) {
		if err := g.selectMenuItem(); err != nil {
			return err
		}
	}
	if isKeyJustPressed(ebiten.KeyEscape) {
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
	if g.handlePauseInput() {
		return nil
	}
	if g.handleExitInput() {
		return nil
	}
	g.runGameLogic()
	if g.state == StateGameOver {
		g.explosionFrame = 0
		g.explosionDone = false
	}
	return nil
}

// handlePauseInput 检查并处理暂停输入
func (g *Game) handlePauseInput() bool {
	if isKeyJustPressed(ebiten.KeyZ) {
		g.state = StatePause
		g.showMessage("Paused", 60)
		return true
	}
	return false
}

// handleExitInput 检查并处理退出输入
func (g *Game) handleExitInput() bool {
	if isKeyJustPressed(ebiten.KeyEscape) {
		g.state = StateExitConfirm
		return true
	}
	return false
}

// runGameLogic 执行游戏主逻辑
func (g *Game) runGameLogic() {
	g.updateGameLogic()
}

// updateHelp 处理帮助界面输入
func (g *Game) updateHelp() error {
	if isKeyJustPressed(ebiten.KeyEnter) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || len(inpututil.AppendJustPressedTouchIDs(nil)) > 0 {
		g.state = StateTitle
	}
	return nil
}

// updateAbout 处理关于界面输入
func (g *Game) updateAbout() error {
	if isKeyJustPressed(ebiten.KeyEnter) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || len(inpututil.AppendJustPressedTouchIDs(nil)) > 0 {
		g.state = StateTitle
	}
	return nil
}

// updateWin 处理胜利界面输入
func (g *Game) updateWin() error {
	if isKeyJustPressed(ebiten.KeyEnter) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || len(inpututil.AppendJustPressedTouchIDs(nil)) > 0 {
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
		g.saveHighScores()
		g.state = StateHighScores
	} else {
		g.insertHighScore("Player", g.score)
		g.saveHighScores()
		g.state = StateHighScores
	}
	return true
}

// updateNameInput 处理玩家名字输入 - 基于原版GetName实现
func (g *Game) updateNameInput() error {
	g.handleNameInputNavigation()
	g.handleNameInputMouseClick()
	g.handleNameInputTouch()
	g.handleNameInputEnter()
	g.handleNameInputBackspace()
	g.handleNameInputEnd()
	return nil
}

// handleNameInputNavigation 处理方向键导航
func (g *Game) handleNameInputNavigation() {
	if isKeyJustPressed(ebiten.KeyUp) {
		g.nameInputCursorY--
		if g.nameInputCursorY < 0 {
			g.nameInputCursorY = 4
		}
	}
	if isKeyJustPressed(ebiten.KeyDown) {
		g.nameInputCursorY++
		if g.nameInputCursorY > 4 {
			g.nameInputCursorY = 0
		}
	}
	if isKeyJustPressed(ebiten.KeyLeft) {
		g.nameInputCursorX--
		if g.nameInputCursorY == 4 {
			if g.nameInputCursorX < 0 {
				g.nameInputCursorX = 10
			}
		} else {
			if g.nameInputCursorX < 0 {
				g.nameInputCursorX = 12
			}
		}
	}
	if isKeyJustPressed(ebiten.KeyRight) {
		g.nameInputCursorX++
		if g.nameInputCursorY == 4 {
			if g.nameInputCursorX > 10 {
				g.nameInputCursorX = 0
			}
		} else {
			if g.nameInputCursorX > 12 {
				g.nameInputCursorX = 0
			}
		}
	}
}

// handleNameInputMouseClick 处理鼠标点击
func (g *Game) handleNameInputMouseClick() {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if g.pressBackspace(x, y) {
			return
		}
		if g.pressEnd(x, y) {
			return
		}
		g.handleNameInputRect(isMouseInRect)
	}
}

// handleNameInputTouch 处理触摸输入
func (g *Game) handleNameInputTouch() {
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		x, y := ebiten.TouchPosition(id)
		if x >= 110 && x < 150 && y >= 50 && y < 58 {
			g.pressBackspace(x, y)
			return
		}
		if x >= 110 && x < 150 && y >= 68 && y < 76 {
			g.pressEnd(x, y)
			return
		}
		g.handleNameInputRect(isTouchInRect)
	}
}

// handleNameInputEnter 处理字符输入（Enter键）
func (g *Game) handleNameInputEnter() {
	if isKeyJustPressed(ebiten.KeyEnter) {
		g.inputSelectedChar()
	}
}

// handleNameInputBackspace 处理删除（Backspace键）
func (g *Game) handleNameInputBackspace() {
	if isKeyJustPressed(ebiten.KeyBackspace) {
		g.pressBackspace(120, 40)
	}
}

// handleNameInputEnd 处理确认输入（空格键结束）
func (g *Game) handleNameInputEnd() {
	if isKeyJustPressed(ebiten.KeySpace) {
		g.pressEnd(120, 60)
	}
}

// handleNameInputRect 处理名字输入界面的点击事件
func (g *Game) handleNameInputRect(inRect func(r image.Rectangle) bool) {
	// 检查是否点击了字符网格
	for gridY := 0; gridY < 5; gridY++ {
		for gridX := 0; gridX < 13; gridX++ {
			// 跳过第5行的空位置
			if gridY == 4 && gridX >= 11 {
				continue
			}

			if inRect(g.nameInputGridRects[gridY][gridX]) {
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
					g.saveHighScores()
					g.state = StateHighScores
				} else {
					g.insertHighScore("Player", g.score)
					g.saveHighScores()
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
	if isKeyJustPressed(ebiten.KeyZ) {
		g.state = StateGame
		g.showMessage("Resume", 60)
	}
	return nil
}

// updateExitConfirm 处理退出确认界面输入
func (g *Game) updateExitConfirm() error {
	if isKeyJustPressed(ebiten.KeyY) {
		// 设置退出标志而不是直接返回 ebiten.Termination
		// 在 Android 中，ebiten.Termination 不会关闭应用
		// 需要通过 MainActivity 来处理退出
		SetExitFlag(true)
		return nil
	}
	if isKeyJustPressed(ebiten.KeyN) || isKeyJustPressed(ebiten.KeyEscape) {
		g.state = StateTitle
	}
	return nil
}

// updateHighScores 处理高分榜界面输入
func (g *Game) updateHighScores() error {
	if isKeyJustPressed(ebiten.KeyEnter) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) ||
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
	} else if isKeyJustPressed(ebiten.KeyEnter) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) ||
		len(inpututil.AppendJustPressedTouchIDs(nil)) > 0 {
		g.state = StateTitle
	}
	return nil
}

// updateHighScoresThenGame 处理高分榜后自动进入游戏
func (g *Game) updateHighScoresThenGame() error {
	if isKeyJustPressed(ebiten.KeyEnter) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) ||
		len(inpututil.AppendJustPressedTouchIDs(nil)) > 0 {
		g.reset()
		g.state = StateCountdown
		return nil
	}
	return nil
}

// updateGameLogic 保留原有 updateGame 的主逻辑部分
func (g *Game) updateGameLogic() {
	// 1. 炸弹状态递减与爆炸效果
	if g.isBombing {
		g.updateBombState()
		return
	}

	// 2. 炸弹触发
	g.tryTriggerBomb()
	if g.isBombing {
		return
	}

	// 3. 距离、分数、胜利判定
	g.updateDistanceAndScore()
	if g.state == StateWin {
		return
	}

	// 4. 隧道坡度与高度调整
	g.updateTunnelSlopeAndHeight()

	// 5. 隧道生成与移动
	g.spawnTunnelIfNeeded()
	g.moveTunnels()
	g.removeOffscreenTunnels()

	// 6. 道具生成与移动
	if g.shouldSpawnCollectible() {
		g.spawnCollectible()
	}
	g.moveCollectibles()
	g.removeOffscreenCollectibles()

	// 7. 玩家操作与物理
	isUp := g.isPressingUp()
	g.updatePlayerVelocity(isUp)
	g.clampPlayerVelocity()
	g.updatePlayerPosition()

	// 8. 碰撞检测
	if g.checkPlayerBoundaryCollision() {
		g.state = StateGameOver
		return
	}
	if g.checkPlayerTunnelCollision() {
		g.state = StateGameOver
		return
	}
	g.checkPlayerCollectibleCollision()

	// 9. 消息提示
	g.updateTipMessage()
}

// isPressingBomb 检查当前是否有炸弹触发输入（键盘、鼠标、触摸）
func (g *Game) isPressingBomb() bool {
	if isKeyJustPressed(ebiten.KeyX) {
		return true
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if g.bombButtonRect.Min.X <= x && x < g.bombButtonRect.Max.X && g.bombButtonRect.Min.Y <= y && y < g.bombButtonRect.Max.Y {
			return true
		}
	}
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		x, y := ebiten.TouchPosition(id)
		if g.bombButtonRect.Min.X <= x && x < g.bombButtonRect.Max.X && g.bombButtonRect.Min.Y <= y && y < g.bombButtonRect.Max.Y {
			return true
		}
	}
	return false
}

// isPressingUp 检查当前是否有上升输入（键盘、鼠标、触摸）
func (g *Game) isPressingUp() bool {
	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		return true
	}
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if g.upButtonRect.Min.X <= x && x < g.upButtonRect.Max.X && g.upButtonRect.Min.Y <= y && y < g.upButtonRect.Max.Y {
			return true
		}
	}
	for _, id := range ebiten.TouchIDs() {
		x, y := ebiten.TouchPosition(id)
		if g.upButtonRect.Min.X <= x && x < g.upButtonRect.Max.X && g.upButtonRect.Min.Y <= y && y < g.upButtonRect.Max.Y {
			return true
		}
	}
	return false
}

// updateBombState 处理炸弹状态递减、爆炸效果（清空隧道和道具）
func (g *Game) updateBombState() {
	g.bombTimer--
	if g.bombTimer <= 0 {
		g.isBombing = false
		g.tunnels = []*Tunnel{}
		g.collectibles = []*Collectible{}
	}
}

// tryTriggerBomb 检查是否满足触发炸弹条件，若满足则消耗炸弹并进入爆炸状态
func (g *Game) tryTriggerBomb() {
	if g.isPressingBomb() && g.bombs > 0 && !g.isBombing {
		g.bombs--
		g.isBombing = true
		g.bombTimer = 15
	}
}

// updateDistanceAndScore 距离递增、分数递增，胜利判定
func (g *Game) updateDistanceAndScore() {
	g.distance++
	if g.distance%40 == 0 {
		g.score++
	}
	if g.distance >= 4000 {
		g.winAnimFrame = 0
		g.showMessage("Win", 60)
		g.state = StateWin
	}
}

// updateTunnelSlopeAndHeight 隧道坡度、顶部高度、隧道高度的动态调整
func (g *Game) updateTunnelSlopeAndHeight() {
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
}

// spawnTunnelIfNeeded 判断是否需要生成新隧道段并生成
func (g *Game) spawnTunnelIfNeeded() {
	// 每帧在屏幕右侧生成新的隧道段
	g.spawnTunnel(159)
}

// moveTunnels 所有隧道段左移
func (g *Game) moveTunnels() {
	for _, t := range g.tunnels {
		t.x -= 1.0
	}
}

// removeOffscreenTunnels 移除超出屏幕的隧道段
func (g *Game) removeOffscreenTunnels() {
	remaining := g.tunnels[:0]
	for _, t := range g.tunnels {
		if t.x+t.width > 0 {
			remaining = append(remaining, t)
		}
	}
	g.tunnels = remaining
}

// shouldSpawnCollectible 判断当前帧是否应生成新道具
func (g *Game) shouldSpawnCollectible() bool {
	if g.distance > 3840 {
		return false
	}
	return g.distance-g.thisItem == g.nextItem
}

// spawnCollectible 生成新道具（如金币）
func (g *Game) spawnCollectible() {
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
		rm := GetResourceManager()
		coinImage := rm.GetResource(ResourceCoin)
		if coinImage != nil {
			coinY := g.tunnelTopY + float64(rand.Intn(int(g.tunnelHeight)-10))
			g.collectibles = append(g.collectibles, &Collectible{
				image: coinImage,
				x:     157,
				y:     coinY,
				w:     coinImage.Bounds().Dx(),
				h:     coinImage.Bounds().Dy(),
			})
		}
	}
	g.thisItem = g.distance
	g.nextItem = (rand.Intn(5) + 1) * 32
}

// moveCollectibles 所有道具左移
func (g *Game) moveCollectibles() {
	for _, c := range g.collectibles {
		c.x -= 1.0
	}
}

// removeOffscreenCollectibles 移除超出屏幕的道具
func (g *Game) removeOffscreenCollectibles() {
	remaining := g.collectibles[:0]
	for _, c := range g.collectibles {
		if c.x+float64(c.w) > 0 {
			remaining = append(remaining, c)
		}
	}
	g.collectibles = remaining
}

// updatePlayerVelocity 根据输入更新玩家速度
func (g *Game) updatePlayerVelocity(isUp bool) {
	if isUp {
		g.player.vy -= 0.2
	} else {
		g.player.vy += 0.1
	}
}

// clampPlayerVelocity 限制玩家速度在合理范围
func (g *Game) clampPlayerVelocity() {
	if g.player.vy > 1.0 {
		g.player.vy = 1.0
	}
	if g.player.vy < -1.0 {
		g.player.vy = -1.0
	}
}

// updatePlayerPosition 根据速度更新玩家位置
func (g *Game) updatePlayerPosition() {
	g.player.y += g.player.vy
}

// checkPlayerBoundaryCollision 检查玩家是否撞到上下边界
func (g *Game) checkPlayerBoundaryCollision() bool {
	return g.player.y < 0 || int(g.player.y)+4 > screenHeight
}

// checkPlayerTunnelCollision 检查玩家是否撞到隧道
func (g *Game) checkPlayerTunnelCollision() bool {
	playerRect := image.Rect(int(g.player.x), int(g.player.y), int(g.player.x)+8, int(g.player.y)+4)
	for _, t := range g.tunnels {
		topRect := image.Rect(int(t.x), 0, int(t.x+t.width), int(t.topY))
		bottomRect := image.Rect(int(t.x), int(t.topY+t.height), int(t.x+t.width), screenHeight)
		if playerRect.Overlaps(topRect) || playerRect.Overlaps(bottomRect) {
			return true
		}
	}
	return false
}

// checkPlayerCollectibleCollision 检查玩家是否吃到道具，处理分数和消息
func (g *Game) checkPlayerCollectibleCollision() (collected bool) {
	playerRect := image.Rect(int(g.player.x), int(g.player.y), int(g.player.x)+8, int(g.player.y)+4)
	remaining := g.collectibles[:0]
	collected = false
	for _, c := range g.collectibles {
		collectibleRect := image.Rect(int(c.x), int(c.y), int(c.x)+c.w, int(c.y)+c.h)
		if playerRect.Overlaps(collectibleRect) {
			g.score += 5
			g.showMessage("获得金币！", 30)
			collected = true
			continue
		}
		remaining = append(remaining, c)
	}
	g.collectibles = remaining
	return
}

// updateTipMessage 定时显示提示消息
func (g *Game) updateTipMessage() {
	g.updateTipTimer()
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
		drawMessage(screen, g.message, 40, 55, color.RGBA{0, 0, 0, 255})
	}
	g.updateMessageTimer()
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

	rm := GetResourceManager()
	titleImage := rm.GetResource(ResourceTitle)
	if titleImage != nil {
		op := &ebiten.DrawImageOptions{}
		screen.DrawImage(titleImage, op)
	} else {
		// 如果标题图像加载失败，显示文本标题
		ebitenutil.DebugPrint(screen, "RUSH OUT THE TUNNEL\n\nAssets failed to load!\nTitle image is nil")
	}

	// Draw menu selector
	selectorY := 8 + g.menuChoice*15
	drawSelector(screen, image.Rect(122, selectorY, 122+34, selectorY+9), color.RGBA{R: 70, G: 130, B: 180, A: 128})
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
	rm := GetResourceManager()
	submarineImage := rm.GetResource(ResourceSubmarine)
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
	rm := GetResourceManager()
	bombImage := rm.GetResource(ResourceBomb)
	if bombImage != nil {
		for i := 0; i < g.bombs; i++ {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(screenWidth-15-i*8), 5)
			screen.DrawImage(bombImage, op)
		}
	}

	// Draw the virtual up button
	buttonColor := color.RGBA{100, 100, 100, 128} // Semi-transparent grey
	drawButton(screen, g.upButtonRect, buttonColor, nil)

	// Draw the virtual bomb button
	drawButton(screen, g.bombButtonRect, buttonColor, bombImage)
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

// 排行榜读写
func (g *Game) loadHighScores() error {
	loaded, err := highScoreStorage.Load()
	if err != nil {
		for i := range highScores {
			highScores[i] = HighScore{"", 0}
		}
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

func (g *Game) saveHighScores() error {
	return highScoreStorage.Save(highScores[:])
}

func (g *Game) insertHighScore(name string, score int) {
	inserted := false
	for i := 0; i < len(highScores); i++ {
		if !inserted && score > highScores[i].Score {
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

// showMessage 显示消息
func (g *Game) showMessage(msg string, duration int) {
	g.message = msg
	g.messageTimer = duration
}

// updateMessageTimer 消息倒计时
func (g *Game) updateMessageTimer() {
	if g.messageTimer > 0 {
		g.messageTimer--
	}
}

// showTip 显示提示语
func (g *Game) showTip(tip string, duration int) {
	g.showMessage(tip, duration)
}

// updateTipTimer 提示语定时切换
func (g *Game) updateTipTimer() {
	g.tipTimer++
	if g.tipTimer%200 == 0 {
		g.curTipIdx = rand.Intn(len(g.tips))
		g.showTip(g.tips[g.curTipIdx], 60)
	}
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
	rm := GetResourceManager()
	gameoverImage := rm.GetResource(ResourceGameOver)
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
	g.drawNameInputTitle(screen)
	g.drawNameInputGrid(screen)
	g.drawNameInputInstructions(screen)
	g.drawNameInputHighlights(screen)
	g.drawNameInputDisplayName(screen)
	g.drawNameInputCursor(screen)
}

// drawNameInputTitle 绘制标题
func (g *Game) drawNameInputTitle(screen *ebiten.Image) {
	drawHandDrawnText(screen, "Your Name", 2, 5, color.Black)
}

// drawNameInputGrid 绘制字符网格
func (g *Game) drawNameInputGrid(screen *ebiten.Image) {
	for y := 0; y < 5; y++ {
		for x := 0; x < 13; x++ {
			if y == 4 && x >= 11 {
				continue
			}
			if y < len(g.nameInputCharGrid) && x < len(g.nameInputCharGrid[y]) {
				char := g.nameInputCharGrid[y][x]
				if char != "" {
					displayChar := char
					if char == " " {
						displayChar = "Spc"
					}
					gridX := 2 + x*8
					gridY := 29 + y*8
					drawHandDrawnText(screen, displayChar, gridX, gridY, color.Black)
				}
			}
		}
	}
}

// drawNameInputInstructions 绘制操作说明（右侧）
func (g *Game) drawNameInputInstructions(screen *ebiten.Image) {
	drawHandDrawnText(screen, "Arrow", 110, 5, color.RGBA{128, 128, 128, 255})
	drawHandDrawnText(screen, "Select", 110, 14, color.Black)
	drawHandDrawnText(screen, "CR", 110, 23, color.RGBA{128, 128, 128, 255})
	drawHandDrawnText(screen, "Input", 110, 32, color.Black)
	drawHandDrawnText(screen, "BS", 110, 41, color.RGBA{128, 128, 128, 255})
	drawHandDrawnText(screen, "Erase", 110, 50, color.Black)
	drawHandDrawnText(screen, "Spc", 110, 59, color.RGBA{128, 128, 128, 255})
	drawHandDrawnText(screen, "End", 110, 68, color.Black)
}

// drawNameInputHighlights 高亮绘制（Erase/End红框）
func (g *Game) drawNameInputHighlights(screen *ebiten.Image) {
	if g.eraseBoxHighlightTimer > 0 {
		vector.DrawFilledRect(screen, float32(eraseBoxX1), float32(eraseBoxY1), float32(eraseBoxX2-eraseBoxX1), float32(eraseBoxY2-eraseBoxY1), color.RGBA{255, 0, 0, 64}, false)
	}
	if g.endBoxHighlightTimer > 0 {
		vector.DrawFilledRect(screen, float32(endBoxX1), float32(endBoxY1), float32(endBoxX2-endBoxX1), float32(endBoxY2-endBoxY1), color.RGBA{255, 0, 0, 64}, false)
	}
	if g.eraseBoxHighlightTimer > 0 {
		g.eraseBoxHighlightTimer--
	}
	if g.endBoxHighlightTimer > 0 {
		g.endBoxHighlightTimer--
	}
}

// drawNameInputDisplayName 绘制当前输入的名字
func (g *Game) drawNameInputDisplayName(screen *ebiten.Image) {
	displayName := g.nameInput
	if g.nameInputPosition < len(g.nameInput) {
		displayName = g.nameInput[:g.nameInputPosition] + "_" + g.nameInput[g.nameInputPosition:]
	} else {
		displayName = g.nameInput + "_"
	}
	drawHandDrawnText(screen, displayName, 2, 15, color.RGBA{0, 0, 255, 255})
}

// drawNameInputCursor 绘制选择框高亮
func (g *Game) drawNameInputCursor(screen *ebiten.Image) {
	if g.nameInputCursorY < len(g.nameInputGridRects) && g.nameInputCursorX < len(g.nameInputGridRects[g.nameInputCursorY]) {
		rect := g.nameInputGridRects[g.nameInputCursorY][g.nameInputCursorX]
		if g.nameInputCursorY == 4 && g.nameInputCursorX == 10 {
			vector.DrawFilledRect(screen, float32(rect.Min.X), float32(rect.Min.Y), 24, 8, color.RGBA{0, 0, 255, 64}, false)
		} else {
			vector.DrawFilledRect(screen, float32(rect.Min.X), float32(rect.Min.Y), 8, 8, color.RGBA{0, 0, 255, 64}, false)
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

// 工具函数区：输入检测与绘制相关
// isKeyJustPressed 检查某个键是否刚被按下
func isKeyJustPressed(key ebiten.Key) bool {
	return inpututil.IsKeyJustPressed(key)
}

// isMouseInRect 检查鼠标是否在指定矩形内
func isMouseInRect(r image.Rectangle) bool {
	x, y := ebiten.CursorPosition()
	return (image.Point{x, y}).In(r)
}

// isTouchInRect 检查是否有触摸点在指定矩形内
func isTouchInRect(r image.Rectangle) bool {
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		x, y := ebiten.TouchPosition(id)
		if (image.Point{x, y}).In(r) {
			return true
		}
	}
	return false
}

// drawButton 绘制按钮（带背景色和可选图标）
func drawButton(screen *ebiten.Image, rect image.Rectangle, bgColor color.Color, icon *ebiten.Image) {
	// 绘制背景
	vector.DrawFilledRect(screen, float32(rect.Min.X), float32(rect.Min.Y), float32(rect.Dx()), float32(rect.Dy()), bgColor, false)
	// 绘制图标（居中）
	if icon != nil {
		iconW, iconH := icon.Size()
		buttonW := rect.Dx()
		buttonH := rect.Dy()
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(rect.Min.X+(buttonW-iconW)/2), float64(rect.Min.Y+(buttonH-iconH)/2))
		screen.DrawImage(icon, op)
	}
}

// drawSelector 绘制菜单选择高亮
func drawSelector(screen *ebiten.Image, rect image.Rectangle, selColor color.Color) {
	vector.DrawFilledRect(screen, float32(rect.Min.X), float32(rect.Min.Y), float32(rect.Dx()), float32(rect.Dy()), selColor, false)
}

// drawMessage 绘制消息提示
func drawMessage(screen *ebiten.Image, msg string, x, y int, clr color.Color) {
	drawHandDrawnText(screen, msg, x, y, clr)
}
