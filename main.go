package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	screenWidth  = 160
	screenHeight = 80
)

// Game State
type GameState int

const (
	StateTitle GameState = iota
	StateGame
	StateHelp
	StateAbout
	StateWin
	StateGameOver
)

var (
	submarineImage  *ebiten.Image
	titleImage      *ebiten.Image
	gameoverImage   *ebiten.Image
	winImage        *ebiten.Image
	coinImage       *ebiten.Image
	tunnelWallColor = color.RGBA{139, 69, 19, 255}  // SaddleBrown
	backgroundColor = color.RGBA{70, 130, 180, 255} // SteelBlue
)

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
	state        GameState
	player       *Player
	tunnels      []*Tunnel
	collectibles []*Collectible
	distance     int
	score        int
	tunnelHeight float64
	tunnelTopY   float64
	slope        int
	upButtonRect image.Rectangle
	menuChoice   int
}

func (g *Game) reset() {
	g.player = &Player{
		x: screenWidth / 4,
	}
	g.tunnels = []*Tunnel{}
	g.collectibles = []*Collectible{}
	g.distance = 0
	g.score = 0
	g.tunnelHeight = 50
	g.tunnelTopY = 15
	g.player.y = g.tunnelTopY + g.tunnelHeight/2
	g.slope = 1 // Start flat

	// Define button position and size
	buttonSize := 40
	g.upButtonRect = image.Rect(screenWidth-buttonSize-10, screenHeight-buttonSize-10, screenWidth-10, screenHeight-10)

	for x := 0.0; x < screenWidth+10; x += 10 {
		g.spawnTunnel(x)
	}
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
			g.menuChoice = (g.menuChoice + 1) % 4
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
			g.menuChoice--
			if g.menuChoice < 0 {
				g.menuChoice = 3
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			switch g.menuChoice {
			case 0: // New Game
				g.reset()
				g.state = StateGame
			case 1: // Help
				g.state = StateHelp
			case 2: // "With" on screen, means About
				g.state = StateAbout
			case 3: // Exit
				return ebiten.Termination
			}
		}
	case StateGame:
		g.updateGame()
	case StateHelp, StateAbout, StateWin, StateGameOver:
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.state = StateTitle
		}
	}
	return nil
}

