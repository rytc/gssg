package gssg

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
)

const DEFAULT_FILE_PERM fs.FileMode = 0775

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

			destFileName := dest + "/" + f.Name()
			srcFileName := src + "/" + f.Name()

			destFileInfo, err := os.Stat(destFileName)

			if err == nil {
				// Only copy the file if the source file was updated after the destination file
				if f.ModTime().Before(destFileInfo.ModTime()) {
					//log.Println("Skipping " + srcFileName)
					continue
				}
			} else {
				log.Println(err.Error())
			}

			content, err := ioutil.ReadFile(src + "/" + f.Name())
			if err != nil {
				return err
			}

			log.Println("Copying " + srcFileName + " to " + destFileName)
			err = ioutil.WriteFile(dest+"/"+f.Name(), content, DEFAULT_FILE_PERM)
			if err != nil {
				return err
			}
		}
	}

	return err
}
