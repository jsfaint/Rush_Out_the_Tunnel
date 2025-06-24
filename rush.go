package rush

import (
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
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

//go:embed assets/images
var assetsFS embed.FS

//go:embed assets/fonts/zpix.ttf
var zpixFontData []byte

var chineseFontFace text.Face

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
	submarineImage  *ebiten.Image
	titleImage      *ebiten.Image
	gameoverImage   *ebiten.Image
	winImage        *ebiten.Image
	coinImage       *ebiten.Image
	bombImage       *ebiten.Image
	tunnelWallColor = color.RGBA{139, 69, 19, 255}  // SaddleBrown
	backgroundColor = color.RGBA{70, 130, 180, 255} // SteelBlue
)

// 排行榜数据结构
const highScoreFilePath = "rush-go/highscores.json"

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
	return g
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

	// Define button position and size
	buttonSize := 40
	margin := 5
	g.bombButtonRect = image.Rect(margin, screenHeight-buttonSize-margin, margin+buttonSize, screenHeight-margin)
	g.upButtonRect = image.Rect(screenWidth-buttonSize-margin, screenHeight-buttonSize-margin, screenWidth-margin, screenHeight-margin)

	for x := 0.0; x < screenWidth+10; x += 10 {
		g.spawnTunnel(x)
	}

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
}

func (g *Game) spawnTunnel(x float64) {
	g.tunnels = append(g.tunnels, &Tunnel{
		x:      x,
		topY:   g.tunnelTopY,
		height: g.tunnelHeight,
		width:  10,
	})

	// Occasionally spawn a coin
	if rand.Intn(10) == 0 { // 10% chance
		coinY := g.tunnelTopY + g.tunnelHeight/2 - float64(coinImage.Bounds().Dy()/2)
		g.collectibles = append(g.collectibles, &Collectible{
			image: coinImage,
			x:     x,
			y:     coinY,
			w:     coinImage.Bounds().Dx(),
			h:     coinImage.Bounds().Dy(),
		})
	}
}

func (g *Game) Update() error {
	switch g.state {
	case StateTitle:
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
					// Immediately execute the choice
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
					// Immediately execute the choice
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
	case StateCountdown:
		g.updateCountdown()
	case StateGame:
		if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
			g.state = StatePause
			g.showMessage("暂停中", 60)
			return nil
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.state = StateExitConfirm
			return nil
		}
		g.updateGame()
		if g.state == StateGameOver {
			g.explosionFrame = 0
			g.explosionDone = false
		}
	case StateHelp, StateAbout, StateWin:
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || len(inpututil.AppendJustPressedTouchIDs(nil)) > 0 {
			g.state = StateTitle
		}
		return nil
	case StateNameInput:
		// 处理玩家名字输入
		for _, key := range ebiten.InputChars() {
			if len(g.nameInput) < 8 && ((key >= 'A' && key <= 'Z') || (key >= 'a' && key <= 'z') || (key >= '0' && key <= '9')) {
				g.nameInput += string(key)
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(g.nameInput) > 0 {
			g.nameInput = g.nameInput[:len(g.nameInput)-1]
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && len(g.nameInput) > 0 {
			g.insertHighScore(g.nameInput, g.score)
			saveHighScores()
			g.state = StateHighScores
			return nil
		}
		return nil
	case StatePause:
		if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
			g.state = StateGame
			g.showMessage("继续游戏", 60)
		}
		return nil
	case StateExitConfirm:
		if inpututil.IsKeyJustPressed(ebiten.KeyY) {
			return ebiten.Termination
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyN) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.state = StateTitle
		}
		return nil
	case StateHighScores:
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			g.state = StateTitle
		}
		return nil
	case StateGameOver:
		if !g.explosionDone {
			g.explosionFrame++
			if g.explosionFrame > 30 {
				g.explosionDone = true
			}
			return nil
		}
		// 爆炸动画结束后才响应输入和显示GAME OVER
		if g.isHighScore(g.score) {
			g.nameInput = ""
			g.state = StateNameInput
		}

		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) ||
			len(inpututil.AppendJustPressedTouchIDs(nil)) > 0 {
			g.state = StateTitle
		}
		return nil
	case StateHighScoresThenGame:
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			g.reset()
			g.state = StateCountdown
			return nil
		}
		return nil
	}
	return nil
}

func (g *Game) selectMenuItem() error {
	switch g.menuChoice {
	case 0: // 排行榜
		g.reset()
		g.state = StateHighScoresThenGame
	case 1: // Help
		g.state = StateHelp
	case 2: // About
		g.state = StateAbout
	case 3: // Exit
		return ebiten.Termination
	}
	return nil
}

