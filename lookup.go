package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
	"strconv"
	"strings"
)

const systemPath = "/usr/local/share/lookup"

const csvExt = ".csv"

// To enable completions in Bash: eval $(lookup complete)
//
const completeScriptForBash = `_lookup_complete() { IFS=$'\n' COMPREPLY=($(compgen -W "$(lookup complete "$COMP_CWORD" "${COMP_WORDS[@]}")" -- "${COMP_WORDS[COMP_CWORD]}")); }; complete -o nospace -F _lookup_complete lookup`

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func getHome() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return usr.HomeDir
}

func getDefaultPath() []string {
	var ans []string
	ans = append(ans, path.Join(getHome(), ".lookup"))
	ans = append(ans, systemPath)
	return ans
}

func getPath() []string {
	// TODO: LOOKUP_PATH envar overrides the default
	return getDefaultPath()
}

func tryIn(pathDir, kind, key string) bool {
	lowKey := strings.ToLower(key)
	fullPath := path.Join(pathDir, kind+csvExt)
	file, err := os.Open(fullPath)
	if err != nil {
		return false
	}
	defer file.Close()
	found := false
	reader := csv.NewReader(file)
	for record, err := reader.Read(); err == nil; record, err = reader.Read() {
		recordMatches := false
		for _, field := range record {
			if strings.ToLower(strings.TrimSpace(field)) == lowKey {
				recordMatches = true
			}
		}
		if recordMatches {
			fmt.Println(record)
			found = true
		}
	}
	if err != nil {
		fmt.Printf(" > Failed!: %v\n", err)
	}
	return found
}

func lookup(kind string, keys []string) {
	for _, key := range keys {
		keyFound := false
		for _, pathDir := range getPath() {
			keyFound = tryIn(pathDir, kind, key)
			if keyFound {
				break
			}
		}
		if !keyFound {
			fmt.Printf("Cannot find %s\n", key)
		}
	}
}

func listCompletions(i int, args []string) {
	if i != 1 {
		return
	}
	var allNames []string
	for _, pathDir := range getPath() {
		fileInfos, _ := ioutil.ReadDir(pathDir)
		for _, fileInfo := range fileInfos {
			name := fileInfo.Name()
			if strings.HasSuffix(name, csvExt) {
				nameStem := name[:len(name)-len(csvExt)]
				allNames = append(allNames, nameStem)
			}
		}
	}
	for _, name := range allNames {
		fmt.Printf("%s \n", name)
	}
}

func main() {
	flag.Parse()
	if flag.Args()[0] == "complete" {
		if len(flag.Args()) == 1 {
			fmt.Println(completeScriptForBash)
			return
		}
		i, err := strconv.Atoi(flag.Args()[1])
		if err != nil {
			panic(err)
		}
		listCompletions(i, flag.Args()[2:])
		return
	}
	lookup(flag.Args()[0], flag.Args()[1:])
}
