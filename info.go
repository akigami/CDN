package main

import (
	"fmt"
	"os"

	bimg "gopkg.in/h2non/bimg.v1"
)

func getImageDimension(data []byte) (int, int, error) {
	size, err := bimg.NewImage(data).Size()
	if err != nil {
		return -1, -1, err
	}
	return size.Width, size.Height, nil
}

func getImageDimensionFromPath(path string) (int, int, error) {
	buffer, err := bimg.Read(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return -1, -1, nil
	}
	return getImageDimension(buffer)
}
