package main

import (
	"fmt"
	"github.com/aquilax/go-perlin"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/audio"
	"github.com/hajimehoshi/ebiten/audio/mp3"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/inpututil"
	"image"
	"image/color"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"time"
)

const (
	windowWidth    = 320 // Width of animation
	windowHeight   = 240 // Height of animation
	scale          = 2   // scale of the window
	windowSize     = windowWidth * windowHeight
	colorMapHeight = 5          // height of the preview of the color map currently displayed
	soundFile      = "fire.mp3" // filename of the sound
	maximumPower   = 255        // Power is stock as a uint8, 255 is the max
)

var (
	lastStatePressedKey map[string]bool

	numberOfHotSpot = 152   // number of hotspots
	firePower       = 2     // velocity of the flames
	hotSpots        []int   // keep trace of the hotspot when they are fixed
	fixHotSpots     = false // flag for fix or not the hotspot
	fireHeight      = 0     // hightest flame

	audioPlayer *audio.Player // to control audio

	pause = false // flag for pausing the animation or not

	displayColorMap = false // flag for displaying the color map or not
	displayHelp     = true  // flag for displaying help or not

	colorMaps            map[string][256]color.RGBA // contains all the color maps
	currentColorMap      [256]color.RGBA            // current color map displayed
	currentColorMapIndex int                        // index of the current color map displayed
	colorMapLabels       []string                   // labels of the differents color maps

	buffer1               [windowWidth * (windowHeight + 1)]uint8                      // keep track of the power of each window pixel
	buffer2               [windowWidth * (windowHeight + 1)]uint8                      // keep track of the power of each window pixel
	collingBuffer         [windowWidth * (windowHeight + 1)]uint8                      // keep track of the coolness of each window pixel
	collingBufferFirstRow int                                                          // to move the colling buffer up at every frame
	imageBuffer           = image.NewRGBA(image.Rect(0, 0, windowWidth, windowHeight)) // drawing window
)

func update(surface *ebiten.Image) error {

	if !pause { // if animation isn't paused

		fireHeight = 0 // reset height of hightest flame

		if !fixHotSpots { // if hotspots aren't static
			initHotSpots()
		}

		// draw hotspots
		for _, x := range hotSpots {
			drawHotspot(x)
		}

		// for each pixel on the screen, compute the average power the neighbourg pixel
		for x := 1; x < windowWidth-1; x++ {
			for y := 1; y <= windowHeight; y++ {
				newHotness := averageHotness(x, y)

				// if the power is enought high, consider this as a flame and compute the hightness flame
				if newHotness > 20 {
					if windowHeight-y > fireHeight {
						fireHeight = windowHeight - y
					}
				}

				// store the average hotness into the new buffer
				buffer2[pixelAt(x, y-1)] = newHotness
			}
		}

		// add random sparkles
		addSparkles()

	} // end if game paused

	// manage keyboard inputs and mouse inputs
	bindings()

	// frame skip
	if ebiten.IsDrawingSkipped() {
		return nil
	}

	// convert the power of a pixel by the corresponding color in the color map
	convertHotnessToImage()

	// display the color map to the screen
	drawColorMap()

	// update surface
	surface.ReplacePixels(imageBuffer.Pix)

	// move cooling buffer up
	moveCollingBufferUp()

	// swap buffer for next animation
	buffer1 = buffer2

	// display FPS and other stuff
	drawFPS(surface)

	return nil
}

func main() {
	// ini binding
	lastStatePressedKey = make(map[string]bool)

	initColorMaps()
	initHotSpots()
	initNoise()
	go initSound() // launch sound in another proc
	// infinit loop
	if err := ebiten.Run(update, windowWidth, windowHeight, scale, "Fire 2"); err != nil {
		log.Fatal(err)
	}
}

// convert x,y coordonnate to an index
func pixelAt(x int, y int) int {
	return x + y*windowWidth
}

