package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/akamensky/argparse"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

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

func main() {
	port, basePath := getParams()

	cachePath := "./cache/"
	os.Mkdir(cachePath, 0755)

	r := gin.Default()
	r.GET("/*path", func(c *gin.Context) {
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
			c.File(fullPath)
		}
	})

	r.Run(fmt.Sprintf(":%d", *port))
}