func (g *Game) updateCountdown() {
	if g.countdownTimer > 0 {
		g.countdownTimer--
	}
	if g.countdownTimer <= 0 {
		g.state = StateGame
	}
}

func (g *Game) updateGame() {
	if g.isBombing {
		g.bombTimer--
		if g.bombTimer <= 0 {
			g.isBombing = false
			// Clear all obstacles after the flash
			g.tunnels = []*Tunnel{}
			g.collectibles = []*Collectible{}
			// Repopulate the screen to continue
			for x := 0.0; x < screenWidth+10; x += 10 {
				g.spawnTunnel(x)
			}
		}
		return // Pause the game during the bomb effect
	}

	isPressingBomb := inpututil.IsKeyJustPressed(ebiten.KeyX)
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if g.bombButtonRect.Min.X <= x && x < g.bombButtonRect.Max.X && g.bombButtonRect.Min.Y <= y && y < g.bombButtonRect.Max.Y {
			isPressingBomb = true
		}
	}
	// Check for touch input on the bomb button
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
		g.bombTimer = 15 // Flash for 1/4 second at 60fps
		g.showMessage("炸弹！", 30)
		return
	}
	if isPressingBomb && g.bombs == 0 {
		g.showMessage("炸弹已用尽", 60)
	}

	g.distance++

	// Score increases with distance
	if g.distance%40 == 0 {
		g.score++
	}

	// Win condition
	if g.distance >= 4000 {
		g.winAnimFrame = 0
		g.showMessage("胜利！", 60)
		g.state = StateWin
		return
	}

	if g.distance%100 == 0 {
		g.slope = rand.Intn(3)
	}
	if g.distance%200 == 0 && g.tunnelHeight > 25 {
		g.tunnelHeight--
	}
	if g.slope == 0 && g.tunnelTopY > 10 {
		g.tunnelTopY--
	}
	if g.slope == 2 && g.tunnelTopY < screenHeight-g.tunnelHeight-10 {
		g.tunnelTopY++
	}

	// Unify input
	isPressingUp := ebiten.IsKeyPressed(ebiten.KeyUp)
	// Check for mouse press on the up button
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if g.upButtonRect.Min.X <= x && x < g.upButtonRect.Max.X && g.upButtonRect.Min.Y <= y && y < g.upButtonRect.Max.Y {
			isPressingUp = true
		}
	}
	// Check for touch press on the up button
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

	for _, t := range g.tunnels {
		t.x -= 1.0
	}

	if len(g.tunnels) > 0 {
		firstTunnel := g.tunnels[0]
		if firstTunnel.x+firstTunnel.width < 0 {
			g.tunnels = g.tunnels[1:]
			lastTunnel := g.tunnels[len(g.tunnels)-1]
			g.spawnTunnel(lastTunnel.x + lastTunnel.width)
		}
	}

	// Move collectibles
	for _, c := range g.collectibles {
		c.x -= 1.0 // Scroll speed
	}

	// Collision detection
	playerRect := image.Rect(int(g.player.x), int(g.player.y), int(g.player.x)+8, int(g.player.y)+4)

	// Tunnel collision
	for _, t := range g.tunnels {
		topRect := image.Rect(int(t.x), 0, int(t.x+t.width), int(t.topY))
		bottomRect := image.Rect(int(t.x), int(t.topY+t.height), int(t.x+t.width), screenHeight)
		if playerRect.Overlaps(topRect) || playerRect.Overlaps(bottomRect) {
			g.state = StateGameOver
			return // Immediately exit update loop after collision
		}
	}

	// Coin collision
	remainingCollectibles := g.collectibles[:0]
	for _, c := range g.collectibles {
		collectibleRect := image.Rect(int(c.x), int(c.y), int(c.x)+c.w, int(c.y)+c.h)
		if playerRect.Overlaps(collectibleRect) {
			g.score += 5 // You got a coin!
			g.showMessage("获得金币！", 30)
			continue
		} else if c.x+float64(c.w) > 0 { // Keep it if it's still on screen
			remainingCollectibles = append(remainingCollectibles, c)
		}
	}
	g.collectibles = remainingCollectibles

	// 定时显示提示语
	g.tipTimer++
	if g.tipTimer%200 == 0 {
		g.curTipIdx = rand.Intn(len(g.tips))
		g.showMessage(g.tips[g.curTipIdx], 60)
	}
}

