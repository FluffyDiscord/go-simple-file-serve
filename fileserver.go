package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/akamensky/argparse"
	"github.com/gin-gonic/gin"
	"gopkg.in/gographics/imagick.v2/imagick"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var allowedIp string

func getCurrentDir() string {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	return strings.ReplaceAll(filepath.Dir(ex), string(os.PathSeparator), "/")
}

func getParams() (*int, *string) {
	parser := argparse.NewParser("basic-file-server", "Basic file server")

	port := parser.Int("p", "port", &argparse.Options{Required: false, Default: 9068, Help: "Port to listen to"})
	basePath := parser.String("b", "basePath", &argparse.Options{Required: false, Default: getCurrentDir(), Help: "Base path from which files will be served"})

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
	}

	return port, basePath
}

func isDirectory(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}

	return fileInfo.IsDir()
}

func getCachedFilePath(key string, cachePath string) string {
	h := md5.New()
	h.Write([]byte(key))
	str := hex.EncodeToString(h.Sum(nil))

	return cachePath + "/" + str + ".json"
}

func fileExists(pathName string) bool {
	_, err := os.Stat(pathName)
	return !errors.Is(err, os.ErrNotExist)
}

func isImage(lookup string) bool {
	switch lookup {
	case
		".jpeg",
		".png",
		".webp",
		".jpg",
		".gif",
		".avif":
		return true
	}
	return false
}

func getSourceImageForCover(lookup string) string {
	if fileExists(lookup + "/1.jpg") {
		return lookup + "/1.jpg"
	}
	if fileExists(lookup + "/1.jpeg") {
		return lookup + "/1.jpeg"
	}
	if fileExists(lookup + "/1.png") {
		return lookup + "/1.png"
	}
	if fileExists(lookup + "/1.avif") {
		return lookup + "/1.avif"
	}
	if fileExists(lookup + "/1.gif") {
		return lookup + "/1.gif"
	}
	if fileExists(lookup + "/1.webp") {
		return lookup + "/1.webp"
	}
	return lookup
}

func main() {
	imagick.Initialize()
	defer imagick.Terminate()

	port, basePath := getParams()

	cachePath := "./cache/"
	os.Mkdir(cachePath, 0755)

	r := gin.Default()
	r.GET("/*path", func(c *gin.Context) {
		if c.ClientIP() != allowedIp {
			c.Status(403)
			return
		}

		path := c.Param("path")

		fullPath := fmt.Sprintf("%s/%s", *basePath, path)

		if isDirectory(fullPath) {
			cachedFilePath := getCachedFilePath(path, cachePath)
			if fileExists(cachedFilePath) {
				c.Header("Content-Type", "application/json")
				c.File(cachedFilePath)
			} else {
				files, _ := ioutil.ReadDir(fullPath)

				var responseFiles []interface{}

				for _, file := range files {
					entry := make(map[string]interface{})
					entry["isDir"] = file.IsDir()
					entry["name"] = file.Name()
					entry["size"] = file.Size()
					responseFiles = append(responseFiles, entry)
				}

				cacheData := make(map[string]interface{})
				cacheData["entries"] = responseFiles
				encodedCacheData, _ := json.Marshal(cacheData)

				ioutil.WriteFile(cachedFilePath, encodedCacheData, 0666)

				c.JSON(200, gin.H{
					"entries": responseFiles,
				})
			}
		} else {
			extension := filepath.Ext(fullPath)

			if isImage(extension) {
				pathNoExtension := strings.TrimSuffix(fullPath, extension)

				isCover := filepath.Base(pathNoExtension) == "cover"
				if isCover {
					coverPathName := filepath.Dir(pathNoExtension) + "/cover.jpg"
					if !fileExists(coverPathName) {
						sourceImagePath := getSourceImageForCover(filepath.Dir(pathNoExtension))
						mw := imagick.NewMagickWand()
						defer mw.Destroy()

						err := mw.ReadImage(sourceImagePath)
						if err != nil {
							c.File(sourceImagePath)
							return
						}

						width, height, err := mw.GetSize()
						if err != nil {
							return
						}

						if width > 320 {
							scaleRatio := 320 / width
							width = width * scaleRatio
							height = height * scaleRatio

							err := mw.ResizeImage(width, height, imagick.FILTER_LANCZOS, -0.1)
							if err != nil {
								c.File(sourceImagePath)
								return
							}
						}

						err = mw.WriteImage(coverPathName)
						if err != nil {
							c.File(sourceImagePath)
							return
						}
						c.File(coverPathName)
						return
					}
				}
			}
			c.File(fullPath)
		}
	})

	r.Run(fmt.Sprintf(":%d", *port))
}
