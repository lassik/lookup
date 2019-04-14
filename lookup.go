// Copyright 2017-2019 Lassi Kortela
// SPDX-License-Identifier: ISC

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
	"sort"
	"strconv"
	"strings"
)

const csvExt = ".csv"

// To enable completions in Bash: eval $(lookup complete script bash)
//
const completeScriptBash = `_lookup_complete() { IFS=$'\n' COMPREPLY=($(compgen -W "$(lookup complete arg "$COMP_CWORD" "${COMP_WORDS[@]}")" -- "${COMP_WORDS[COMP_CWORD]}")); }; complete -o nospace -F _lookup_complete lookup`

func maxStringLen(strings []string) int {
	maxlen := 0
	for _, s := range strings {
		if maxlen < len(s) {
			maxlen = len(s)
		}
	}
	return maxlen
}

func getHome() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return usr.HomeDir
}

func getSystemPath() string {
	return "/usr/local/share/lookup"
}

func getDefaultPath() []string {
	return []string{path.Join(getHome(), ".lookup"), getSystemPath()}
}

func getPath() []string {
	// TODO: LOOKUP_PATH envar overrides the default
	return getDefaultPath()
}

func getAllTablesInDir(tables map[string][]string, dir string, depth int) {
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
		} else if info.Mode().IsDir() && depth < 5 {
			getAllTablesInDir(tables, fullPath, depth+1)
		}
	}
}

func getAllTables() map[string][]string {
	tables := map[string][]string{}
	for _, dir := range getPath() {
		// TODO: Assert abs path?
		getAllTablesInDir(tables, dir, 1)
	}
	return tables
}

func lookupKeyInFile(key, csvFile string, anyKeyFound bool) bool {
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
			if lookupKeyInFile(key, csvFile, anyKeyFound) {
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

func usage() {
	log.Fatal("usage: lookup table item [item ...]")
}

func compUsage() {
	log.Print("usage: lookup complete script bash")
	log.Fatal("usage: lookup complete arg [arg-index] arg1 [arg2 ... argn]")
}

func complete(args []string) {
	if len(args) < 1 {
		compUsage()
	}
	switch args[0] {
	case "script":
		if len(args) != 2 {
			compUsage()
		}
		switch args[1] {
		case "bash":
			fmt.Println(completeScriptBash)
		default:
			compUsage()
		}
	case "arg":
		if len(args) < 2 {
			compUsage()
		}
		argi, err := strconv.Atoi(args[1])
		if err != nil {
			log.Fatal(err)
		}
		args = append([]string{""}, args[2:]...)
		if argi < 1 {
			log.Fatal("arg index too low")
		}
		if argi == len(args) {
			args = append(args, "")
		}
		if argi >= len(args) {
			log.Fatal("arg index too high")
		}
		comps := []string{}
		if argi == 1 {
			for kind, _ := range getAllTables() {
				comps = append(comps, kind)
			}
		}
		sort.Strings(comps)
		for _, comp := range comps {
			fmt.Printf("%s \n", comp)
		}
	default:
		compUsage()
	}
}

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		usage()
	} else if args[0] == "complete" {
		complete(args[1:])
	} else {
		lookup(args[0], args[1:])
	}
}
