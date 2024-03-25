package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"github.com/go-shiori/go-epub"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/vincent-petithory/dataurl"
	"gopkg.in/yaml.v2"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

//go:embed epub.css
var emb embed.FS

// var r = regexp.MustCompile(`(?i)[(].*[^)]\.(png|jpg|svg)\)`) // ![title](image.png)
var r = regexp.MustCompile(`(?i)\([^)].*?\)`)
var imageReference = regexp.MustCompile(`(?i)\]: (.*(jpg|png|svg))`)   // [test]: ./images/1.png
var imageInHTML = regexp.MustCompile(`(?i)src="(.*.(?:png|svg|jpg))"`) // <a src="./images/1.png">
var inc = regexp.MustCompile("(?i)\\[\\!include\\[\\]\\((.*.md)\\)]")
var tripleColon = regexp.MustCompile("(?i):::image .* source=\"(.*?)\".*:::")
var altText = regexp.MustCompile("(?i)alt-text=\"(.*?)\"")

type TocItem struct {
	Name  string    `yaml:"name"`
	Href  string    `yaml:"href,omitempty"`
	Items []TocItem `yaml:"items,omitempty"`
}

type Toc struct {
	Items []TocItem `yaml:"items"`
}

func getPics(basePath string, e *epub.Epub) {
	err := filepath.Walk(basePath, func(p string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		// `.png` as filename?
		if len(info.Name()) < 5 {
			return nil
		}

		ext := info.Name()[len(info.Name())-3 : len(info.Name())]

		switch ext {
		case "png", "jpg":
			c := strings.TrimPrefix(p, path.Clean(basePath))
			c = strings.TrimPrefix(c, "/")
			newName := strings.ReplaceAll(c, "/", "_")
			//fmt.Println("new image " + newName)

			_, err = e.AddImage(p, strings.ToLower(newName))
			if err != nil {
				panic(err)
			}
		}
		return nil
	})

	if err != nil {
		panic(err)
	}
}

// MDFromYML extracts path to first included md file from yml
// path to md is relative to yml
func MDFromYML(ymlPath string) (md []byte, contentPath string, err error) {
	var ymlData []byte
	if ymlData, err = os.ReadFile(ymlPath); err != nil {
		return nil, "", fmt.Errorf("cannot read md from yml: %w", err)
	}

	// only first occurrence
	res := inc.FindStringSubmatch(string(ymlData))
	if len(res) < 2 {
		return nil, "", fmt.Errorf("failed to found included md with actual contents in yml %s", ymlPath)
	}

	d := path.Dir(ymlPath)
	mdPath := path.Join(d, res[1])

	if md, err = os.ReadFile(mdPath); err != nil {
		return nil, "", fmt.Errorf("cannot open md %s from yml %s: %w", ymlPath, mdPath, err)
	}
	return md, mdPath, nil
}

func FixMDImages(f []byte, fullPath string, basePath string) []byte {
	d := path.Dir(fullPath)

	replaces := map[string]string{}

	referencesInMD := imageReference.FindAllSubmatch(f, -1)
	for _, i := range referencesInMD {
		if len(i) < 2 {
			continue
		}

		img := i[1]
		clean := path.Clean(path.Join(d, string(img)))
		fixedName := strings.TrimPrefix(clean, path.Clean(basePath))
		fixedName = strings.TrimPrefix(fixedName, "/")
		fixedName = strings.ReplaceAll(fixedName, "/", "_")
		if strings.HasSuffix(fixedName, ".svg") {
			fixedName = strings.TrimSuffix(fixedName, ".svg") + ".png"
		}
		fixedName = "../images/" + fixedName
		fixedName = strings.ToLower(fixedName)
		replaces[string(img)] = fixedName
	}

	imagesInMD := r.FindAll(f, -1)
	extensions := []string{"jpg", "png", "svg"}
	for _, i := range imagesInMD {
		var fixedName string
		var extIndex int
		for _, e := range extensions {
			extIndex = strings.Index(strings.ToLower(string(i)), "."+e)
			if extIndex > 0 {
				fixedName = string(i[0 : extIndex+1+3])
				break
			}
		}

		if extIndex == -1 {
			continue
		}

		fixedName = strings.TrimPrefix(string(i), "(")
		fixedName = strings.TrimSuffix(fixedName, ")")
		clean := path.Clean(path.Join(d, fixedName))
		fixedName = strings.TrimPrefix(clean, path.Clean(basePath))
		fixedName = strings.TrimPrefix(fixedName, "/")
		fixedName = strings.ReplaceAll(fixedName, "/", "_")

		if strings.HasSuffix(fixedName, ".svg") {
			fixedName = strings.TrimSuffix(fixedName, ".svg") + ".png"
			//println("replacing svg " + string(i) + " with " + fixedName)
		}

		fixedName = "(../images/" + fixedName + ")"
		fixedName = strings.ToLower(fixedName)

		replaces[string(i)] = fixedName
	}

	for k, v := range replaces {
		//fmt.Printf("replacing MD: %s -> %s\n", k, v)
		f = bytes.ReplaceAll(f, []byte(k), []byte(v))
	}

	return f
}

