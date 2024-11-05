package main

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/unidoc/unipdf/v3/common"
	"github.com/unidoc/unipdf/v3/common/license"
	"github.com/unidoc/unipdf/v3/creator"
	"github.com/unidoc/unipdf/v3/extractor"
	"github.com/unidoc/unipdf/v3/model"
)

func init() {
	// Make sure to load your metered License API key prior to using the library.
	// If you need a key, you can sign up and create a free one at https://cloud.unidoc.io
	err := license.SetMeteredKey(os.Getenv("UNIDOC_LICENSE_API_KEY"))
	if err != nil {
		fmt.Printf("ERROR: Failed to set metered key: %v\n", err)
		fmt.Printf("Make sure to get a valid key from https://cloud.unidoc.io\n")
		panic(err)
	}

	common.SetLogger(common.NewConsoleLogger(common.LogLevelInfo))
}

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("Syntax: go run main.go input.pdf output.pdf\n")
		os.Exit(1)
	}

	inputPath := os.Args[1]
	outputPath := os.Args[2]

	// Create temporary directory, this will be removed after operation is done.
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("*-%s", inputPath))
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir)
	fmt.Printf("Creating temporary directory at: %s\n", tempDir)

	// Extract images from the input PDF to a temporary directory.
	err = extractImagesToTempDir(inputPath, tempDir)
	if err != nil {
		panic(err)
	}

	files, err := ioutil.ReadDir(tempDir)
	if err != nil {
		panic(err)
	}

	var images []image.Image

	fmt.Printf("total images: %v\n", len(files))
	for _, f := range files {
		ext := filepath.Ext(f.Name())
		if ext != ".jpg" && ext != ".png" {
			log.Printf("extension not supported: %v\n", ext)
			continue
		}

		fImg, err := os.Open(filepath.Join(tempDir, f.Name()))
		if err != nil {
			log.Fatalf("err: %v\n", err)
		}
		defer fImg.Close()

		imgDec, _, err := image.Decode(fImg)
		if err != nil {
			log.Fatalf("err: %v\n", err)
		}

		images = append(images, imgDec)
	}

	pdfBytes, err := pdfFromGoImages(images)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(outputPath, pdfBytes, 0755)
	if err != nil {
		panic(err)
	}
}

// Extracts images and properties of a PDF specified by inputPath.
// The output images are stored into a zip archive whose path is given by outputPath.
func extractImagesToTempDir(inputPath string, tempDir string) error {
	pdfReader, f, err := model.NewPdfReaderFromFile(inputPath, nil)
	if err != nil {
		return err
	}
	defer f.Close()

	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		return err
	}
	fmt.Printf("PDF Num Pages: %d\n", numPages)

	// Prepare output archive.
	totalImages := 0
	for i := 0; i < numPages; i++ {
		fmt.Printf("-----\nPage %d:\n", i+1)

		page, err := pdfReader.GetPage(i + 1)
		if err != nil {
			return err
		}

		pextract, err := extractor.New(page)
		if err != nil {
			return err
		}

		pimages, err := pextract.ExtractPageImages(nil)
		if err != nil {
			return err
		}

		fmt.Printf("%d Images\n", len(pimages.Images))
		for idx, img := range pimages.Images {
			fmt.Printf("Image %d - X: %.2f Y: %.2f, Width: %.2f, Height: %.2f\n",
				totalImages+idx+1, img.X, img.Y, img.Width, img.Height)
			fname := filepath.Join(tempDir, fmt.Sprintf("p%d_%d.jpg", i+1, idx))

			gimg, err := img.Image.ToGoImage()
			if err != nil {
				return err
			}

			imgf, err := os.Create(fname)
			if err != nil {
				return err
			}
			opt := jpeg.Options{Quality: 100}
			err = jpeg.Encode(imgf, gimg, &opt)
			if err != nil {
				return err
			}
		}
		totalImages += len(pimages.Images)
	}

	fmt.Printf("Total: %d images\n", totalImages)
	return nil
}

// pdfFromGoImages creates a pdf from an array of images, each on a separate page
func pdfFromGoImages(images []image.Image) ([]byte, error) {
	c := creator.New()

	margins := float64(10)

	for _, img := range images {
		pImg, err := c.NewImageFromGoImage(img)
		if err != nil {
			return nil, err
		}
		c.NewPage()

		// printMemStats()

		// scale to page width
		pImg.ScaleToWidth(c.Width() - (margins * 2))
		pImg.SetPos(margins, margins)

		if err := c.Draw(pImg); err != nil {
			return nil, err
		}

		// printMemStats()
	}

	var outBytes bytes.Buffer
	writer := bufio.NewWriter(&outBytes)
	if err := c.Write(writer); err != nil {
		return nil, err
	}

	return outBytes.Bytes(), nil
}

func printMemStats() {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	fmt.Printf("Alloc = %v MiB, TotalAlloc = %v MiB, Sys = %v MiB, NumGC = %v\n",
		bToMb(mem.Alloc), bToMb(mem.TotalAlloc), bToMb(mem.Sys), mem.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
