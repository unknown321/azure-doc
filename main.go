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
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

//go:embed epub.css
var emb embed.FS

var r = regexp.MustCompile("(?i)\\(.*.(png|svg|jpg)\\)")
var imageInHTML = regexp.MustCompile(`(?i)src="(.*.(?:png|svg|jpg))"`)
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
			newName := strings.ReplaceAll(p[len(basePath)+1:], "/", "_")
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

func MDFromYML(ymlPath string) ([]byte, string, error) {
	var f []byte
	var err error
	if f, err = os.ReadFile(ymlPath); err != nil {
		return nil, "", fmt.Errorf("cannot read md from yml: %w", err)
	}
	// only first occurrence
	res := inc.FindStringSubmatch(string(f))
	if len(res) < 2 {
		return nil, "", fmt.Errorf("failed to found included md with actual contents in yml %s", ymlPath)
	}

	d := path.Dir(ymlPath)
	mdPath := path.Join(d, res[1])

	var md []byte
	if md, err = os.ReadFile(mdPath); err != nil {
		return nil, "", fmt.Errorf("cannot open md %s from yml %s: %w", ymlPath, mdPath, err)
	}
	return md, mdPath, nil
}

func FixImages(f []byte, fullPath string, basePath string) []byte {
	d := path.Dir(fullPath)
	images := r.FindAll(f, -1)

	inHTML := imageInHTML.FindAllStringSubmatch(string(f), -1)

	for _, ii := range inHTML {
		for n, i := range ii {
			if n == 0 {
				continue
			}

			clean := path.Clean(path.Join(d, i))
			fixedName := clean[len(basePath)+1:]
			fixedName = strings.ReplaceAll(fixedName, "/", "_")
			if strings.HasSuffix(fixedName, ".svg") {
				fixedName = strings.TrimSuffix(fixedName, ".svg") + ".png"
			}
			fixedName = "../images/" + fixedName
			fixedName = strings.ToLower(fixedName)
			//println("replacing HTML " + i + " with " + fixedName)
			f = bytes.ReplaceAll(f, []byte(i), []byte(fixedName))
		}
	}

	for _, i := range images {
		fixedName := strings.TrimPrefix(string(i), "(")
		fixedName = strings.TrimSuffix(fixedName, ")")
		clean := path.Clean(path.Join(d, fixedName))
		fixedName = clean[len(basePath)+1:]
		fixedName = strings.ReplaceAll(fixedName, "/", "_")

		fixedName = "(../images/" + fixedName + ")"
		fixedName = strings.ToLower(fixedName)

		//println("replacing MD" + string(i) + " with " + fixedName)
		f = bytes.ReplaceAll(f, i, []byte(fixedName))
	}

	return f
}

func ItemToEpub(item TocItem, e *epub.Epub, parent string, cssPath string, basePath string, renderer markdown.Renderer) (filename string) {
	var err error

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

	ext := item.Href[len(item.Href)-3:]

	var f []byte
	var fullPath string
	switch ext {
	case "yml":
		f, fullPath, err = MDFromYML(path.Join(basePath, item.Href))
		if err != nil {
			log.Println(err.Error())
		}
	case ".md":
		fullPath = path.Join(basePath, item.Href)
		if f, err = os.ReadFile(fullPath); err != nil {
			log.Println(fmt.Sprintf("cannot read MD: %s", err.Error()))
			return ""
		}
	default:
		//println("unrecognized extension " + item.Href)
	}

	// convert :::image to md image
	vv := tripleColon.FindAllStringSubmatch(string(f), -1)
	for _, v := range vv {
		src := ""
		alt := ""
		for n, i := range v {
			if n == 0 {
				alt = altText.FindString(i)
			} else {
				src = i
			}

		}
		if src != "" && alt != "" {
			tag := fmt.Sprintf("![%s](%s)", alt, src)

			f = bytes.Replace(f, []byte(v[0]), []byte(tag), -1)
		}
	}

	if bytes.HasPrefix(f, []byte("---")) {
		parts := bytes.Split(f, []byte("---"))
		if len(parts) > 1 {
			f = []byte{}
			for _, v := range parts[2:] {
				f = append(f, v...)
			}
		}
	}

	f = FixImages(f, fullPath, basePath)

	// generate page title if it is not present
	if len(f) < len(item.Name)*2 {
		f = append([]byte("## "+item.Name+"\n"), f...)
	} else {
		if !strings.Contains(strings.ToLower(string(f)[0:len(item.Name)*2]), "# "+strings.ToLower(item.Name)) {
			f = append([]byte("## "+item.Name+"\n"), f...)
		}
	}

	p := parser.NewWithExtensions(parser.CommonExtensions)
	md := p.Parse(f)
	text := string(markdown.Render(md, renderer))

	var fullPathSafe string
	if len(fullPath) > 0 {
		fullPathSafe = strings.ReplaceAll(fullPath[len(basePath):], "/", "_")
	}

	if parent == "" {
		filename, err = e.AddSection(text, item.Name, fullPathSafe, cssPath)
	} else {
		filename, err = e.AddSubSection(parent, text, item.Name, fullPathSafe, cssPath)
	}

	if err != nil {
		if strings.HasPrefix(err.Error(), "Filename already used") {

		} else {
			panic(err)
		}
	}

	if len(item.Items) > 0 {
		for _, i := range item.Items {
			ItemToEpub(i, e, filename, cssPath, basePath, renderer)
		}
	}

	return filename
}

func (t *Toc) ToEPUB(e *epub.Epub, basePath string, renderer markdown.Renderer, csspath string) {
	for _, i := range t.Items {
		ItemToEpub(i, e, "", csspath, basePath, renderer)
	}

	getPics(basePath, e)
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
