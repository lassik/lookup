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

func getAllTablesInDir(tables map[string][]string, dir string) {
	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Println(err)
		}
		return
	}
	for _, info := range infos {
		name := info.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		fullPath := path.Join(dir, name)
		if info.Mode()&os.ModeSymlink != 0 {
			info, err = os.Stat(fullPath)
			if err != nil {
				continue
			}
		}
		if info.Mode().IsRegular() {
			lowName := strings.ToLower(name)
			if strings.HasSuffix(lowName, csvExt) {
				lowStem := lowName[:len(lowName)-len(csvExt)]
				list := tables[lowStem]
				list = append(list, fullPath)
				tables[lowStem] = list
			}
		} else if info.Mode().IsDir() {
			getAllTablesInDir(tables, fullPath)
		}
	}
}

func getAllTables() map[string][]string {
	tables := map[string][]string{}
	for _, dir := range getPath() {
		// TODO: Assert abs path?
		getAllTablesInDir(tables, dir)
	}
	return tables
}

func tryIn(csvFile, key string, anyKeyFound bool) bool {
	// Add spaces around both the needle and the haystack so that
	// we can match individual words (but not parts of words) as
	// well as multi-word sequences.
	lowKey := " " + strings.ToLower(key) + " "
	file, err := os.Open(csvFile)
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
			fmt.Fprintf(os.Stderr, "error: %s %v\n", csvFile, err)
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
	csvFiles := getAllTables()[kind]
	if len(csvFiles) == 0 {
		fmt.Printf("No such lookup table: %s\n", kind)
	}
	anyKeyFound := false
	missing := []string{}
	for _, key := range keys {
		keyFound := false
		for _, csvFile := range csvFiles {
			if tryIn(csvFile, key, anyKeyFound) {
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
	kinds := []string{}
	for kind, _ := range getAllTables() {
		kinds = append(kinds, kind)
	}
	for _, kind := range kinds {
		fmt.Printf("%s \n", kind)
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
