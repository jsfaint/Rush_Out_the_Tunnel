package main

import (
	"log"
	"rush"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	screenWidth  = 160
	screenHeight = 80
)

func main() {
	rush.LoadAssets()

	ebiten.SetWindowSize(screenWidth*5, screenHeight*5)
	ebiten.SetWindowTitle("Rush Out the Tunnel")

	game := rush.NewGame()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
