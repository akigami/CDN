package main

import (
	bimg "gopkg.in/h2non/bimg.v1"
)

func imageFromData(data []byte) *bimg.Image {
	return bimg.NewImage(data)
}

func imageFromPath(path string) (*bimg.Image, error) {
	data, err := bimg.Read(path)
	if err != nil {
		return nil, err
	}
	return bimg.NewImage(data), nil
}

func writeImage(image *bimg.Image, outputPath string) error {
	jpegImage, err := image.Convert(bimg.JPEG)
	if err != nil {
		//fmt.Fprintln(os.Stderr, err)
		return err
	}
	bimg.Write(outputPath+".jpg", jpegImage)
	webpImage, err := image.Convert(bimg.WEBP)
	if err != nil {
		//fmt.Fprintln(os.Stderr, err)
		return err
	}
	bimg.Write(outputPath+".webp", webpImage)
	return nil
}
