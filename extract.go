package main

import (
	"errors"

	bimg "gopkg.in/h2non/bimg.v1"
)

func extract(image *bimg.Image, x int, y int, width int, height int) (*bimg.Image, error) {
	if x < 0 || y < 0 || width <= 0 || height <= 0 {
		return nil, errors.New("Dimension error")
	}
	image.Extract(x, y, width, height)
	return image, nil
}

func extractImage(data []byte, x int, y int, width int, height int) (*bimg.Image, error) {
	return extract(bimg.NewImage(data), x, y, width, height)
}

func extractImageFromPath(inputPath string, x int, y int, width int, height int) (*bimg.Image, error) {
	buffer, err := bimg.Read(inputPath)
	if err != nil {
		return nil, err
	}
	return extractImage(buffer, x, y, width, height)
}
