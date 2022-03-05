package gssg

import (
	"html/template"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func LoadTemplatesFromDir(dir string) map[string]*template.Template {
	tplDir, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal(err.Error())
	}

	templates := make(map[string]*template.Template)

	for _, tpl := range tplDir {

		tplFile, err := ioutil.ReadFile(filepath.Join(dir, tpl.Name()))
		if err != nil {
			log.Fatal("Failed to load template " + tpl.Name() + ": " + err.Error())
		}
		tplName := strings.TrimSuffix(string(tpl.Name()), filepath.Ext(tpl.Name()))

		log.Print("Parsing template " + tplName)
		tpl, err := template.New(tplName).Funcs(template.FuncMap{
			"noescape":     Unescape,
			"getdomain":    GetDomain,
			"gettld":       GetTLD,
			"geturltag":    GetURLTag,
			"removeurltag": RemoveURLTag}).Parse(string(tplFile))
		if err != nil {
			log.Println("Error parsing template")
			log.Fatal(err.Error())
		}

		templates[tplName] = tpl
	}

	return templates
}

func Unescape(s string) template.HTML {
	return template.HTML(s)
}

func GetDomain(s string) template.HTML {
	url, _ := url.Parse(s)
	urlParts := strings.Split(url.Host, ".")

	return template.HTML(urlParts[len(urlParts)-2])
}

func GetTLD(s string) template.HTML {
	url, _ := url.Parse(s)
	urlParts := strings.Split(url.Host, ".")

	return template.HTML(urlParts[len(urlParts)-1])
}

// A URL tag is a way for me to encode some metadata in a project link
// so that I can have a special icon and text depending on what the link is for
// for example:
//    youtube:https://youtube.com/...
//    demo:https://github.io/...
//    download:https://github.com/...
func GetURLTag(s string) template.HTML {
	urlParts := strings.Split(s, ":")
	return template.HTML(urlParts[0])
}

func RemoveURLTag(s string) template.HTML {
	urlParts := strings.Split(s, ":")
	return template.HTML(strings.Join(urlParts[1:], ":"))
}
