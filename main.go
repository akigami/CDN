package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/olebedev/config"
	"github.com/rs/xid"

	"gopkg.in/h2non/bimg.v1"

	"github.com/kataras/iris"

	"github.com/kataras/iris/middleware/logger"
	"github.com/kataras/iris/middleware/recover"
)

func contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

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

func resizeImage(data []byte, outputPath string, width int, height int) {
	newImage := bimg.NewImage(data)
	newImage.Resize(width, height)
	jpegImage, err := newImage.Convert(bimg.JPEG)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	bimg.Write(outputPath+".jpg", jpegImage)
	webpImage, err := newImage.Convert(bimg.WEBP)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	bimg.Write(outputPath+".webp", webpImage)
}

func resizeImageFromFile(inputPath string, outputPath string, width int, height int) {
	buffer, err := bimg.Read(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}
	resizeImage(buffer, outputPath, width, height)
}

func initConfig(executablePath string) (string, string, []string) {
	file, err := ioutil.ReadFile(path.Join(executablePath, "config.yml"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(0)
	}
	yamlString := string(file)

	cfg, err := config.ParseYaml(yamlString)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(0)
	}
	port, err := cfg.Int("port")

	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(0)
	}

	token, err := cfg.String("token")

	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(0)
	}

	referer, err := cfg.List("referer")

	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(0)
	}
	var ref []string
	for _, value := range referer {
		ref = append(ref, value.(string))
	}
	return strconv.Itoa(port), token, ref
}

func main() {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	executablePath := filepath.Dir(ex)

	port, accessToken, ref := initConfig(executablePath)
	app := iris.New().Configure(iris.WithConfiguration(iris.Configuration{
		PostMaxMemory: 5 << 20,
	}))
	app.Logger().SetLevel("debug")

	app.Use(recover.New())
	app.Use(logger.New())

	app.OnErrorCode(iris.StatusNotFound, func(ctx iris.Context) {
		ctx.WriteString("404")
	})

	app.OnErrorCode(iris.StatusInternalServerError, func(ctx iris.Context) {
		ctx.HTML("Message: <b>" + ctx.Values().GetString("message") + "</b>")
	})

	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("It's index")
	})

	app.Get("/404.jpg", func(ctx iris.Context) {
		ctx.ServeFile(path.Join(executablePath, "images", "image_not_found.jpg"), false)
	})

	app.Get("/images/{directory:path}", func(ctx iris.Context) {
		var referer = ctx.GetReferrer()
		var url = referer.Domain + "." + referer.Tld
		if contains(ref, "*") || contains(ref, url) {
			ctx.Next()
		}
		ctx.Redirect("/404.jpg")
	},
		func(ctx iris.Context) {
			if width, _ := strconv.Atoi(ctx.FormValue("width")); width == 64 || width == 128 || width == 256 || width == 512 {
				var format string
				if format = "jpg"; strings.Contains(ctx.GetHeader("Accept"), "image/webp") {
					format = "webp"
				}
				var directory = ctx.Params().Get("directory")
				var imageFolder = path.Join(executablePath, "static", directory)
				var widthPath = path.Join(imageFolder, fmt.Sprintf("%d.%s", width, format))
				if _, err := os.Stat(widthPath); !os.IsNotExist(err) {
					ctx.ServeFile(widthPath, false)
					return
				}
				var imagePath = path.Join(imageFolder, fmt.Sprintf("image.%s", format))
				if _, err := os.Stat(imagePath); os.IsNotExist(err) {
					ctx.Next()
					return
				}
				var w, _, _ = getImageDimensionFromPath(imagePath)
				if width > w {
					ctx.Next()
					return
				}
				os.MkdirAll(imageFolder, os.ModePerm)
				var outputPath = path.Join(imageFolder, fmt.Sprintf("%d", width))
				var tempImagePath = path.Join(imageFolder, "image.jpg")
				resizeImageFromFile(tempImagePath, outputPath, width, 0)
				ctx.ServeFile(widthPath, false)
			} else {
				ctx.Next()
			}
		}, func(ctx iris.Context) {
			var format string
			if format = "jpg"; strings.Contains(ctx.GetHeader("Accept"), "image/webp") {
				format = "webp"
			}
			var directory = ctx.Params().Get("directory")
			var staticPath = path.Join(executablePath, "static", directory, "image."+format)
			if _, err := os.Stat(staticPath); os.IsNotExist(err) {
				ctx.Redirect("/404.jpg")
				return
			}
			ctx.ServeFile(staticPath, false)
		})

	app.Post("/api/upload", func(ctx iris.Context) {
		if token := ctx.FormValue("token"); token != accessToken {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.JSON(iris.Map{
				"error": "Token not found",
			})
			return
		}
		file, _, err := ctx.FormFile("file")
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.JSON(iris.Map{
				"error": err.Error(),
			})
			return
		}
		defer file.Close()
		data, _ := ioutil.ReadAll(file)
		var w, h, dimErr = getImageDimension(data)
		if dimErr != nil {
			ctx.JSON(iris.Map{
				"error": dimErr.Error(),
			})
			return
		}
		var now = time.Now()
		var year, month, day, id = now.Year(), now.Month(), now.Day(), xid.New().String()
		os.MkdirAll(fmt.Sprintf("./static/%d/%d/%d/%s", year, month, day, id), os.ModePerm)
		var path = fmt.Sprintf("./static/%d/%d/%d/%s/image", year, month, day, id)
		if w >= h && w > 1000 {
			w = 1000
			h = 0
		} else if h >= w && h > 1000 {
			w = 0
			h = 1000
		}
		resizeImage(data, path, w, h)
		ctx.JSON(iris.Map{
			"path": fmt.Sprintf("/images/%d/%d/%d/%s", year, month, day, id),
		})
	})

	app.Run(iris.Addr(":"+port), iris.WithoutServerError(iris.ErrServerClosed))
}
