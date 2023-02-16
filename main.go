package main

import (
	"KM8Oz/svg2png-go/docs"
	"bytes"
	"fmt"
	"image"
	"image/png"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/pkg/errors"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	// gin-swagger middleware
	// swagger embed files
)

// @BasePath /api/v1

// s godoc

// @Summary Convert SVG to PNG

// @version 1.0.0

// @title Swagger SVG to PNG API

// @Schemes http https

// @Description Convert an SVG image to a PNG image with the specified dimensions.

// @Tags SVG, SVG2PNG, PNG, CONVERT SVG2PNG, CONVERT SVG, PNGSVG, SVGPNG

// @host svg2png.kmoz.dev

// @contact.name   API Support

// @contact.url    http://www.swagger.io/support

// @contact.email  support@swagger.io

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key

// @license.name  Apache 2.0

// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @Summary	Convert an SVG image to a PNG image with the specified dimensions.
// @Tags SVG2PNG
// @Accept plain
// @Produce png
// @Param width query int false "Width of the resulting PNG image" default(512)
// @Param height query int false "Height of the resulting PNG image" default(512)
// @Param svg_url query string false "svg url to convert"
// @Success 200 {string} binary "Binary PNG data"
// @Failure 400 {string} string "Invalid request parameters"
// @Failure 500 {string} string "Internal server error"
// @Router /convert [get]
// @security ApiKeyAuth
func convertHandler(c *gin.Context) {
	// Get the SVG file URL and API key from the request headers
	// Create a new rotatelogs file writer
	writer, err := rotatelogs.New(
		"app.log.%Y%m%d%H%M%S",
		rotatelogs.WithMaxAge(time.Hour*24*7),
		rotatelogs.WithRotationSize(1024*1024),
	)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Error initializing log file"})
		return
	}
	defer writer.Close()
	logger := log.New(writer, "", log.LstdFlags)
	logger.Printf("Received request to convert SVG to PNG: %s %s\n", c.ClientIP(), c.RemoteIP())
	// svgURL := c.GetHeader("SVG-URL")
	apiKey := c.GetHeader("X-API-Key")
	if apiKey != "eaf9919f6f57f0be0f556c30f2f0fd9dbd0e80ffc5eb836a083e8cc1c99b6fdbc690" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid apiKey"})
	}

	// Get the requested width and height from the query parameters
	widthStr := c.DefaultQuery("width", "512")
	svgURL := c.Query("svg_url")
	heightStr := c.DefaultQuery("height", "512")
	dpiStr := c.DefaultQuery("dpi", "72")

	// Parse the width, height, and DPI values into integers
	width, err := strconv.Atoi(widthStr)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid width"})
		return
	}
	height, err := strconv.Atoi(heightStr)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid height"})
		return
	}
	dpi, err := strconv.Atoi(dpiStr)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid DPI"})
		return
	}

	// Convert the SVG to PNG
	pngData, err := convertSVGToPNG(svgURL, apiKey, width, height, dpi)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": errors.Wrap(err, "Error converting SVG to PNG").Error()})
		return
	}

	// Set the content type and write the PNG data to the response
	c.Header("Content-Type", "image/png")
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Write(pngData)
}

func convertSVGToPNG(svgURL, apiKey string, width, height, dpi int) ([]byte, error) {
	// Read the SVG data from the URL
	client := &http.Client{}

	// Send a GET request to the URL
	req, err := http.NewRequest("GET", svgURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Error reading SVG data from %v", svgURL))
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Error reading SVG data from %v", svgURL))
	}
	defer resp.Body.Close()

	// Parse the SVG data
	icon, err := oksvg.ReadIconStream(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Error parsing SVG data")
	}

	// create output image
	rgba := image.NewRGBA(image.Rect(0, 0, width, height))

	// draw SVG onto output image
	icon.SetTarget(0, 0, float64(width), float64(height))
	icon.Draw(rasterx.NewDasher(width, height, rasterx.NewScannerGV(width, height, rgba, rgba.Bounds())), 1)

	// encode output image as PNG
	var pngData bytes.Buffer
	if err := png.Encode(&pngData, rgba); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	resizedImg := imaging.Resize(img, width, height, imaging.Lanczos)
	// Encode the image as PNG
	if err := png.Encode(&pngData, resizedImg); err != nil {
		return nil, errors.Wrap(err, "Error encoding image as PNG")
	}

	// Return the PNG data
	return pngData.Bytes(), nil
}

func main() {
	r := gin.Default()
	docs.SwaggerInfo.BasePath = "/api/v1"
	docs.SwaggerInfo.Title = "Swagger SVG to PNG API"
	docs.SwaggerInfo.Version = "1.0.0"
	v1 := r.Group("/api/v1")
	{
		v1.POST("/convert", convertHandler)
		v1.GET("/convert", convertHandler)
	}
	// r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	// Serve the Swagger UI
	r.GET("/swagger.json", func(c *gin.Context) {
		c.File("./docs/swagger.json")
	})
	r.Run(":3890")
}
