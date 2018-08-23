package main

import (
	"log"
	"image"
	"image/color"
	"math/rand"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"fmt"
	"time"
)

const (
	WINDOW_WIDTH  = 320
	WINDOW_HEIGHT = 240
	SCALE        = 3
	WINDOW_SIZE   = WINDOW_WIDTH * WINDOW_HEIGHT
)

var (
	lastStatePressedKeyC bool = false
	lastStatePressedKeyP bool = false
	lastStatePressedKeyF bool = false
	lastStatePressedKeyH bool = false

	numberOfHotSpot int = 5
	hotSpots []int
	fixHotSpots bool = false

	pause bool = false
	displayColorMap bool = false
	displayHelp bool = true

	currentColorMapIndex int  = 0
	colorMaps map[string][256]color.RGBA
	currentColorMap [256]color.RGBA
	colorMapLabels []string
	
	buffer1 [WINDOW_WIDTH][WINDOW_HEIGHT]uint8 // hotPower
	buffer2 [WINDOW_WIDTH][WINDOW_HEIGHT]uint8 // hotPower
	imageBuffer *image.RGBA = image.NewRGBA(image.Rect(0, 0, WINDOW_WIDTH, WINDOW_HEIGHT))
)

func update(surface *ebiten.Image) error {
	bindings() // manage control
	
	if !pause {

		if !fixHotSpots {
			initHotSpots()
		}
		for _,x := range hotSpots {
			buffer1[x-1][WINDOW_HEIGHT-1] = 255 // last line maximum power
			buffer1[x  ][WINDOW_HEIGHT-1] = 255 // last line maximum power
			buffer1[x+1][WINDOW_HEIGHT-1] = 255 // last line maximum power
		}

		for x:=1 ; x<WINDOW_WIDTH-1 ; x++ {
			for y:=1 ; y<WINDOW_HEIGHT-1 ; y++ {
				hotness1 := int(buffer1[x+1][y  ])
				hotness2 := int(buffer1[x-1][y  ])
				hotness3 := int(buffer1[x  ][y+1])
				hotness4 := int(buffer1[x  ][y-1])

				newHotness := (hotness1+hotness2+hotness3+hotness4) / 4
				buffer2[x][y-1] = uint8(newHotness)
			}

			// last line
			y := WINDOW_HEIGHT-1
			hotness1 := int(buffer1[x+1][y  ])
			hotness2 := int(buffer1[x-1][y  ])
			hotness4 := int(buffer1[x  ][y-1])

			newHotness := (hotness1+hotness2+hotness4) / 3
			buffer2[x][y-1] = uint8(newHotness)
		}
	}

	//frame skip
	if ebiten.IsDrawingSkipped() {
		return nil
	}

	convertHotnessToImage()

	if displayColorMap {
		drawColorMap()
	}

	// update surface
	surface.ReplacePixels( imageBuffer.Pix )

	buffer1 = buffer2

	if displayHelp {
		// display FPS and other stuff
		ebitenutil.DebugPrint(surface,
			fmt.Sprintf("FPS:%f\n[Up/Down] numberOfHotSpot=%d\n[C] Colors[%d]=%s\n[P]ause [F]ix [H]elp",
				ebiten.CurrentFPS(),
				numberOfHotSpot,
				currentColorMapIndex,
				colorMapLabels[currentColorMapIndex],
		))
	}
	
	return nil
}

func main() {
	initColorMaps()
	initHotSpots()
	
	if err := ebiten.Run(update, WINDOW_WIDTH, WINDOW_HEIGHT, SCALE, "Fire 2"); err != nil {
		log.Fatal(err)
	}
}


func convertHotnessToImage() {
	for x:=0 ; x<WINDOW_WIDTH ; x++ {
		for y:=0 ; y<WINDOW_HEIGHT ; y++ {
			imageBuffer.SetRGBA(x, y, currentColorMap[ buffer1[x][y] ])
		}
	}
}

func drawColorMap() {
	for x:=0 ; x<len(currentColorMap) ; x++ {
		imageBuffer.SetRGBA(x+20, 100, currentColorMap[x])
		imageBuffer.SetRGBA(x+20, 101, currentColorMap[x])
		imageBuffer.SetRGBA(x+20, 102, currentColorMap[x])
		imageBuffer.SetRGBA(x+20, 103, currentColorMap[x])
		imageBuffer.SetRGBA(x+20, 104, currentColorMap[x])
	}
}

