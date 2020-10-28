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
	log.Printf(Green + "[INFO]" + Reset + ": " + str)
}

func ErrorLog(str string) {
	log.Printf(Red + "[ERROR]" + Reset + ": " + str)
}

func MessageLog(sender, method, msg string) {
	log.Printf(Blue+"Message Received"+Reset+": %s says (via %s): %s", sender, method, msg)
}

func CheckError(err error) {

	if err != nil {
		log.Printf("<<Error>>: %s", err.Error())
		os.Exit(1)
	}

}
