package main

import (
	"fmt"
	"os"

	bimg "gopkg.in/h2non/bimg.v1"
)

func resize(image *bimg.Image, width int, height int) (*bimg.Image, error) {
	_, err := image.Resize(width, height)
	return image, err
}

func resizeImage(data []byte, width int, height int) (*bimg.Image, error) {
	return resize(bimg.NewImage(data), width, height)
}

func resizeImageFromPath(inputPath string, width int, height int) (*bimg.Image, error) {
	buffer, err := bimg.Read(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return nil, err
	}
	return resizeImage(buffer, width, height)
}
