package gssg

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

func LoadTemplatesFromDir(dir string) map[string]*template.Template {
	tplDir, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal(err.Error())
	}

	templates := make(map[string]*template.Template)

	for _, tpl := range tplDir {

		tplFile, err := ioutil.ReadFile(path.Join(dir, tpl.Name()))
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