func FixHTMLImages(f []byte, fullPath string, basePath string) []byte {
	d := path.Dir(fullPath)
	inHTML := imageInHTML.FindAllStringSubmatch(string(f), -1)

	for _, ii := range inHTML {
		for n, i := range ii {
			if n == 0 {
				continue
			}

			var fixedName string
			if strings.HasPrefix(i, "/azure/architecture") {
				fixedName = strings.TrimPrefix(i, "/azure/architecture")
			} else {
				fixedName = path.Clean(path.Join(d, i))
			}
			fixedName = strings.TrimPrefix(fixedName, path.Clean(basePath))
			fixedName = strings.TrimPrefix(fixedName, "/")
			fixedName = strings.ReplaceAll(fixedName, "/", "_")
			if strings.HasSuffix(fixedName, ".svg") {
				fixedName = strings.TrimSuffix(fixedName, ".svg") + ".png"
				//fmt.Printf("replacing svg: %s -> %s\n", i, fixedName)
			}
			fixedName = "../images/" + fixedName
			fixedName = strings.ToLower(fixedName)
			f = bytes.ReplaceAll(f, []byte(i), []byte(fixedName))

			//fmt.Printf("replacing html: %s -> %s\n", i, fixedName)
		}
	}

	return f
}

func FixImages(f []byte, fullPath string, basePath string) []byte {

	f = FixMDImages(f, fullPath, basePath)
	f = FixHTMLImages(f, fullPath, basePath)

	return f
}

func GetContents(filepath string) (data []byte, contentPath string, err error) {
	ext := filepath[len(filepath)-3:]

	switch ext {
	case "yml":
		data, contentPath, err = MDFromYML(filepath)
		if err != nil {
			return nil, "", err
		}
	case ".md":
		if data, err = os.ReadFile(filepath); err != nil {
			return nil, "", fmt.Errorf("cannot read MD: %w", err)
		}
		contentPath = filepath
	default:
		//println("unrecognized extension " + item.Href)
	}

	return data, contentPath, nil
}

// ImageFromTripleColon convert `:::image` to md image
func ImageFromTripleColon(f []byte) []byte {
	vv := tripleColon.FindAllStringSubmatch(string(f), -1)
	for _, v := range vv {
		src := ""
		alt := ""
		for n, i := range v {
			if n == 0 {
				alts := altText.FindStringSubmatch(i)
				if len(alts) > 1 {
					alt = alts[1]
				}
			} else {
				src = i
			}

		}

		if src != "" {
			tag := fmt.Sprintf("![%s](%s)", alt, src)

			f = bytes.Replace(f, []byte(v[0]), []byte(tag), -1)
		}
	}

	return f
}

func RemoveYMLHeader(f []byte) []byte {
	if bytes.HasPrefix(f, []byte("---")) {
		parts := bytes.Split(f, []byte("---"))
		if len(parts) > 1 {
			f = []byte{}
			for _, v := range parts[2:] {
				f = append(f, v...)
			}
		}
	}

	return f
}

