Go-fire
=======

A fire simulation writen in Go for educational purpose.

It use the [Ebiten](https://github.com/hajimehoshi/ebiten) library for the 2D graphics engine

It use the [Perlin](https://github.com/aquilax/go-perlin) library for generating cooling map

Install
=======

```bash
$ go get -u github.com/hajimehoshi/ebiten
$ go get -u github.com/aquilax/go-perlin
$ go get -u github.com/ryosama/go-fire
```

Screenshot
===========

![Red Yellow color map](https://github.com/ryosama/go-fire/raw/master/screenshot1.png "Red Yellow color map")

![Yellow color map](https://github.com/ryosama/go-fire/raw/master/screenshot2.png "Yellow color map")

![White color map](https://github.com/ryosama/go-fire/raw/master/screenshot3.png "White color map")

Documentation
=============

- __[Up/Down]__ Increase or decrease the number of fire hotspots

- __[Left/Right]__ Increase or decrease the power of the flames

- __[C]__ Change the color map

- __[H]__ Display the commands and FPS

- __[S]__ Fix the hotspots places

- __[P]__ Pause the animation

- __[M]__ Mute the sound

- __[Alt+Enter]__ Toogle fullscreen

- __[Left Click]__ Draw a fire circle

TODO
=============

- Add a video capture