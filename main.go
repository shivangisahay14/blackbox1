package main

import (
	"fmt"
	"github.com/adrs/shortestpath/graph"
	"image"
	"image/color"
	"image/gif"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strconv"
)

func inRange(cord graph.Cord, minLat, maxLat, minLong, maxLong int) bool {
	return cord.Lat >= minLat && cord.Lat <= maxLat && cord.Long >= minLong && cord.Long <= maxLong
}

func pixelLocation(cord graph.Cord, minLat, minLong, radius, size int) (x, y int) {
	x = int(float64((cord.Long-minLong)*size) / float64(2*radius))
	y = int(float64((cord.Lat-minLat)*size) / float64(2*radius))
	return
}

var palette = []color.Color{
	color.RGBA{255, 255, 255, 255}, // White
	color.RGBA{128, 128, 128, 255}, // Grey
	color.RGBA{0, 0, 255, 255},     // Green
	color.RGBA{255, 0, 128, 255},   // Red
}

const (
	backgroundColor = 0
	unvisitedColor  = 1
	visitedColor    = 2
	pathColor       = 3
)

func makeMap(centerx, centery, radius, size int) *image.Paletted {
	img := image.NewPaletted(image.Rect(0, 0, size, size), palette)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.SetColorIndex(x, y, backgroundColor)
		}
	}
	minLat := centery - radius
	maxLat := centery + radius
	minLong := centerx - radius
	maxLong := centerx + radius
	for _, cord := range roadNetwork.Nodes {
		if !inRange(cord, minLat, maxLat, minLong, maxLong) {
			continue
		}
		x, y := pixelLocation(cord, minLat, minLong, radius, size)
		img.SetColorIndex(x, size-y, unvisitedColor)
	}
	return img
}

func drawMap(out io.Writer, centerx, centery, radius, size int) {
	img := gif.GIF{}
	img.Delay = append(img.Delay, 0)
	img.Image = append(img.Image, makeMap(centerx, centery, radius, size))
	gif.EncodeAll(out, &img)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Finds range of coordinate values specified by indices
func findCordinateRange(indices []int, cords []graph.Cord) (minLat, maxLat, minLong, maxLong int) {
	minLat, maxLat, minLong, maxLong = 180000000, -180000000, 180000000, -180000000
	for _, i := range indices {
		if cords[i].Lat > maxLat {
			maxLat = cords[i].Lat
		}
		if cords[i].Lat < minLat {
			minLat = cords[i].Lat
		}
		if cords[i].Long > maxLong {
			maxLong = cords[i].Long
		}
		if cords[i].Long < minLong {
			minLong = cords[i].Long
		}
	}
	return minLat, maxLat, minLong, maxLong
}

func drawShortestPath(out io.Writer, src, dest, size int) { //, frames int, animate bool) {
	// TODO: validate src and dest
	shortestPath, searchSeq := graph.SearchSequence(roadNetwork, src, dest)

	// Determine bounds from search sequence
	minLat, maxLat, minLong, maxLong := findCordinateRange(searchSeq, roadNetwork.Nodes)
	centerx := (minLong + maxLong) / 2
	centery := (minLat + maxLat) / 2
	dx := maxLong - minLong
	dy := maxLat - minLat
	radius := max(max(dx, dy)*11/20, 5e4)
	// TODO: add map boundaries
	// Resize image to remove space beyond map boundaries
	minLat = centery - radius
	maxLat = centery + radius
	minLong = centerx - radius
	maxLong = centerx + radius

	// TODO: generate animation
	anim := gif.GIF{}
	baseMap := makeMap(centerx, centery, radius, size)
	anim.Image = append(anim.Image, baseMap)
	anim.Delay = append(anim.Delay, 0)

	// Show search sequence
	for _, v := range searchSeq {
		cord := roadNetwork.Nodes[v]
		if !inRange(cord, minLat, maxLat, minLong, maxLong) {
			continue
		}
		x, y := pixelLocation(cord, minLat, minLong, radius, size)
		baseMap.SetColorIndex(x, size-y, visitedColor)
	}

	// Show shortest path
	for _, v := range shortestPath {
		cord := roadNetwork.Nodes[v]
		if !inRange(cord, minLat, maxLat, minLong, maxLong) {
			continue
		}
		x, y := pixelLocation(cord, minLat, minLong, radius, size)
		baseMap.SetColorIndex(x, size-y, pathColor)
	}
	gif.EncodeAll(out, &anim)
}

var roadNetwork *graph.Graph

func setup() {
	log.Print("Loading graph...")
	base := "/mnt/c/Users/adrian/Documents/EECS477/project/data/"
	g, err := graph.LoadGraph(base+"USA-road-d.LKS.co", base+"USA-road-t.LKS.gr")
	if err != nil {
		log.Fatal(err)
	}
	roadNetwork = g
}

// Parses integer, ensuring result is in [min, max]
// Provides default value of s is not an integer
func parseInt(s string, min, max, defaultValue int) int {
	x, err := strconv.Atoi(s)
	if err != nil {
		x = defaultValue
	}
	if x < min {
		x = min
	}
	if x > max {
		x = max
	}
	return x
}

// takes part of a lat long cordinate and parses it
func parseCordPart(s string, min, max, defaultValue float64) int {
	x, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return int(defaultValue * 1e6)
	}
	if x < min {
		x = min
	}
	if x > max {
		x = max
	}
	return int(x * 1e6)
}

func main() {
	rand.Seed(42)
	setup()
	http.Handle("/", http.FileServer(http.Dir("static")))
	http.HandleFunc("/map", func(w http.ResponseWriter, r *http.Request) {
		centerx := parseCordPart(r.FormValue("centerx"), -180, 180, -85)
		centery := parseCordPart(r.FormValue("centery"), -180, 180, 44)
		radius := parseCordPart(r.FormValue("radius"), 0.01, 90, 5)
		size := parseInt(r.FormValue("size"), 24, 2000, 400)
		drawMap(w, centerx, centery, radius, size)
	})
	http.HandleFunc("/shortest-path", func(w http.ResponseWriter, r *http.Request) {
		maxIdx := len(roadNetwork.Nodes)
		src := parseInt(r.FormValue("src"), 1, maxIdx, rand.Intn(maxIdx)+1) - 1
		dest := parseInt(r.FormValue("dest"), 1, maxIdx, rand.Intn(maxIdx)+1) - 1
		size := parseInt(r.FormValue("size"), 24, 2000, 400)
		//animate := (parseInt(r.FormValue("animate"), 0, 1, 1) == 1)
		//frames := parseInt(r.FormValue("frames"), 1, 120, 15)
		drawShortestPath(w, src, dest, size)
	})
	http.HandleFunc("/vertex", func(w http.ResponseWriter, r *http.Request) {
		i, err := strconv.Atoi(r.FormValue("i"))
		if err != nil {
			fmt.Fprintf(w, "error: invalid \"i\" parameter")
			return
		}
		i--
		if i < 0 || i >= len(roadNetwork.Nodes) {
			fmt.Fprintf(w, "error: index out of range")
			return
		}
		fmt.Fprintf(w, "Cordinates: %s\n", roadNetwork.Nodes[i])
		fmt.Fprintf(w, "Edges:")
		for _, dest := range roadNetwork.AdjacencyLists[i] {
			fmt.Fprintf(w, "\tDestination: %d at %s, Distance: %d\n", dest.Dest+1, roadNetwork.Nodes[dest.Dest], dest.Dist)
		}
	})
	log.Print("Starting server...")
	log.Fatal(http.ListenAndServe("localhost:8888", nil))
}
