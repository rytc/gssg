package main

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-yaml/yaml"
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
	ImageURL    string
	URLs        []string
	Description string
}

type BlogPost struct {
	Title   string
	Date    time.Time
	Draft   bool
	Content string
}

type BlogPostList []BlogPost

func (a BlogPostList) Len() int { return len(a) }
func (a BlogPostList) Less(i, j int) bool {
	return a[j].Date.Before(a[i].Date)
}
func (a BlogPostList) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
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
		log.Println("Failed to read file " + fileName)
		return result
	}

	if string(val) != "---" {
		log.Println("Missing meta data start delimeter in " + fileName)
	} else {
		val, _, err = buffReader.ReadLine()
		if err != nil {
			return result
		}
		for string(val) != "---" {
			readLength += len(string(val))
			split := strings.Split(string(val), ":")

			if split[0] == "title" {
				// Need a join here just in case there are multiple : in the line
				result.Title = strings.Join(split[1:], ":")
				log.Println("Found title: " + result.Title)
			} else if split[0] == "date" {
				result.Date, err = time.Parse("2006-01-02", strings.TrimSpace(split[1]))
				if err != nil {
					log.Println("Error parsing date " + split[1])
					log.Println(err.Error())
				}
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

func ReadBlogPosts(dir string) BlogPostList {
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

			//fmt.Println(newPost.Content)

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

			if fileNameSplit[len(fileNameSplit)-1] != "yaml" {
				log.Println("Skipping project file " + file.Name() + " due to unknown file extension")
				continue
			}

			projData := Project{}
			err = yaml.Unmarshal(projFile, &projData)
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

func GetDomain(s string) template.HTML {
	url, _ := url.Parse(s)
	urlParts := strings.Split(url.Host, ".")

	return template.HTML(urlParts[len(urlParts)-2])
}

func BuildSite() {
	err := CopyDir("static", "public")
	if err != nil {
		log.Println("Error building site...")
		log.Fatal(err.Error())
	}

	templates := ReadTemplates()

	featuredProjects := ReadProjects("content/projects/featured/")

	for _, proj := range featuredProjects {
		log.Println("imageURL: " + proj.ImageURL)
	}

	miniProjects := ReadProjects("content/projects/mini/")
	retiredProjects := ReadProjects("content/projects/retired/")

	blogPosts := ReadBlogPosts("content/blog/")

	sort.Sort(BlogPostList(blogPosts))

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
		tpl, err := template.New("index.html").Funcs(template.FuncMap{
			"noescape":  Unescape,
			"getdomain": GetDomain}).Parse(string(tplFile))
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
	AssertMkdir(os.Mkdir("content", DEFAULT_FILE_PERM), "Error creating directory ./content")
	AssertMkdir(os.Mkdir("content/blog", DEFAULT_FILE_PERM), "Error creating directory ./content/blogs")
	AssertMkdir(os.Mkdir("content/projects", DEFAULT_FILE_PERM), "Error creating directory ./content/projects")
	AssertMkdir(os.Mkdir("content/projects/featured", DEFAULT_FILE_PERM), "Error creating directory ./content/projects/featured")
	AssertMkdir(os.Mkdir("content/projects/mini", DEFAULT_FILE_PERM), "Error creating directory ./content/projects/mini")
	AssertMkdir(os.Mkdir("content/projects/retired", DEFAULT_FILE_PERM), "Error creating directory ./content/projects/retired")
	AssertMkdir(os.Mkdir("pages", DEFAULT_FILE_PERM), "Error creating directory ./pages")
}

func RunServer() {
	fs := http.FileServer(http.Dir("./public/"))
	http.Handle("/", http.StripPrefix("/", fs))

	log.Println("Listening on :1313..")
	err := http.ListenAndServe(":1313", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func PrintHelp() {
	fmt.Println("gssg - Static site generator")
	fmt.Println("Help will go here eventually")
}

func main() {

	arg := os.Args[1]

	if arg == "help" {
		PrintHelp()
	} else if arg == "init" {
		InitNewSite()
	} else if arg == "build" {
		BuildSite()
	} else if arg == "server" {
		RunServer()
	} else {
		PrintHelp()
	}

}