func averageHotness(x int, y int) uint8 {
	// neighbourg pixel
	newHotness := 0.0

	if y >= windowHeight { // for last line
		y := windowHeight
		hotness1 := int(buffer1[pixelAt(x+1, y)])
		hotness2 := int(buffer1[pixelAt(x-1, y)])
		hotness4 := int(buffer1[pixelAt(x, y-1)])
		newHotness = float64(hotness1+hotness2+hotness4) / 3

	} else { // for other lines on the screen
		hotness1 := int(buffer1[pixelAt(x+1, y)])
		hotness2 := int(buffer1[pixelAt(x-1, y)])
		hotness3 := int(buffer1[pixelAt(x, y+1)])
		hotness4 := int(buffer1[pixelAt(x, y-1)])
		newHotness = float64(hotness1+hotness2+hotness3+hotness4) / 4
	}

	// apply coolness from cooling map
	yCoolness := y + collingBufferFirstRow
	yCoolness %= windowHeight
	coolness := collingBuffer[pixelAt(x, yCoolness)]

	// store new value into buffer2
	newHotness = newHotness - float64(coolness)
	if newHotness < 0 {
		newHotness = 0
	} else if newHotness > maximumPower {
		newHotness = maximumPower
	}
	return uint8(newHotness)
}

// convert the power of a pixel by the corresponding color in the color map
func convertHotnessToImage() {
	for x := 0; x < windowWidth; x++ {
		for y := 0; y < windowHeight; y++ {
			imageBuffer.SetRGBA(x, y, currentColorMap[buffer1[pixelAt(x, y)]])
			//imageBuffer.SetRGBA(x, y, currentColorMap[ collingBuffer[pixelAt(x,y)] ])
		}
	}
}

// move the cooling buffer up for better effect
func moveCollingBufferUp() {
	collingBufferFirstRow += firePower
	if collingBufferFirstRow > windowHeight {
		collingBufferFirstRow = 0 // after one complete screen roll up, reset
	}
}

// display some stuff on the screen
func drawFPS(surface *ebiten.Image) {
	if displayHelp {
		ebitenutil.DebugPrint(surface,
			fmt.Sprintf("FPS:%f\n[Up/Down] Number of flames=%d\n[Left/Right] Fire Power=%d\n[C] Color Map %d=%s\n[P]ause [S]tatic [M]ute [H]elp",
				ebiten.CurrentFPS(),
				numberOfHotSpot,
				firePower,
				currentColorMapIndex,
				colorMapLabels[currentColorMapIndex],
			))
	}
}

// draw a hotspot (3 pixel large) at the bottom of the screen
func drawHotspot(x int) {
	buffer1[pixelAt(x-1, windowHeight)] = maximumPower
	buffer1[pixelAt(x, windowHeight)] = maximumPower
	buffer1[pixelAt(x+1, windowHeight)] = maximumPower
}

// draw the current color map on the screen (5 pixels large)
func drawColorMap() {
	if displayColorMap {
		for x := 0; x < len(currentColorMap); x++ {
			for y := 0; y < colorMapHeight; y++ {
				imageBuffer.SetRGBA(x+20, 100+y, currentColorMap[x])
			}
		}
	}
}

// the cooling map use Perlin Noise for the generation
func initNoise() {
	p := perlin.NewPerlin(4, 2, 3, int64(rand.Intn(1000)))
	for x := 0.0; x < windowWidth; x++ {
		for y := 0.0; y < windowHeight; y++ {
			noise := p.Noise2D(x/10, y/10) * 10
			if noise < 0 {
				noise = 0
			}
			// store cooling value
			collingBuffer[pixelAt(int(x), int(y))] = uint8(math.Round(noise)) // fill the colling map with -1 to +1
		}
	}
}

// store the places of each hotspot
func initHotSpots() {
	// draw hot spots
	hotSpots = []int{} // reset hotSpots
	for i := 0; i < numberOfHotSpot; i++ {
		x := rand.Intn(windowWidth)
		if x < 2 {
			x = 2
		} else if x > windowWidth-2 {
			x = windowWidth - 2
		}
		hotSpots = append(hotSpots, x) // store hotspot
	}
}

