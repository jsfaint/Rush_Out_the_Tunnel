package main

import (
	"log"
	"rush"

	_ "github.com/ebitengine/hideconsole"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	screenWidth  = 160
	screenHeight = 80
)

func main() {
	game := rush.NewGame()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
