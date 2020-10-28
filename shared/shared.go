package shared

import (
	"log"
	"os"
	"runtime"
)

var Reset = "\033[0m"
var Red = "\033[31m"
var Green = "\033[32m"
var Yellow = "\033[33m"
var Blue = "\033[34m"
var Purple = "\033[35m"
var Cyan = "\033[36m"
var Gray = "\033[37m"
var White = "\033[97m"

func init() {

	// Credits to https://twinnation.org/articles/35/how-to-add-colors-to-your-console-terminal-output-in-go for colourization.
	if runtime.GOOS == "windows" {
		Reset = ""
		Red = ""
		Green = ""
		Yellow = ""
		Blue = ""
		Purple = ""
		Cyan = ""
		Gray = ""
		White = ""
	}
}

func InfoLog(str string) {
	log.Printf(color.Green + "[INFO]" + color.Reset + ": " + str)
}

func ErrorLog(fostring, args ...interface{}) {
	log.Printf(color.Red + "[ERROR]" + color.Reset + ": " + str)
}

func CheckError(err error) {

	if err != nil {
		log.Printf("<<Error>>: %s", err.Error())
		os.Exit(1)
	}

}