// store the color maps
func initColorMaps() {
	colorMaps = make(map[string][256]color.RGBA)
	colorMaps["Black_Red_Yellow_White"] = blackRedYellowWhiteColorMap()
	colorMaps["Black_Yellow_White"] = blackYellowWhiteColorMap()
	colorMaps["Black_White"] = blackWhiteColorMap()
	colorMaps["Predator"] = predatorColorMap()

	colorMapLabels = []string{}
	for key := range colorMaps {
		colorMapLabels = append(colorMapLabels, key)
	}
	currentColorMap = colorMaps[colorMapLabels[currentColorMapIndex]]
	launchColorMapTimer()
}

// create the Black -> Yellow -> White  color map
func blackYellowWhiteColorMap() [256]color.RGBA {
	var colorMap [256]color.RGBA

	j := 0
	for i := 0; i < 128; i++ { // black to yellow
		colorMap[i] = color.RGBA{R: uint8(j * 2), G: uint8(j * 2), B: 0, A: 255}
		j++
	}

	j = 0
	for i := 128; i < 256; i++ { // yellow to white
		colorMap[i] = color.RGBA{R: 255, G: 255, B: uint8(j * 2), A: 255}
		j++
	}
	return colorMap
}

// create the Black -> Red -> Yellow -> White  color map
func blackRedYellowWhiteColorMap() [256]color.RGBA {
	var colorMap [256]color.RGBA

	j := 0
	for i := 0; i < 64; i++ { // from black to RED
		colorMap[i] = color.RGBA{R: uint8(j * 4), G: 0, B: 0, A: 255}
		j++
	}

	j = 0
	for i := 64; i < 192; i++ { // from RED to YELLOW
		colorMap[i] = color.RGBA{R: 255, G: uint8(j * 2), B: 0, A: 255}
		j++
	}

	j = 0
	for i := 192; i < 256; i++ { // from YELLOW to WHITE
		colorMap[i] = color.RGBA{R: 255, G: 255, B: uint8(j * 4), A: 255}
		j++
	}
	return colorMap
}

// create the Black -> White  color map
func blackWhiteColorMap() [256]color.RGBA {
	var colorMap [256]color.RGBA

	for i := 0; i < 256; i++ { // black to white
		colorMap[i] = color.RGBA{R: uint8(i), G: uint8(i), B: uint8(i), A: 255}
	}
	return colorMap
}

// create the Black -> Red -> Yellow -> White  color map
func predatorColorMap() [256]color.RGBA {
	var colorMap [256]color.RGBA

	step := 5

	j := 0
	for i := 0; i < int(255/step)*1; i++ { // from black to VIOLET
		colorMap[i] = color.RGBA{R: uint8(j * step), G: 0, B: uint8(j * step), A: 255}
		j++
	}

	j = 0
	for i := int(255/step) * 1; i < int(255/step)*2; i++ { // from VIOLET to BLUE
		colorMap[i] = color.RGBA{R: 255 - uint8(j*step), G: 0, B: 255, A: 255}
		j++
	}

	j = 0
	for i := int(255/step) * 2; i < int(255/step)*3; i++ { // from BLUE to RED
		colorMap[i] = color.RGBA{R: uint8(j * step), G: 0, B: 255 - uint8(j*step), A: 255}
		j++
	}

	j = 0
	for i := int(255/step) * 3; i < int(255/step)*4; i++ { // from RED to YELLOW
		colorMap[i] = color.RGBA{R: 255, G: uint8(j * step), B: 0, A: 255}
		j++
	}

	j = 0
	for i := int(255/step) * 4; i < 256; i++ { // from YELLOW to WHITE
		colorMap[i] = color.RGBA{R: 255, G: 255, B: uint8(j * step), A: 255}
		j++
	}

	return colorMap
}

// hide the color map after 2 seconds
func launchColorMapTimer() {
	displayColorMap = true
	timer2 := time.NewTimer(time.Second * 2)
	go func() {
		<-timer2.C
		displayColorMap = false
	}()
}

// play the fire sound
func initSound() {
	// load the file into memory
	soundFile, err := ioutil.ReadFile(soundFile)
	if err != nil {
		log.Fatal(err)
	}

	audioContext, err := audio.NewContext(44100)
	if err != nil {
		log.Fatal(err)
	}

	// Decode the mp3 file.
	wavS, err := mp3.Decode(audioContext, audio.BytesReadSeekCloser(soundFile))
	if err != nil {
		log.Fatal(err)
	}

	// Create an infinite loop stream from the decoded bytes.
	s := audio.NewInfiniteLoop(wavS, wavS.Length())

	audioPlayer, err = audio.NewPlayer(audioContext, s)
	if err != nil {
		log.Fatal(err)
	}

	// Play the infinite-length stream. This never ends.
	audioPlayer.Play()
}