// drawText 辅助函数，简化 text/v2 的文本绘制
func drawText(screen *ebiten.Image, str string, x, y int, clr color.Color) {
	lines := strings.Split(str, "\n")
	for i, line := range lines {
		dopt := &text.DrawOptions{}
		dopt.GeoM.Translate(float64(x), float64(y+i*(fontSize+1)))
		dopt.ColorScale.ScaleWithColor(clr)
		text.Draw(screen, line, chineseFontFace, dopt)
	}
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
		screen.Fill(color.White)
		helpText := `Help about the game
Hold [UP] to go up
Release to go down
[Z] Pause the game
[X] Launch the bomb
[Esc] Exit game
Coin Increase score
(:  Have fun!  :)
Press Enter to return
`
		drawText(screen, helpText, 5, 0, color.Black)

	case StateAbout:
		screen.Fill(color.White)
		aboutText := `Rush out the Tunnel
For WQX Lava 12K
Version: 1.0
Design : Anson
Program: Jay
Created: 6/15/2005
Welcome to: www.emsky.net
Press Enter to return
`
		drawText(screen, aboutText, 5, 0, color.Black)

	case StateWin:
		screen.Fill(color.White)
		msg := "YOU WIN"
		letters := g.winAnimFrame/15 + 1
		if letters > len(msg) {
			letters = len(msg)
		}
		drawText(screen, msg[:letters], 50, 40, color.RGBA{0, 128, 0, 255})
		if g.winAnimFrame < len(msg)*15 {
			g.winAnimFrame++
		}
		return
	case StateGameOver:
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
		return
	case StateNameInput:
		screen.Fill(color.White)
		prompt := "请输入你的名字 (A-Z, 0-9):"
		drawText(screen, prompt, 20, 30, color.Black)
		drawText(screen, g.nameInput+"_", 20, 50, color.RGBA{0, 0, 255, 255})
		drawText(screen, "按Enter确认", 20, 70, color.Gray{128})

		g.drawHighScores(screen)

		return
	case StatePause:
		screen.Fill(color.White)
		drawText(screen, "暂停中", 60, 40, color.RGBA{255, 0, 0, 255})
		return
	case StateExitConfirm:
		screen.Fill(color.White)
		drawText(screen, "确认退出？Y/N", 40, 40, color.RGBA{255, 0, 0, 255})
		return
	case StateHighScores, StateHighScoresThenGame:
		g.drawHighScores(screen)

		return
	}

	// 消息提示统一绘制
	if g.messageTimer > 0 {
		drawText(screen, g.message, 40, 55, color.RGBA{0, 0, 0, 255})
		g.messageTimer--
	}
}

func (g *Game) drawHighScores(screen *ebiten.Image) {
	screen.Fill(color.White)

	// 显示当前高分榜
	title := "高分榜 Top 5"
	drawText(screen, title, 50, 2, color.RGBA{0, 0, 0, 255})

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
		drawText(screen, fmt.Sprintf("%d. %s", i+1, name), 50, 12+11*i, colorName)
		drawText(screen, scoreStr, 110, 12+11*i, colorScore)
	}

	drawText(screen, "按Enter返回/继续", 45, 68, color.Gray{128})
}

func (g *Game) drawTitle(screen *ebiten.Image) {
	screen.Fill(color.White)
	op := &ebiten.DrawImageOptions{}
	screen.DrawImage(titleImage, op)

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

		// Use DebugPrint to draw the text onto the temporary image.
		ebitenutil.DebugPrint(textImg, textStr)

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
		ebitenutil.DebugPrint(screen, "Loading assets...")
	}
}

func (g *Game) drawGameHUD(screen *ebiten.Image) {
	// Draw HUD text (score)
	scoreText := fmt.Sprintf("Score: %d", g.score)
	drawText(screen, scoreText, 5, 5, color.White)

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

func LoadChineseFont() {
	ft, err := opentype.Parse(zpixFontData)
	if err != nil {
		panic("无法解析中文字体: " + err.Error())
	}
	face, err := opentype.NewFace(ft, &opentype.FaceOptions{
		Size:    fontSize,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		panic("无法创建中文字体Face: " + err.Error())
	}
	chineseFontFace = text.NewGoXFace(face)
}

func LoadAssets() {
	submarineImage = loadImage("submarine.png")
	titleImage = loadImage("title.png")
	gameoverImage = loadImage("gameover.png")
	winImage = loadImage("win.png")
	coinImage = loadImage("coin.png")
	bombImage = loadImage("bomb.png")

	LoadChineseFont()
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
