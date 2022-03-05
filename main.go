package main

import (
	"bufio"
	"bytes"
	"fmt"
	"gssg/gssg"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
    "path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-yaml/yaml"
	"github.com/gomarkdown/markdown"
)

type SiteConfig struct {
	SiteName string
}

var config SiteConfig

type PageData struct {
	SiteName    string
	SiteContent map[string]interface{}
	Content     template.HTML
}

type Project struct {
	Title       string
	Subtitle    string
	Date        time.Time
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
    Permalink string
}

type BlogPostList []BlogPost

func (a BlogPostList) Len() int { return len(a) }
func (a BlogPostList) Less(i, j int) bool {
	return a[j].Date.Before(a[i].Date)
}
func (a BlogPostList) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

type ProjectList []Project

func (a ProjectList) Len() int { return len(a) }
func (a ProjectList) Less(i, j int) bool {
	return a[j].Date.Before(a[i].Date)
}
func (a ProjectList) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func ParseBlogPost(fileName string, postContent []byte) BlogPost {
	result := BlogPost{}
	bytesReader := bytes.NewReader(postContent)
	buffReader := bufio.NewReader(bytesReader)

    fileNameSplit := strings.Split(fileName, "/")
    fileNameSplit = strings.Split(fileNameSplit[len(fileNameSplit)-1], ".")
    result.Permalink = fileNameSplit[0] + ".html"

	var val []byte
	var readLength int

	val, _, err := buffReader.ReadLine()
    readLength += len(string(val))

	if err != nil {
		log.Println("Failed to read file " + fileName)
		return result
	}

	if string(val) != "---" {
		log.Println("Missing meta data start delimeter in " + fileName)
	} else {
		val, _, err = buffReader.ReadLine()
        readLength += len(string(val))

		if err != nil {
			return result
		}
		for string(val) != "---" {
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
				if strings.TrimSpace(string(split[1:][0])) == "true" {
					result.Draft = true
				} else {
                    result.Draft = false
                }
			}

			val, _, err = buffReader.ReadLine()
            readLength += len(string(val))

			if err != nil {
				break
			}
		}

        readLength += len(string(val)) + 1

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
			newPosts := ReadBlogPosts(path.Join(dir, postEntry.Name()))
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

			posts = append(posts, newPost)

		}
	}

	return posts
}

func UnmarshalAllProjcts(file []byte, out *[]Project) error {
	reader := bytes.NewReader(file)
	decoder := yaml.NewDecoder(reader)

	for {
		var newProject Project
		if err := decoder.Decode(&newProject); err != nil {
			if err != io.EOF {
				return err
			}
			break
		}
		*out = append(*out, newProject)
	}

	return nil
}

func ReadProjects(projectFilename string, out *[]Project) error {

	projectFile, err := ioutil.ReadFile(projectFilename)
	if err != nil {
		return err
	}

	err = UnmarshalAllProjcts(projectFile, out)

	return err
}

func BuildSite(config SiteConfig) {

	log.Println("Copying static files...")
	err := gssg.CopyDir("static", "public")
	if err != nil {
		log.Println("Error building site...")
		log.Fatal(err.Error())
	}

	log.Println("Reading templates...")
	templates := gssg.LoadTemplatesFromDir("templates")

	log.Println("Parsing projects...")

	var featuredProjects []Project
	var miniProjects []Project
	var retiredProjects []Project

	err = ReadProjects("content/projects/featured.yaml", &featuredProjects)
	if err != nil {
		log.Println("Failed to read featured projects file. Site will still build")
	}

	err = ReadProjects("content/projects/mini.yaml", &miniProjects)
	if err != nil {
		log.Println("Failed to read mini projects file. Site will still build")
	}

	err = ReadProjects("content/projects/retired.yaml", &retiredProjects)
	if err != nil {
		log.Println("Failed to read retired projects file. Site will still build.")
	}

	blogPosts := ReadBlogPosts("content/blog/")
	sort.Sort(BlogPostList(blogPosts))

	data := PageData{
		SiteName:    config.SiteName,
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
		tplFile, err := ioutil.ReadFile(path.Join("pages", page.Name()))
		if err != nil {
			log.Fatal("Failed to load page index.html: " + err.Error())
		}
		tpl, err := template.New("index.html").Funcs(template.FuncMap{
			"noescape":     gssg.Unescape,
			"getdomain":    gssg.GetDomain,
			"gettld":       gssg.GetTLD,
			"geturltag":    gssg.GetURLTag,
			"removeurltag": gssg.RemoveURLTag}).Parse(string(tplFile))
		if err != nil {
			log.Println("Error parsing template")
			log.Fatal(err.Error())
		}

		sw := new(strings.Builder)
		tpl.Execute(sw, data)
		data.Content = template.HTML(sw.String())

		sw.Reset()
		templates["main"].Execute(sw, data)
		ioutil.WriteFile(path.Join("public", page.Name()), []byte(sw.String()), gssg.DEFAULT_FILE_PERM)

		log.Println("Writing page " + page.Name())
	}

    // Output blog posts

    for _,blogPost := range blogPosts {
        if(blogPost.Draft == true) {
            log.Println("Skipping draft...")
            continue;
        }

		sw := new(strings.Builder)
		templates["blogPost"].Execute(sw, blogPost)
		data.Content = template.HTML(sw.String())

		sw.Reset()
		templates["main"].Execute(sw, data)
		//ioutil.WriteFile("public/"+page.Name(), []byte(sw.String()), gssg.DEFAULT_FILE_PERM)

        // []byte(blogPost.Content)
        ioutil.WriteFile(path.Join("public", "blog", blogPost.Permalink), []byte(sw.String()), gssg.DEFAULT_FILE_PERM)
        log.Println("Writing blog " + blogPost.Permalink)
    }
}