// draw a circle
func drawCircle(x0, y0, r int) {
	x, y, dx, dy := r-1, 0, 1, 1
	err := dx - (r * 2)

	for x > y {
		buffer2[pixelAt(x0+x, y0+y)] = maximumPower
		buffer2[pixelAt(x0+y, y0+x)] = maximumPower
		buffer2[pixelAt(x0-y, y0+x)] = maximumPower
		buffer2[pixelAt(x0-x, y0+y)] = maximumPower
		buffer2[pixelAt(x0-x, y0-y)] = maximumPower
		buffer2[pixelAt(x0-y, y0-x)] = maximumPower
		buffer2[pixelAt(x0+y, y0-x)] = maximumPower
		buffer2[pixelAt(x0+x, y0-y)] = maximumPower

		if err <= 0 {
			y++
			err += dy
			dy += 2
		}
		if err > 0 {
			x--
			dx += 2
			err += dx - (r * 2)
		}
	}
}

// add random sparkles inside the flame
func addSparkles() {
	// 20% of the times, add a spakle
	if rand.Intn(10) > 8 {
		x := rand.Intn(windowWidth - 1)
		if x < 1 {
			x = 1
		}

		// but below the fire hightest point
		y := rand.Intn(fireHeight - 1)
		if y < 1 {
			y = 1
		}
		if y > windowHeight-1 {
			y = windowHeight - 1
		}

		// draw a little sparkle
		drawCircle(x, y, 2)
	}
}

// manage keyboard and mouse input
func bindings() {
	// if up, increase number of hotspots
	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		numberOfHotSpot = int(math.Min(float64(numberOfHotSpot+1), 300)) // maximum 300 hotspots
		initHotSpots()
	}

	// if down, decrease number of hotspots
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		numberOfHotSpot = int(math.Max(float64(numberOfHotSpot-1), 0))
		initHotSpots()
	}

	// if right, increase power the flames
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		firePower = int(math.Min(float64(firePower+1), 5)) // maximum 5
	}

	// if left, decrease power of the flames
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		firePower = int(math.Max(float64(firePower-1), 0))
	}

	// if C, change tghe current color map for the next one
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		currentColorMapIndex++
		if currentColorMapIndex > len(colorMaps)-1 {
			currentColorMapIndex = 0
		}
		currentColorMap = colorMaps[colorMapLabels[currentColorMapIndex]]
		launchColorMapTimer()
	}

	// if P, pause the animation
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		if pause {
			audioPlayer.Play() // restore audio
		} else {
			audioPlayer.Pause() // mute audio
		}
		pause = !pause
	}

	// if S, fix the hotspots place
	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		fixHotSpots = !fixHotSpots
	}

	// if H, toogle help displaying
	if inpututil.IsKeyJustPressed(ebiten.KeyH) {
		displayHelp = !displayHelp
	}

	// if M, toogle sound
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		if audioPlayer.IsPlaying() {
			audioPlayer.Pause()
		} else {
			audioPlayer.Play()
		}
	}

	// if Alt+Enter, toogle fullscreen
	if ebiten.IsKeyPressed(ebiten.KeyAlt) && ebiten.IsKeyPressed(ebiten.KeyEnter) {
		if lastStatePressedKey["Alt+Enter"] == false {
			if ebiten.IsFullscreen() {
				ebiten.SetFullscreen(false)
			} else {
				ebiten.SetFullscreen(true)
			}
			lastStatePressedKey["Alt+Enter"] = true
		}
	} else {
		lastStatePressedKey["Alt+Enter"] = false
	}

	// draw a fire circle where the mouse is pressed
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if x > 10 && x < windowWidth-10 && y > 10 && y < windowHeight-10 { // not click on the edge
			for radius := 1; radius <= 10; radius++ { // draw 10 circle from with R=i to 10
				drawCircle(x, y, radius)
			}
		}
	}

}
