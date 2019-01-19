package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/xid"
	bimg "gopkg.in/h2non/bimg.v1"
)

func notFoundError(c *gin.Context) {
	c.AbortWithStatusJSON(404, gin.H{
		"error": "image not found",
	})
}

func sendError(c *gin.Context, err error) {
	c.AbortWithStatusJSON(500, gin.H{
		"error": err.Error(),
	})
}

func setFormat() gin.HandlerFunc {
	return func(c *gin.Context) {
		var format string
		if format = "jpg"; strings.Contains(c.GetHeader("Accept"), "image/webp") {
			format = "webp"
		}
		c.Set("format", format)
		c.Next()
	}
}

func stringToInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func main() {
	bimg.VipsCacheSetMax(0)
	bimg.VipsCacheSetMaxMem(0)

	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	executablePath := filepath.Dir(ex)

	config := initConfig(executablePath)

	r := gin.Default()

	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"code": "PAGE_NOT_FOUND", "message": "Page not found"})
	})

	r.POST("/api/upload", func(c *gin.Context) {
		if token := c.Request.FormValue("token"); token != config.token {
			c.AbortWithStatusJSON(403, gin.H{
				"error": "Token not found",
			})
			return
		}
		file, _, err := c.Request.FormFile("file")
		if err != nil {
			sendError(c, err)
			return
		}
		defer file.Close()
		data, err := ioutil.ReadAll(file)
		if err != nil {
			sendError(c, err)
			return
		}
		w, h, err := getImageDimension(data)
		if err != nil {
			sendError(c, err)
			return
		}
		var now = time.Now()
		var year, month, day, id = strconv.Itoa(now.Year()), strconv.Itoa(int(now.Month())), strconv.Itoa(now.Day()), xid.New().String()
		err = os.MkdirAll(path.Join(executablePath, "static", year, month, day, id), os.ModePerm)
		if err != nil {
			sendError(c, err)
			return
		}
		var path = path.Join(executablePath, "static", year, month, day, id, "image")

		form, err := c.MultipartForm()
		if err != nil {
			sendError(c, err)
			return
		}
		process := form.Value["process[]"]

		if w >= h && w > 1000 {
			w = 1000
			h = 0
		} else if h >= w && h > 1000 {
			w = 0
			h = 1000
		}

		image := imageFromData(data)

		for _, p := range process {
			if strings.HasPrefix(p, "resize") {
				r, _ := regexp.Compile("resize,(\\d+),(\\d+)")
				arr := r.FindStringSubmatch(p)
				if len(arr) > 0 {
					data := arr[1:]
					w, h = stringToInt(data[0]), stringToInt(data[1])
				}
			}
			if strings.HasPrefix(p, "extract") {
				r, _ := regexp.Compile("extract,(\\d+),(\\d+),(\\d+),(\\d+)")
				arr := r.FindStringSubmatch(p)
				if len(arr) > 0 {
					data := arr[1:]
					var x, y, w, h = stringToInt(data[0]), stringToInt(data[1]), stringToInt(data[2]), stringToInt(data[3])
					extract(image, x, y, w, h)
				}
			}
		}

		_, err = resize(image, w, h)
		if err != nil {
			sendError(c, err)
			return
		}
		err = writeImage(image, path)
		if err != nil {
			sendError(c, err)
			return
		}
		c.JSON(200, gin.H{
			"path": fmt.Sprintf("/%s/%s/%s/%s", year, month, day, id),
		})
	})

	r.GET("/:year/:month/:day/:id", setFormat(), func(c *gin.Context) {
		u, err := url.Parse(c.GetHeader("referer"))
		if err != nil {
			notFoundError(c)
			return
		}
		host, _, _ := net.SplitHostPort(u.Host)
		if contains(config.referers, "*") || contains(config.referers, host) {
			c.Next()
			return
		}
		notFoundError(c)
	}, func(c *gin.Context) {
		if width, _ := strconv.Atoi(c.Query("width")); width == 64 || width == 128 || width == 256 || width == 512 {
			var format = c.MustGet("format").(string)
			var directory = c.Request.URL.Path
			var imageFolder = path.Join(executablePath, "static", directory)
			var widthPath = path.Join(imageFolder, fmt.Sprintf("%d.%s", width, format))
			if _, err := os.Stat(widthPath); !os.IsNotExist(err) {
				c.File(widthPath)
				c.Abort()
				return
			}
			var imagePath = path.Join(imageFolder, fmt.Sprintf("image.%s", format))
			if _, err := os.Stat(imagePath); os.IsNotExist(err) {
				c.Next()
				return
			}
			var w, _, _ = getImageDimensionFromPath(imagePath)
			if width > w {
				c.Next()
				return
			}
			os.MkdirAll(imageFolder, os.ModePerm)
			var outputPath = path.Join(imageFolder, fmt.Sprintf("%d", width))
			var tempImagePath = path.Join(imageFolder, "image.jpg")
			image, err := resizeImageFromPath(tempImagePath, width, 0)
			if err != nil {
				c.AbortWithStatusJSON(500, gin.H{
					"error": err.Error(),
				})
			}
			writeImage(image, outputPath)
			c.File(widthPath)
			c.AbortWithStatus(200)
		} else {
			c.Next()
		}
	}, func(c *gin.Context) {
		var format = c.MustGet("format").(string)
		var directory = c.Request.URL.Path
		var staticPath = path.Join(executablePath, "static", directory, "image."+format)
		if _, err := os.Stat(staticPath); os.IsNotExist(err) {
			notFoundError(c)
			return
		}
		c.File(staticPath)
	})
	r.Run(":" + config.port)
}
