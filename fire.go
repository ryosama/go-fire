package main

import (
	"log"
	"image"
	"image/color"
	"math"
	"math/rand"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/aquilax/go-perlin"
	"fmt"
	"time"
)

const (
	WINDOW_WIDTH  = 320
	WINDOW_HEIGHT = 240
	SCALE         = 2 		// scale of the window
	WINDOW_SIZE   = WINDOW_WIDTH * WINDOW_HEIGHT
)

var (	
	lastStatePressedKey	 map[string]bool

	numberOfHotSpot int = 152 	// number of hotspots
	firePower int 		= 2 	// velocity of the flames
	hotSpots []int 				// keep trace of the hotspot when they are fixed
	fixHotSpots bool 	= false // flag for fix or not the hotspot

	pause bool = false			// flag for pausing the animation or not
	
	displayColorMap bool = false	// flag for displaying the color map or not
	displayHelp bool = true			// flag for displaying help or not

	colorMaps 			map[string][256]color.RGBA 	// contains all the color maps
	currentColorMap 	[256]color.RGBA 			// current color map displayed
	currentColorMapIndex int  = 0					// index of the current color map displayed
	colorMapLabels 		[]string 					// labels of the differents color maps

	buffer1			[WINDOW_WIDTH * (WINDOW_HEIGHT+1)]uint8 // keep track of the power of each window pixel
	buffer2			[WINDOW_WIDTH * (WINDOW_HEIGHT+1)]uint8 // keep track of the power of each window pixel
	collingBuffer	[WINDOW_WIDTH * (WINDOW_HEIGHT+1)]uint8 // keep track of the coolness of each window pixel
	collingBufferFirstRow int = 0 							// to move the colling buffer up at every frame
	imageBuffer *image.RGBA = image.NewRGBA(image.Rect(0, 0, WINDOW_WIDTH, WINDOW_HEIGHT)) // drawing window
)

func update(surface *ebiten.Image) error {
	bindings() // manage keyboard inputs and mouse inputs
	
	if !pause {

		if !fixHotSpots {
			initHotSpots()
		}
		
		// draw hotspot a the bottom of the screen (3 pixel large)
		for _,x := range hotSpots {
			buffer1[pixelAt(x-1,WINDOW_HEIGHT)] = 255
			buffer1[pixelAt(x  ,WINDOW_HEIGHT)] = 255
			buffer1[pixelAt(x+1,WINDOW_HEIGHT)] = 255
		}

		// for each pixel on the screen, compute the average power the neighbourg pixel
		for x:=1 ; x<WINDOW_WIDTH-1 ; x++ {
			for y:=1 ; y<WINDOW_HEIGHT ; y++ {

				// neighbourg pixel
				hotness1 := int(buffer1[pixelAt(x+1,y  )])
				hotness2 := int(buffer1[pixelAt(x-1,y  )])
				hotness3 := int(buffer1[pixelAt(x  ,y+1)])
				hotness4 := int(buffer1[pixelAt(x  ,y-1)])
				newHotness 	:= float64(hotness1+hotness2+hotness3+hotness4) / 4

				// apply coolness from cooling map
				yCoolness 	:= y + collingBufferFirstRow
				yCoolness 	%= WINDOW_HEIGHT
				coolness 	:= collingBuffer[ pixelAt(x, yCoolness ) ]

				// store new value into buffer2
				newHotness 	= newHotness - float64(coolness)
				if (newHotness < 0) {
					newHotness = 0
				} else if newHotness > 255 {
					newHotness = 255
				}
				buffer2[pixelAt(x,y-1)] = uint8(newHotness)
			}

			// for the last line do the same
			y := WINDOW_HEIGHT
			hotness1 := int(buffer1[pixelAt(x+1,y  )])
			hotness2 := int(buffer1[pixelAt(x-1,y  )])
			hotness4 := int(buffer1[pixelAt(x  ,y-1)])
			newHotness := float64(hotness1+hotness2+hotness4) / 3
			buffer2[pixelAt(x,y-1)] = uint8(newHotness)
		}
	}

	// frame skip
	if ebiten.IsDrawingSkipped() {
		return nil
	}

	// convert the power of a pixel by the corresponding color in the color map
	convertHotnessToImage()

	// display the color map to the screen
	if displayColorMap {
		drawColorMap()
	}

	// update surface
	surface.ReplacePixels( imageBuffer.Pix )

	// move cooling buffer up
	moveCollingBufferUp()

	// swap buffer for next animation
	buffer1 = buffer2

	// display FPS and other stuff
	if displayHelp {
		drawFPS(surface)
	}
	
	return nil
}

func main() {
	initBinding()
	initColorMaps()
	initHotSpots()
	initNoise()

	// infinit loop
	if err := ebiten.Run(update, WINDOW_WIDTH, WINDOW_HEIGHT, SCALE, "Fire 2"); err != nil {
		log.Fatal(err)
	}
}

// convert x,y coordonnate to an index
func pixelAt(x int, y int) int {
	return x + y*WINDOW_WIDTH
}

// convert the power of a pixel by the corresponding color in the color map
func convertHotnessToImage() {
	for x:=0 ; x<WINDOW_WIDTH ; x++ {
		for y:=0 ; y<WINDOW_HEIGHT ; y++ {
			imageBuffer.SetRGBA(x, y, currentColorMap[ buffer1[pixelAt(x,y)] ])
			//imageBuffer.SetRGBA(x, y, currentColorMap[ collingBuffer[pixelAt(x,y)] ])
		}
	}
}

// move the cooling buffer up for better effect
func moveCollingBufferUp() {
	collingBufferFirstRow += firePower
	if collingBufferFirstRow > WINDOW_HEIGHT {
		collingBufferFirstRow = 0 // after one complete screen roll up, reset
	}
}

