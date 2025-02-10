package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"
	"net/http"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"

	chart "github.com/wcharczuk/go-chart/v2"
)

// PingResult holds the timestamp and measured latencies.
type PingResult struct {
	Timestamp       time.Time
	GstaticLatency  int64 // in milliseconds
	ApenwarrLatency int64 // in milliseconds
}

var (
	results   []PingResult
	resultsMu sync.Mutex
)

const (
	gstaticURL  = "https://www.gstatic.com/generate_204"
	apenwarrURL = "https://apenwarr.ca"
)

// doPings pings two endpoints every second.
func doPings() {
	for {
		now := time.Now()

		// Ping gstatic.
		start := time.Now()
		resp, err := http.Get(gstaticURL)
		gLatency := time.Since(start).Milliseconds()
		if err != nil {
			log.Println("gstatic ping error:", err)
			gLatency = -1
		} else {
			resp.Body.Close()
		}

		// Ping apenwarr.
		start = time.Now()
		resp, err = http.Get(apenwarrURL)
		aLatency := time.Since(start).Milliseconds()
		if err != nil {
			log.Println("apenwarr ping error:", err)
			aLatency = -1
		} else {
			resp.Body.Close()
		}

		// Record the result.
		resultsMu.Lock()
		results = append(results, PingResult{
			Timestamp:       now,
			GstaticLatency:  gLatency,
			ApenwarrLatency: aLatency,
		})
		// Keep only the most recent 60 data points.
		if len(results) > 60 {
			results = results[len(results)-60:]
		}
		resultsMu.Unlock()

		time.Sleep(1 * time.Second)
	}
}

// renderChart creates a PNG image (decoded as image.Image) from the recorded data.
// It only renders a chart if there are at least 2 data points.
func renderChart() image.Image {
	// Check for minimum data points.
	resultsMu.Lock()
	n := len(results)
	resultsMu.Unlock()
	if n < 2 {
		return nil
	}

	resultsMu.Lock()
	defer resultsMu.Unlock()

	// Prepare data for the time series.
	xValues := make([]time.Time, n)
	yValuesG := make([]float64, n)
	yValuesA := make([]float64, n)
	for i, r := range results {
		xValues[i] = r.Timestamp
		yValuesG[i] = float64(r.GstaticLatency)
		yValuesA[i] = float64(r.ApenwarrLatency)
	}

	// Create a chart with two time series.
	graph := chart.Chart{
		XAxis: chart.XAxis{
			ValueFormatter: chart.TimeValueFormatterWithFormat("15:04:05"),
		},
		YAxis: chart.YAxis{ValueFormatter: func(v interface{}) string {
			return fmt.Sprintf("%.0f ms", v.(float64))
		}},
		Series: []chart.Series{
			chart.TimeSeries{
				Name:    "gstatic" + " (" + gstaticURL + ")", // Label the series with the URL.
				XValues: xValues,
				YValues: yValuesG,
			},
			chart.TimeSeries{
				Name:    "apenwarr" + " (" + apenwarrURL + ")", // Label the series with the URL.
				XValues: xValues,
				YValues: yValuesA,
			},
		},
	}
	// Add a legend so that each graph is clearly labeled.
	graph.Elements = []chart.Renderable{
		chart.Legend(&graph),
	}

	// Render the chart into a PNG image.
	buffer := bytes.NewBuffer(nil)
	if err := graph.Render(chart.PNG, buffer); err != nil {
		log.Println("Error rendering chart:", err)
		return nil
	}
	img, _, err := image.Decode(buffer)
	if err != nil {
		log.Println("Error decoding chart image:", err)
		return nil
	}
	return img
}

// renderPlaceholder creates a blank placeholder image.
// (If you wish to actually render text into the image, consider using an additional library such as github.com/fogleman/gg.)
func renderPlaceholder() image.Image {
	width, height := 800, 600
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill with a white background.
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	return img
}

func main() {
	// Create a new Fyne application.
	myApp := app.New()
	myWindow := myApp.NewWindow("Native Go Blip")

	// Create the image widget.
	chartImage := canvas.NewImageFromImage(nil)
	chartImage.FillMode = canvas.ImageFillContain

	// Create a text placeholder.
	loadingText := canvas.NewText("Loading...", color.Black)
	loadingText.TextStyle = fyne.TextStyle{Bold: true}
	loadingText.Alignment = fyne.TextAlignCenter

	// Overlay the image and text.
	content := container.NewStack(chartImage, loadingText)
	myWindow.SetContent(content)
	myWindow.Resize(fyne.NewSize(800, 600))

	// Start the pinging process.
	go doPings()

	// Update loop: if there are not enough data points, show the placeholder.
	go func() {
		for {
			resultsMu.Lock()
			n := len(results)
			resultsMu.Unlock()
			if n < 2 {
				// While waiting for data, show the placeholder.
				loadingText.Show()
				chartImage.Image = renderPlaceholder() // A blank background (optionally replace with a drawn "Loading..." image).
				chartImage.Refresh()
			} else {
				// Data available: hide the placeholder and update the chart.
				loadingText.Hide()
				if img := renderChart(); img != nil {
					chartImage.Image = img
					chartImage.Refresh()
				}
			}
			time.Sleep(1 * time.Second)
		}
	}()

	myWindow.ShowAndRun()
}