func (g *Game) updateGame() {
	g.distance++

	// Score increases with distance
	if g.distance%40 == 0 {
		g.score++
	}

	// Win condition
	if g.distance >= 4000 {
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
	touchIDs := ebiten.TouchIDs()
	if len(touchIDs) > 0 {
		for _, id := range touchIDs {
			x, y := ebiten.TouchPosition(id)
			point := image.Point{
				X: x,
				Y: y,
			}
			if point.In(g.upButtonRect) {
				isPressingUp = true
				break // One touch is enough
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
		} else if c.x+float64(c.w) > 0 { // Keep it if it's still on screen
			remainingCollectibles = append(remainingCollectibles, c)
		}
	}
	g.collectibles = remainingCollectibles
}

func (g *Game) Draw(screen *ebiten.Image) {
	switch g.state {
	case StateTitle:
		screen.Fill(color.White)
		op := &ebiten.DrawImageOptions{}
		screen.DrawImage(titleImage, op)

		// Draw menu selector
		// Adjusted Y and Height to better center the highlight on the text.
		selectorY := float64(8 + g.menuChoice*15)
		selectorColor := color.RGBA{R: 70, G: 130, B: 180, A: 128} // Semi-transparent SteelBlue
		ebitenutil.DrawRect(screen, 122, selectorY, 34, 9, selectorColor)

	case StateGame:
		g.drawGame(screen)

	case StateHelp:
		screen.Fill(color.White)
		helpText := `
Hold [UP] to go up
Release to go down

[P] Pause the game (Not yet implemented)
[X] Launch bomb (Not yet implemented)
[Esc] Exit to Title (From Game)

Collect coins to increase score!

(: Have fun! :)


Press Enter to return
`
		ebitenutil.DebugPrint(screen, helpText)

	case StateAbout:
		screen.Fill(color.White)
		aboutText := `
Rush out the Tunnel

Version: 2.0 (Go Remake)

Original Design: Anson
Original Program: Jay
Remake by: John (AI PM) & You!

Created: 6/15/2005


Press Enter to return
`
		ebitenutil.DebugPrint(screen, aboutText)

	case StateWin:
		screen.Fill(color.White)
		op := &ebiten.DrawImageOptions{}
		screen.DrawImage(winImage, op)
	case StateGameOver:
		screen.Fill(color.White)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(screenWidth/2-gameoverImage.Bounds().Dx()/2), 0)
		screen.DrawImage(gameoverImage, op)
	}
}

func (g *Game) drawGame(screen *ebiten.Image) {
	screen.Fill(backgroundColor)

	for _, t := range g.tunnels {
		ebitenutil.DrawRect(screen, t.x, 0, t.width, t.topY, tunnelWallColor)
		ebitenutil.DrawRect(screen, t.x, t.topY+t.height, t.width, screenHeight-(t.topY+t.height), tunnelWallColor)
	}

	// Draw Collectibles
	for _, c := range g.collectibles {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(c.x, c.y)
		screen.DrawImage(c.image, op)
	}

	// Draw the virtual button
	buttonColor := color.RGBA{100, 100, 100, 128} // Semi-transparent grey
	ebitenutil.DrawRect(screen, float64(g.upButtonRect.Min.X), float64(g.upButtonRect.Min.Y), float64(g.upButtonRect.Dx()), float64(g.upButtonRect.Dy()), buttonColor)

	// Draw Player
	if submarineImage != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(g.player.x, g.player.y)
		screen.DrawImage(submarineImage, op)
	} else {
		ebitenutil.DebugPrint(screen, "Loading assets...")
	}

	// Draw Score
	scoreText := fmt.Sprintf("Score: %d", g.score)
	ebitenutil.DebugPrint(screen, scoreText)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	rand.Seed(time.Now().UnixNano())

	loadAssets()

	ebiten.SetWindowSize(screenWidth*5, screenHeight*5)
	ebiten.SetWindowTitle("Rush Out the Tunnel")

	game := &Game{}
	game.reset()
	game.state = StateTitle

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

func loadAssets() {
	var err error
	submarineImage, _, err = ebitenutil.NewImageFromFile("assets/images/submarine.png")
	if err != nil {
		fmt.Println("An asset is missing, running extractor...")
		_extractAssets()
	}

	submarineImage, _, err = ebitenutil.NewImageFromFile("assets/images/submarine.png")
	if err != nil {
		log.Fatalf("failed to load submarine image: %v", err)
	}
	titleImage, _, err = ebitenutil.NewImageFromFile("assets/images/title.png")
	if err != nil {
		log.Fatalf("failed to load title image: %v", err)
	}
	gameoverImage, _, err = ebitenutil.NewImageFromFile("assets/images/gameover.png")
	if err != nil {
		log.Fatalf("failed to load gameover image: %v", err)
	}
	winImage, _, err = ebitenutil.NewImageFromFile("assets/images/win.png")
	if err != nil {
		log.Fatalf("failed to load win image: %v", err)
	}
	coinImage, _, err = ebitenutil.NewImageFromFile("assets/images/coin.png")
	if err != nil {
		log.Fatalf("failed to load coin image: %v", err)
	}
}

// _extractAssets is the moved asset extraction logic.
// It is kept for reference but not called on every run.
func _extractAssets() {
	sourcePath := "../Rush_Out_the_Tunnel_For_Lava1.txt"
	content, err := ioutil.ReadFile(sourcePath)
	if err != nil {
		log.Printf("Failed to read source file for asset extraction: %v", err)
		return
	}
	sourceCode := string(content)
	extractAndSaveAsset(sourceCode, "coin", 3, 5, "assets/images/coin.png")
	extractAndSaveAsset(sourceCode, "bomb", 6, 3, "assets/images/bomb.png")
	extractAndSaveAsset(sourceCode, "submarine", 8, 4, "assets/images/submarine.png")
	extractAndSaveAsset(sourceCode, "title", 160, 80, "assets/images/title.png")
	extractAndSaveAsset(sourceCode, "gameover", 126, 80, "assets/images/gameover.png")
	extractAndSaveAsset(sourceCode, "win", 160, 80, "assets/images/win.png")
}

func extractAndSaveAsset(source, name string, width, height int, outputPath string) {
	// Check if file already exists
	if _, err := os.Stat(outputPath); err == nil {
		fmt.Printf("Asset '%s' already exists. Skipping.\n", name)
		return
	}

	fmt.Printf("Extracting asset: %s\n", name)
	re := regexp.MustCompile(fmt.Sprintf(`char\s+%s\[\d*\]\s*=\s*\{([^}]+)\};`, name))
	matches := re.FindStringSubmatch(source)

	if len(matches) < 2 {
		fmt.Printf("Could not find asset data for '%s'\n", name)
		return
	}

	byteString := strings.TrimSpace(matches[1])
	byteParts := strings.Split(byteString, ",")

	data := make([]byte, 0)
	for _, part := range byteParts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "0x") {
			val, err := strconv.ParseUint(part[2:], 16, 8)
			if err != nil {
				fmt.Printf("Failed to parse hex value '%s': %v\n", part, err)
				continue
			}
			data = append(data, byte(val))
		}
	}

	if len(data) == 0 {
		fmt.Printf("No data extracted for asset '%s'\n", name)
		return
	}

	palette := color.Palette{color.Transparent, color.Black}
	img := image.NewPaletted(image.Rect(0, 0, width, height), palette)

	pixelDataIndex := 0
	for y := 0; y < height; y++ {
		for x_byte := 0; x_byte < (width+7)/8; x_byte++ {
			if pixelDataIndex >= len(data) {
				break
			}
			byteVal := data[pixelDataIndex]
			pixelDataIndex++
			for x_bit := 0; x_bit < 8; x_bit++ {
				x := x_byte*8 + x_bit
				if x >= width {
					continue
				}
				if (byteVal>>(7-x_bit))&1 == 1 {
					img.SetColorIndex(x, y, 1)
				}
			}
		}
	}

	f, err := os.Create(outputPath)
	if err != nil {
		fmt.Printf("Failed to create file '%s': %v\n", outputPath, err)
		return
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		fmt.Printf("Failed to encode PNG for '%s': %v\n", name, err)
		return
	}
	fmt.Printf("Successfully saved %s\n", outputPath)
}