// display some stuff on the screen
func drawFPS(surface *ebiten.Image) {
	ebitenutil.DebugPrint(surface,
		fmt.Sprintf("FPS:%f\n[Up/Down] Number of flames=%d\n[Left/Right] Fire Power=%d\n[C] Color Map %d=%s\n[P]ause [S]tatic [H]elp",
			ebiten.CurrentFPS(),
			numberOfHotSpot,
			firePower,
			currentColorMapIndex,
			colorMapLabels[currentColorMapIndex],
	))
}

// draw the current color map on the screen (5 pixels large)
func drawColorMap() {
	for x:=0 ; x<len(currentColorMap) ; x++ {
		for y:=0 ; y<5 ; y++ {
			imageBuffer.SetRGBA(x+20, 100+y, currentColorMap[x])
		}
	}
}

func initBinding() {
	lastStatePressedKey = make(map[string]bool)
	lastStatePressedKey["Right"] 	= false
	lastStatePressedKey["Left"]  	= false
	lastStatePressedKey["P"] 	 	= false
	lastStatePressedKey["S"] 	 	= false
	lastStatePressedKey["H"] 	 	= false
	lastStatePressedKey["C"]		= false
}

// the cooling map use Perlin Noise for the generation
func initNoise() {
	p := perlin.NewPerlin(4, 2, 3, int64(rand.Intn(1000)))
	for x := 0.0; x < WINDOW_WIDTH ; x++ {
		for y := 0.0; y < WINDOW_HEIGHT ; y++ {
			noise := p.Noise2D(x/10, y/10) *10
			if noise < 0 {
			 	noise = 0
			}
			// store cooling value
			collingBuffer[pixelAt(int(x),int(y))] = uint8(math.Round(noise)) // fill the colling map with -1 to +1
		}
	}
}

// store the places of each hotspot
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

// store the color maps
func initColorMaps() {
	colorMaps = make(map[string][256]color.RGBA)
	colorMaps["Black_Red_Yellow_White"] = Black_Red_Yellow_White_ColorMap()
	colorMaps["Black_Yellow_White"]     = Black_Yellow_White_ColorMap()
	colorMaps["Black_White"]     		= Black_White_ColorMap()

	colorMapLabels = []string{}
	for key,_ := range colorMaps {
		colorMapLabels = append(colorMapLabels, key)
	}
	currentColorMap = colorMaps[ colorMapLabels[currentColorMapIndex] ]
	launchColorMapTimer()
}

// create the Black -> Yellow -> White  color map
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

// create the Black -> Red -> Yellow -> White  color map
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
	for i:=192 ; i<256 ; i++ { // from YELLOW to WHITE
		colorMap[i] = color.RGBA{ R:255, G:255, B:uint8( j*4 ), A:255 }
		j++
	}
	return colorMap
}

// create the Black -> White  color map
func Black_White_ColorMap() [256]color.RGBA {
	var colorMap [256]color.RGBA
	
	for i:=0 ; i<256 ; i++ { // black to white
		colorMap[i] = color.RGBA{ R:uint8(i), G:uint8(i), B:uint8(i), A:255 }
	}
	return colorMap
}

// hide the color map afert 2 second
func launchColorMapTimer() {
	displayColorMap=true
	timer2 := time.NewTimer(time.Second * 2)
    go func() {
		<-timer2.C
		displayColorMap=false
    }()
}

// manage keyboard and mouse input
func bindings() {
	// if up, increase number of hotspots
	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		numberOfHotSpot++
		if numberOfHotSpot>300 {
			numberOfHotSpot=300
		}
		initHotSpots()
	}

	// if down, decrease number of hotspots
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		numberOfHotSpot--
		if numberOfHotSpot<0 {
			numberOfHotSpot=0
		}
		initHotSpots()
	}
	
	// if right, increase power the flames
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		if lastStatePressedKey["Right"] == false {
			firePower = int(math.Min(float64(firePower+1),5))
			lastStatePressedKey["Right"] = true
		}
	} else {
		lastStatePressedKey["Right"] = false
	}

	// if left, decrease power of the flames
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		if lastStatePressedKey["Left"] == false {
			firePower = int(math.Max(float64(firePower-1),0))
			lastStatePressedKey["Left"] = true
		}
	} else {
		lastStatePressedKey["Left"] = false
	}

	// if C, change tghe current color map for the next one
	if ebiten.IsKeyPressed(ebiten.KeyC) {
		if lastStatePressedKey["C"] == false {
			currentColorMapIndex++
			if currentColorMapIndex > len(colorMaps)-1 {
				currentColorMapIndex=0
			}
			currentColorMap = colorMaps[ colorMapLabels[currentColorMapIndex] ]
			launchColorMapTimer()
			lastStatePressedKey["C"] = true
		}
	} else {
		lastStatePressedKey["C"] = false
	}

	// if P, pause the animation
	if ebiten.IsKeyPressed(ebiten.KeyP) {
		if lastStatePressedKey["P"] == false {
			pause = !pause
			lastStatePressedKey["P"] = true
		}
	} else {
		lastStatePressedKey["P"] = false
	}

	// if S, fix the hotspots place
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		if lastStatePressedKey["S"] == false {
			fixHotSpots = !fixHotSpots
			lastStatePressedKey["S"] = true
		}
	} else {
		lastStatePressedKey["S"] = false
	}

	// if H, toogle help displaying
	if ebiten.IsKeyPressed(ebiten.KeyH) {
		if lastStatePressedKey["H"] == false {
			displayHelp = !displayHelp
			lastStatePressedKey["H"] = true
		}
	} else {
		lastStatePressedKey["H"] = false
	}
	
}