func initHotSpots() {
	// draw hot spots
	hotSpots = []int{} // reset hotSpots
	for i:=0 ; i<numberOfHotSpot ; i++ {
		x := rand.Intn(WINDOW_WIDTH)
		if x < 2 {
			x=2
		} else if x > WINDOW_WIDTH-2 {
			x=WINDOW_WIDTH-2
		}
		hotSpots = append(hotSpots, x) // store hotspot
	}
}

func initColorMaps() {
	colorMaps = make(map[string][256]color.RGBA)
	colorMaps["Black_Red_Yellow_White"] = Black_Red_Yellow_White_ColorMap()
	colorMaps["Black_Yellow_White"]     = Black_Yellow_White_ColorMap()

	colorMapLabels = []string{}
	for key,_ := range colorMaps {
		colorMapLabels = append(colorMapLabels, key)
	}
	currentColorMap = colorMaps[ colorMapLabels[currentColorMapIndex] ]
	launchColorMapTimer()
}

func Black_Yellow_White_ColorMap() [256]color.RGBA {
	var colorMap [256]color.RGBA

	j:=0
	for i:=0 ; i<128 ; i++ { // black to yellow
		colorMap[i] = color.RGBA{ R:uint8(j*2), G:uint8(j*2), B:0, A:255 }
		j++
	}

	j=0
	for i:=128 ; i<256 ; i++ { // yellow to white
		colorMap[i] = color.RGBA{ R:255, G:255, B:uint8(j*2), A:255 }
		j++
	}
	return colorMap
}


func Black_Red_Yellow_White_ColorMap() [256]color.RGBA {
	var colorMap [256]color.RGBA

	j:=0
	for i:=0 ; i<64 ; i++ { // from black to RED
		colorMap[i] = color.RGBA{ R:uint8(j*4), G:0, B:0, A:255 }
		j++
	}

	j=0
	for i:=64 ; i<192 ; i++ { // from RED to YELLOW
		colorMap[i] = color.RGBA{ R:255, G:uint8( j*2 ), B:0, A:255 }
		j++
	}

	j=0
	for i:=192 ; i<256 ; i++ { // YELLOW to WHITE
		colorMap[i] = color.RGBA{ R:255, G:255, B:uint8( j*4 ), A:255 }
		j++
	}
	return colorMap
}

func launchColorMapTimer() {
	displayColorMap=true
	timer2 := time.NewTimer(time.Second * 2)
    go func() {
		<-timer2.C
		displayColorMap= false
    }()
}

func bindings() {
	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		numberOfHotSpot++
		if numberOfHotSpot>300 {
			numberOfHotSpot=300
		}
		initHotSpots()
	}

	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		numberOfHotSpot--
		if numberOfHotSpot<0 {
			numberOfHotSpot=0
		}
		initHotSpots()
	}

	if ebiten.IsKeyPressed(ebiten.KeyC) {
		if lastStatePressedKeyC == false {
			currentColorMapIndex++
			if currentColorMapIndex > len(colorMaps)-1 {
				currentColorMapIndex=0
			}
			currentColorMap = colorMaps[ colorMapLabels[currentColorMapIndex] ]
			launchColorMapTimer()
			lastStatePressedKeyC = true
		}
	} else {
		lastStatePressedKeyC = false
	}

	if ebiten.IsKeyPressed(ebiten.KeyP) {
		if lastStatePressedKeyP == false {
			pause = !pause
			lastStatePressedKeyP = true
		}
	} else {
		lastStatePressedKeyP = false
	}

	if ebiten.IsKeyPressed(ebiten.KeyF) {
		if lastStatePressedKeyF == false {
			fixHotSpots = !fixHotSpots
			lastStatePressedKeyF = true
		}
	} else {
		lastStatePressedKeyF = false
	}

	if ebiten.IsKeyPressed(ebiten.KeyH) {
		if lastStatePressedKeyH == false {
			displayHelp = !displayHelp
			lastStatePressedKeyH = true
		}
	} else {
		lastStatePressedKeyH = false
	}
	
}