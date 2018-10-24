package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
	"strconv"
	"strings"
)

const csvExt = ".csv"

// To enable completions in Bash: eval $(lookup complete)
//
const completeScriptForBash = `_lookup_complete() { IFS=$'\n' COMPREPLY=($(compgen -W "$(lookup complete "$COMP_CWORD" "${COMP_WORDS[@]}")" -- "${COMP_WORDS[COMP_CWORD]}")); }; complete -o nospace -F _lookup_complete lookup`

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func maxStringLen(strings []string) int {
	maxlen := 0
	for _, s := range strings {
		if maxlen < len(s) {
			maxlen = len(s)
		}
	}
	return maxlen
}

func getSystemPath() string {
	return "/usr/local/share/lookup"
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
	ans = append(ans, getSystemPath())
	return ans
}

func getPath() []string {
	// TODO: LOOKUP_PATH envar overrides the default
	return getDefaultPath()
}

func tryIn(pathDir, kind, key string, anyKeyFound bool) bool {
	// Add spaces around both the needle and the haystack so that
	// we can match individual words (but not parts of words) as
	// well as multi-word sequences.
	lowKey := " " + strings.ToLower(key) + " "
	fullPath := path.Join(pathDir, kind+csvExt)
	file, err := os.Open(fullPath)
	if err != nil {
		return false
	}
	defer file.Close()
	found := false
	reader := csv.NewReader(file)
	fieldTitles := []string{}
	maxTitleLen := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s %v\n", fullPath, err)
			break
		}
		if len(fieldTitles) == 0 {
			fieldTitles = record
			maxTitleLen = maxStringLen(fieldTitles)
			continue
		}
		recordMatches := false
		for _, field := range record {
			lowField := strings.ToLower(" " + field + " ")
			if strings.Contains(lowField, lowKey) {
				recordMatches = true
			}
		}
		if recordMatches {
			if anyKeyFound {
				fmt.Println()
			}
			for i := 0; i < len(fieldTitles); i++ {
				title := fieldTitles[i]
				field := record[i]
				gap := maxTitleLen - len(title)
				fmt.Println(title + ":" +
					strings.Repeat(" ", gap) + " " +
					field)
			}
			found = true
			anyKeyFound = true
		}
	}
	return found
}

func lookup(kind string, keys []string) {
	anyKeyFound := false
	missing := []string{}
	for _, key := range keys {
		keyFound := false
		for _, pathDir := range getPath() {
			if tryIn(pathDir, kind, key, anyKeyFound) {
				keyFound = true
				anyKeyFound = true
			}
		}
		if !keyFound {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		if anyKeyFound {
			fmt.Println()
		}
		for _, key := range missing {
			fmt.Printf("Not found: %s\n", key)
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
