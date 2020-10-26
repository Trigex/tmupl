package main

import (
	"encoding/json"
	"fmt"
	"github.com/atotto/clipboard"
	"github.com/gen2brain/beeep"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

func main() {
	args := os.Args[1:]

	if len(args) >= 1 {
		file := args[0]
		// Make sure file to upload exists
		if fileExists(file) {
			// Get home path in order to find config directory
			userDir, err := os.UserHomeDir()
			if err != nil {
				errorMsgQuit("Unable to find your users' home directory!")
			}

			// Get config struct
			confStruct := handleConfig(userDir)

			// Get file mime type
			mimeType, err := exec.Command("file", "-b", "--mime-type", file).Output()
			if err != nil {
				errorMsgQuit(err.Error())
			}

			fmt.Println("Uploading...")

			// Upload the file
			res, err := exec.Command("curl", "-s", "-F", "file=@\""+file+"\";type="+string(mimeType), "-H", "Authorization: Bearer "+confStruct.Token, "-X", "POST", confStruct.ApiRoot+"media/upload").Output()
			if err != nil {
				errorMsgQuit(err.Error())
			}

			// Convert upload response to struct
			var upRes uploadResponse
			err = json.Unmarshal(res, &upRes)
			if err != nil {
				errorMsgQuit(err.Error())
			}

			// Only on if the upload was unsuccessful
			if upRes.Status == "success" {
				// Grab upload ID
				id := upRes.Id

				println("Tagging...")

				// Default tags
				tags := confStruct.Tags
				// TODO: Grab tags from command line
				for i := 1; i < len(args); i++ {
					// Grab new cli tag
					tag := args[i]
					tag = strings.ToLower(tag)
					tag = strings.ReplaceAll(tag, " ", "_")
					// Append tag
					tags = append(tags, tag)
				}

				// Upload tags
				tagRes, err := exec.Command("curl", "-s", "-H", "Content-Type: application/x-www-form-urlencoded", "-H", "Authorization: Bearer "+confStruct.Token, "-X", "POST", "--data", tagsToBody(tags), confStruct.ApiRoot+"media/"+id+"/edit").Output()
				if err != nil {
					errorMsgQuit(err.Error())
				}

				// Convert tag set response to struct
				var statusRes statusResponse
				err = json.Unmarshal(tagRes, &statusRes)
				if err != nil {
					errorMsgQuit(err.Error())
				}

				// Go on if tags uploaded correctly (possibly change this and still give the url even if tags failed?)
				if statusRes.Status == "success" {
					// Get just the filename if file is a path
					if strings.Contains(file, "/") {
						fnr := []rune(file)
						file = string(fnr[strings.LastIndex(file, "/")+1 : len(file)])
					}

					// Construct url and print to stdout
					url := confStruct.FileRoot + id + "/" + url.QueryEscape(file)
					fmt.Println(url)

					// Copy url to clipboard
					clipboard.WriteAll(url)
					// Notification on upload success
					_ = beeep.Notify("tmupl", file+" uploaded to "+url, "")
				}
			} else {
				fmt.Println(upRes)
				errorMsgQuit("Failed to upload file!")
			}
		} else {
			errorMsgQuit("Unable to find file at path.")
		}
	} else {
		errorMsgQuit("Please provide a file path argument.")
	}
}

func handleConfig(userDir string) config {
	confDir := userDir + "/.config/tmupl/"
	confFile := confDir + "config.json"

	// create config files if they don't exist
	if _, err := os.Stat(confDir); os.IsNotExist(err) {
		os.Mkdir(confDir, 0755)
	}

	if _, err := os.Stat(confFile); os.IsNotExist(err) {
		// init struct to write as default
		defaultConf := &config{
			Token:    "YOUR API KEY TOKEN, GET IT AT THE API KEYS PAGE",
			FileRoot: "https://static.termer.net/download/",
			ApiRoot:  "https://static.termer.net/api/v1/",
			Tags:     []string{"tmupl_upload"},
		}
		// marshal to json
		defaultConfString, _ := json.Marshal(defaultConf)
		// write to file
		ioutil.WriteFile(confFile, []byte(defaultConfString), 0644)

		fmt.Println("Created config file at " + confFile + ", please supply it with valid data!")
		os.Exit(1)
	}

	// read config
	confJson, err := ioutil.ReadFile(confFile)
	var confStruct config
	if err != nil {
		errorMsgQuit("Unable to reaad config file!")
	}

	// unmarshal config to struct
	err = json.Unmarshal(confJson, &confStruct)
	if err != nil {
		fmt.Println(err)
		errorMsgQuit("Unable to unmarshal json file into config struct!")
	}

	return confStruct
}

func tagsToBody(tags []string) string {
	str := "tags="
	tagsJson, _ := json.Marshal(tags)
	str += url.QueryEscape(string(tagsJson))
	return str
}

func errorMsgQuit(message string) {
	fmt.Println(message)
	os.Exit(1)
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

type config struct {
	Token    string   `json:"token"`
	FileRoot string   `json:"fileRoot"`
	ApiRoot  string   `json:"apiRoot"`
	Tags     []string `json:"tags"`
}

type uploadResponse struct {
	Id     string `json:"id"`
	Status string `json:"status"`
}

type statusResponse struct {
	Status string `json:"status"`
}
