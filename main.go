package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
)

const DEFAULT_FILE_PERM fs.FileMode = 0775

type PageData struct {
	SiteName    string
	PageTitle   string
	SiteContent map[string]interface{}
	Content     template.HTML
}

type Project struct {
	Title       string
	Subtitle    string
	Tags        []string
	Image       string
	URLs        []string
	Description string
}

type BlogPost struct {
	Title   string
	Date    time.Time
	Draft   bool
	Content string
}

func CopyDir(src, dest string) error {

	f, err := os.Open(src)
	if err != nil {
		return err
	}

	file, err := f.Stat()
	if err != nil {
		return err
	}

	if !file.IsDir() {
		return fmt.Errorf("CopyDir: source " + file.Name() + " is not a directory!")
	}

	_, err = os.Stat(dest)
	if os.IsNotExist(err) {
		err = os.Mkdir(dest, DEFAULT_FILE_PERM)
		if err != nil {
			return err
		}
	}

	files, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	for _, f := range files {
		if f.IsDir() {
			err = CopyDir(src+"/"+f.Name(), dest+"/"+f.Name())
			if err != nil {
				return err
			}
		} else {
			content, err := ioutil.ReadFile(src + "/" + f.Name())
			if err != nil {
				return err
			}

			err = ioutil.WriteFile(dest+"/"+f.Name(), content, DEFAULT_FILE_PERM)
			if err != nil {
				return err
			}
		}
	}

	return err
}

func ParseBlogPost(fileName string, postContent []byte) BlogPost {
	result := BlogPost{}
	bytesReader := bytes.NewReader(postContent)
	buffReader := bufio.NewReader(bytesReader)

	var val []byte
	var readLength int

	val, _, err := buffReader.ReadLine()

	if err != nil {
		return result
	}

	if string(val) != "---" {
		log.Println("Missing meta data start delimeter in " + fileName)
	} else {
		val, _, err = buffReader.ReadLine()
		for string(val) != "---" {
			readLength += len(string(val))
			split := strings.Split(string(val), ":")

			if split[0] == "title" {
				result.Title = string(val[1:])
			} else if split[0] == "date" {
				result.Date, err = time.Parse("MM-DD-YYYY", string(val[1:]))
			} else if split[0] == "draft" {
				if string(split[1:][0]) == "true" {
					result.Draft = true
				}
			}

			val, _, err = buffReader.ReadLine()

			if err != nil {
				break
			}
		}
	}

	result.Content = string(markdown.ToHTML(postContent[readLength:], nil, nil))

	return result
}

func ReadBlogPosts(dir string) []BlogPost {
	postsDir, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal(err.Error())
	}

	posts := []BlogPost{}

	for _, postEntry := range postsDir {
		if postEntry.IsDir() {
			newPosts := ReadBlogPosts(dir + "/" + postEntry.Name())
			posts = append(posts, newPosts...)
		} else {
			log.Println("Parsing blog post " + postEntry.Name())

			fileNameSplit := strings.Split(postEntry.Name(), ".")

			if fileNameSplit[len(fileNameSplit)-1] != "md" {
				log.Println("Skipping blog post file " + postEntry.Name() + " due to unknown file extension")
				continue
			}

			postFile, err := ioutil.ReadFile(dir + "/" + postEntry.Name())
			if err != nil {
				log.Println("Error reading blog post " + postEntry.Name())
				log.Fatal(err.Error())
			}

			newPost := ParseBlogPost(dir+postEntry.Name(), postFile)

			fmt.Println(newPost.Content)

			posts = append(posts, newPost)

		}
	}

	return posts
}

func ReadTemplates() map[string]*template.Template {
	tplDir, err := os.ReadDir("templates")
	if err != nil {
		log.Fatal(err.Error())
	}

	templates := make(map[string]*template.Template)

	for _, tpl := range tplDir {

		tplFile, err := ioutil.ReadFile("templates/" + tpl.Name())
		if err != nil {
			log.Fatal("Failed to load template " + tpl.Name() + ": " + err.Error())
		}
		tplName := strings.TrimSuffix(string(tpl.Name()), filepath.Ext(tpl.Name()))

		log.Print("Parsing template " + tplName)
		tpl, err := template.New(tplName).Parse(string(tplFile))
		if err != nil {
			log.Println("Error parsing template")
			log.Fatal(err.Error())
		}

		templates[tplName] = tpl
	}

	return templates
}