func AssertMkdir(err error, msg string) {
	if err != nil {
		log.Println(msg)
		log.Fatal(err.Error())
	}
}

func InitNewSite() {
	AssertMkdir(os.Mkdir("static", gssg.DEFAULT_FILE_PERM), "Error creating directory ./static")
	AssertMkdir(os.Mkdir("public", gssg.DEFAULT_FILE_PERM), "Error creating directory ./public")
    AssertMkdir(os.Mkdir("public/blog", gssg.DEFAULT_FILE_PERM), "Error creating directory ./public/blog")
	AssertMkdir(os.Mkdir("templates", gssg.DEFAULT_FILE_PERM), "Error creating directory ./templates")
	AssertMkdir(os.Mkdir("content", gssg.DEFAULT_FILE_PERM), "Error creating directory ./content")
	AssertMkdir(os.Mkdir("content/blog", gssg.DEFAULT_FILE_PERM), "Error creating directory ./content/blogs")
	AssertMkdir(os.Mkdir("content/projects", gssg.DEFAULT_FILE_PERM), "Error creating directory ./content/projects")
	AssertMkdir(os.Mkdir("pages", gssg.DEFAULT_FILE_PERM), "Error creating directory ./pages")

    baseConfigYaml := `---
    sitename: "My Site"`

    ioutil.WriteFile("config.yaml", []byte(baseConfigYaml), gssg.DEFAULT_FILE_PERM)

    baseTemplate := `
    <!DOCTYPE html>
    <html>
    <head>
    <title>{{.SiteName}}</title>
    </head>
    <body>
        <h1>Welcome to {{.SiteName}}</h1>
        <br />
        {{.Content}}
    </body>
    </html>
    `
    ioutil.WriteFile(path.Join("templates", "main.html"), []byte(baseTemplate), gssg.DEFAULT_FILE_PERM)

    baseIndex := `
        <h2>How to use gssg</h2>
        <ul>
            <li>Images, css, etc. go into the ./static directory and will automatically copied into public</li>
            <li>Write blog posts into content/blog/ as Markdown files. Blog posts start with YAML header with the attributes: 
            title (String), date (Date), draft (Boolean). Example: <pre>
            ---
            title: My Blog Post Title
            date: 2022-01-01
            draft: false
            ---
            </pre></li>
            <li>Pages are formatted as HTML go into the ./pages directory and will directly be parsed and outputted into ./public with the same filename.</li>
            <li>Blog posts use a special template named "templates/blogPosts.html"</li>
        </ul>
    `

    ioutil.WriteFile(path.Join("pages", "index.html"), []byte(baseIndex), gssg.DEFAULT_FILE_PERM)

    log.Println("Site generation is done! Run: `gssg build` then run `gssg server`. Browse to localhost:1313 to see site running live.");


}

func AddDirToWatcher(watcher *fsnotify.Watcher, rootPath string) {

	if err := filepath.Walk(rootPath, func(path string, fi os.FileInfo, err error) error {
		if fi.IsDir() {
			return watcher.Add(path)
		}
		return nil

	}); err != nil {
		log.Println("Failed to walk path " + rootPath)
		log.Fatal(err.Error())
	}

}

func RunServer() {

	watcher, _ := fsnotify.NewWatcher()

	AddDirToWatcher(watcher, "./pages")
	AddDirToWatcher(watcher, "./static")
	AddDirToWatcher(watcher, "./templates")
	AddDirToWatcher(watcher, "./content")

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("Modified file " + event.Name + ", rebuilding")
					BuildSite(config)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("Watch error: " + err.Error())
			}
		}
	}()

	fs := http.FileServer(http.Dir("./public/"))
	http.Handle("/", http.StripPrefix("/", fs))

	log.Println("Listening on :1313..")
	err := http.ListenAndServe(":1313", nil)
	if err != nil {
		log.Fatal(err)

	}

	<-done
}

func PrintHelp() {
	fmt.Println("gssg - Static site generator")
	fmt.Println("github.com/rytc/gssg")
	fmt.Println("  gssg init     Create the initial file structure")
	fmt.Println("  gssg build    Parse all files, generate the static site into ./public")
	fmt.Println("  gssg server   Start a local server at localhost:1313")

}

func LoadConfig() {
    configFile, err := ioutil.ReadFile("config.yaml")

	if err != nil {
		log.Fatal("Failed to open config.yaml for this site")
	}

	err = yaml.Unmarshal(configFile, &config)

    if err != nil {
		log.Fatal(err.Error())
	}
}

func main() {

	arg := ""

	if len(os.Args) > 1 {
		arg = os.Args[1]
	} else {
		PrintHelp()
		return
	}

    

	if arg == "help" {
		PrintHelp()
	} else if arg == "init" {
		InitNewSite()
	} else if arg == "build" {
		LoadConfig()
        BuildSite(config)
	} else if arg == "server" {
        LoadConfig()
		RunServer()
	} else {
		PrintHelp()
	}

}
