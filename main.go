package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

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
		err = os.Mkdir(dest, 0755)
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

			err = ioutil.WriteFile(dest+"/"+f.Name(), content, 0755)
			if err != nil {
				return err
			}
		}
	}

	return err
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

func ReadProjects() []Project {

	files, err := ioutil.ReadDir("content/projects/featured")
	if err != nil {
		log.Fatal(err.Error())
	}

	featuredProj := make([]Project, len(files))

	for i, f := range files {
		projFile, err := ioutil.ReadFile("content/projects/featured/" + f.Name())
		if err != nil {
			log.Fatal("Failed to read projectfile")
		}
		projData := Project{}
		err = json.Unmarshal(projFile, &projData)
		if err != nil {
			log.Fatal(err.Error())
		}

		featuredProj[i] = projData
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
	projects := ReadProjects()

	for _, proj := range projects {
		fmt.Println("Reading project " + proj.Title)
	}

	data := PageData{
		SiteName:    "rytc.io",
		PageTitle:   "test page title",
		SiteContent: make(map[string]interface{}),
	}

	data.SiteContent["projects"] = projects

	tplFile, err := ioutil.ReadFile("pages/index.html")
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
	templates["main"].Execute(os.Stdout, data)
}

func AssertMkdir(err error, msg string) {
	if err != nil {
		log.Println(msg)
		log.Fatal(err.Error())
	}
}

func InitNewSite() {
	AssertMkdir(os.Mkdir("static", 0755), "Error creating directory ./static")
	AssertMkdir(os.Mkdir("public", 0755), "Error creating directory ./public")
	AssertMkdir(os.Mkdir("templates", 0755), "Error creating directory ./templates")
	AssertMkdir(os.Mkdir("content", 0755), "Error creating directory ./templates")
	AssertMkdir(os.Mkdir("pages", 0755), "Error creating directory ./templates")

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