func ReadProjects(dir string) []Project {

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal(err.Error())
	}

	featuredProj := make([]Project, len(files))

	for i, file := range files {

		if file.IsDir() {
			continue
		} else {

			// TODO probably should check for trailing slash?
			projFile, err := ioutil.ReadFile(dir + file.Name())
			if err != nil {
				log.Println("Failed to read project file")
				log.Fatal(err.Error())
			}

			fileNameSplit := strings.Split(file.Name(), ".")

			if fileNameSplit[len(fileNameSplit)-1] != "json" {
				log.Println("Skipping project file " + file.Name() + " due to unknown file extension")
				continue
			}

			projData := Project{}
			err = json.Unmarshal(projFile, &projData)
			if err != nil {
				log.Fatal(err.Error())
			}

			featuredProj[i] = projData
		}
	}

	return featuredProj
}

func Unescape(s string) template.HTML {
	return template.HTML(s)
}

func BuildSite() {
	err := CopyDir("static", "public")
	if err != nil {
		log.Println("Error building site...")
		log.Fatal(err.Error())
	}

	templates := ReadTemplates()

	featuredProjects := ReadProjects("content/projects/featured/")
	miniProjects := ReadProjects("content/projects/mini/")
	retiredProjects := ReadProjects("content/projects/retired/")

	blogPosts := ReadBlogPosts("content/blog/")

	// TODO don't hardcode these
	data := PageData{
		SiteName:    "rytc.io",
		PageTitle:   "test page title",
		SiteContent: make(map[string]interface{}),
	}

	projectMap := make(map[string]interface{})
	projectMap["featuredProjects"] = featuredProjects
	projectMap["miniProjects"] = miniProjects
	projectMap["retiredProjects"] = retiredProjects
	data.SiteContent["projects"] = projectMap

	data.SiteContent["blog"] = blogPosts

	pageDir, err := os.ReadDir("pages")
	if err != nil {
		log.Fatal(err.Error())
	}

	for _, page := range pageDir {
		if page.IsDir() {
			// TODO handle making subdirectories
			// Will probably need to be recursive?
			continue
		}
		tplFile, err := ioutil.ReadFile("pages/" + page.Name())
		if err != nil {
			log.Fatal("Failed to load page index.html: " + err.Error())
		}
		tpl, err := template.New("index.html").Funcs(template.FuncMap{"noescape": Unescape}).Parse(string(tplFile))
		if err != nil {
			log.Println("Error parsing template")
			log.Fatal(err.Error())
		}

		sw := new(strings.Builder)
		tpl.Execute(sw, data)
		data.Content = template.HTML(sw.String())

		sw.Reset()
		templates["main"].Execute(sw, data)
		ioutil.WriteFile("public/"+page.Name(), []byte(sw.String()), DEFAULT_FILE_PERM)

		log.Println("Writing page " + page.Name())
	}
}

func AssertMkdir(err error, msg string) {
	if err != nil {
		log.Println(msg)
		log.Fatal(err.Error())
	}
}

func InitNewSite() {
	AssertMkdir(os.Mkdir("static", DEFAULT_FILE_PERM), "Error creating directory ./static")
	AssertMkdir(os.Mkdir("public", DEFAULT_FILE_PERM), "Error creating directory ./public")
	AssertMkdir(os.Mkdir("templates", DEFAULT_FILE_PERM), "Error creating directory ./templates")
	AssertMkdir(os.Mkdir("content", DEFAULT_FILE_PERM), "Error creating directory ./templates")
	AssertMkdir(os.Mkdir("pages", DEFAULT_FILE_PERM), "Error creating directory ./templates")

}

func PrintHelp() {
	fmt.Println("gssg - Static site generator")
	fmt.Println("Help will go here eventually")
}

func main() {

	for _, arg := range os.Args[1:] {
		if arg == "help" {
			PrintHelp()
		} else if arg == "init" {
			InitNewSite()
		} else {
			BuildSite()
		}
	}
}
