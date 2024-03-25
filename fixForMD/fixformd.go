package fixForMD

import (
	"bytes"
	"path"
	"regexp"
	"strings"
)

var imageReference = regexp.MustCompile(`(?i)\]: (.*(jpg|png|svg))`) // [test]: ./images/1.png
var r = regexp.MustCompile(`(?i)\([^)].*?\)`)
var imageInHTML = regexp.MustCompile(`(?i)src="(.*.(?:png|svg|jpg))"`) // <a src="./images/1.png">

func RewritePath(imagePath string, contentPath string, basePath string) string {
	d := path.Dir(path.Clean(contentPath))
	fixed := path.Join(d, imagePath)
	fixed = strings.TrimPrefix(fixed, path.Clean(basePath)+"/")
	//fmt.Printf("%s: %s, (%s) (%s)\n", imagePath, fixed, contentPath, path.Clean(basePath))
	return fixed
}

func FixPathsMD(f []byte, contentPath string, basePath string) []byte {
	images := [][]byte{}

	referencesInMD := imageReference.FindAllSubmatch(f, -1)
	for _, i := range referencesInMD {
		if len(i) < 2 {
			continue
		}

		img := i[1]
		images = append(images, img)
	}

	imagesInMD := r.FindAll(f, -1)
	extensions := []string{"jpg", "png", "svg"}
	for _, i := range imagesInMD {
		images = append(images, i)
	}

	inHTML := imageInHTML.FindAllSubmatch(f, -1)

	for _, ii := range inHTML {
		for n, i := range ii {
			if n == 0 {
				continue
			}

			var fixedName []byte
			fixedName = i
			if bytes.HasPrefix(fixedName, []byte("/azure/architecture")) {
				fixedName = bytes.TrimPrefix(fixedName, []byte("/azure/architecture"))
			}

			images = append(images, fixedName)
		}
	}

	for _, v := range images {
		var fixedName []byte
		var extIndex int
		for _, e := range extensions {
			extIndex = bytes.Index(bytes.ToLower(v), []byte("."+e))
			if extIndex > 0 {
				fixedName = v[0 : extIndex+1+3]
				break
			}
		}

		if extIndex == -1 {
			continue
		}

		c := bytes.TrimPrefix(fixedName, []byte("("))

		res := RewritePath(string(c), contentPath, basePath)
		//fmt.Printf("replacing %s -> %s\n", c, res)
		f = bytes.ReplaceAll(f, c, []byte(res))
	}

	return f
}