// AddPageTitle generate page title if it is not present
func AddPageTitle(f []byte, name string) []byte {
	if len(f) < len(name)*2 {
		f = append([]byte("## "+name+"\n"), f...)
	} else {
		if !strings.Contains(strings.ToLower(string(f)[0:len(name)*2]), "# "+strings.ToLower(name)) {
			f = append([]byte("## "+name+"\n"), f...)
		}
	}

	return f
}

func Render(f []byte, renderer markdown.Renderer) string {
	p := parser.NewWithExtensions(parser.CommonExtensions)
	md := p.Parse(f)
	text := string(markdown.Render(md, renderer))
	return text
}

func SaveToEpub(e *epub.Epub, text string, contentPath string, parent string, basePath string, cssPath string, name string) (filename string) {
	var fullPathSafe string
	var err error

	if len(contentPath) > 0 {
		fullPathSafe = strings.ReplaceAll(contentPath[len(basePath):], "/", "_")
	}

	if parent == "" {
		filename, err = e.AddSection(text, name, fullPathSafe, cssPath)
	} else {
		filename, err = e.AddSubSection(parent, text, name, fullPathSafe, cssPath)
	}

	if err != nil {
		if !strings.HasPrefix(err.Error(), "Filename already used") {
			slog.Error("unexpected epub error", "error", err.Error())
			os.Exit(1)
		}
	}

	return filename
}

func ItemToEpub(item TocItem, e *epub.Epub, parent string, cssPath string, basePath string, renderer markdown.Renderer) (filename string) {
	var err error

	// href is too small for "*.yml/md" or just empty
	if len(item.Href) < 5 {
		if parent == "" {
			filename, err = e.AddSection(item.Name, item.Name, "", cssPath)
		} else {
			filename, err = e.AddSubSection(parent, item.Name, item.Name, "", cssPath)
		}

		if err != nil {
			panic(err)
		}

		if len(item.Items) > 0 {
			for _, i := range item.Items {
				ItemToEpub(i, e, filename, cssPath, basePath, renderer)
			}
		}

		return filename
	}

	f, contentPath, err := GetContents(path.Join(basePath, item.Href))
	if err != nil {
		slog.Error("cannot read contents of toc item", "error", err.Error())
		return ""
	}

	f = ImageFromTripleColon(f)
	f = RemoveYMLHeader(f)
	f = FixImages(f, contentPath, basePath)
	f = AddPageTitle(f, item.Name)

	text := Render(f, renderer)

	filename = SaveToEpub(e, text, contentPath, parent, basePath, cssPath, item.Name)

	if len(item.Items) > 0 {
		for _, i := range item.Items {
			ItemToEpub(i, e, filename, cssPath, basePath, renderer)
		}
	}

	return filename
}

func (t *Toc) ToEPUB(e *epub.Epub, basePath string, renderer markdown.Renderer, csspath string) {
	getPics(basePath, e)
	for _, i := range t.Items {
		ItemToEpub(i, e, "", csspath, basePath, renderer)
	}
}

// this code is poorly structured; I am not proud of it
// still works

func main() {
	var basePath string
	var output string
	title := "Azure Architecture Center"
	outDefault := strings.Replace(title, " ", "_", -1) + ".epub"
	flag.StringVar(&basePath, "path", "./architecture-center/docs", "path to `docs` dir")
	flag.StringVar(&output, "out", outDefault, "output file")
	flag.Parse()

	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	e, err := epub.NewEpub(title)
	if err != nil {
		log.Println(err)
	}

	e.SetAuthor("Microsoft")

	d, err := emb.ReadFile("epub.css")
	if err != nil {
		log.Fatalf("cannot read embed file epub.css: %s\n", err.Error())
	}

	cssPath, _ := e.AddCSS(dataurl.EncodeBytes(d), "epub.css")

	tocBytes, err := os.ReadFile(path.Join(basePath, "toc.yml"))
	if err != nil {
		log.Fatalf("cannot read yml: %s\n", err.Error())
	}

	toc := &Toc{}

	err = yaml.Unmarshal(tocBytes, toc)
	if err != nil {
		log.Fatalf("cannot unmarshal toc: %s\n", err.Error())
	}

	toc.ToEPUB(e, basePath, renderer, cssPath)

	err = e.Write(output)
	if err != nil {
		log.Fatalf("cannot write epub: %s\n", err.Error())
	}
}
