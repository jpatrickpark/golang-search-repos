/**
 * Copyright 2016 Jungkyu Park
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package handlers

import (
	"encoding/json"
	"github.com/adam-hanna/arrayOperations"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

type PackageApiResult struct {
	Package    string `json:"Package"`
	StarCount  int    `json:"StarCount"`
	ProjectURL string `json:"ProjectURL"`
}

type CustomResult struct {
	Package   string `json:"package"`
	Author    string `json:"author"`
	StarCount int    `json:"StarCount"`
}

type Foo struct {
	Imported []string
}

type Bar struct {
	Query string `json:"query"`
	Hits  []struct {
		Package string `json:"package"`
		Author  string `json:"author"`
	} `json:hits`
}

func getJson(url string, target interface{}) error {
	r, err := http.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}

type PackageApiResultList []PackageApiResult

func (slice PackageApiResultList) Len() int {
	return len(slice)
}

func (slice PackageApiResultList) Less(i, j int) bool {
	return slice[i].StarCount > slice[j].StarCount
}

func (slice PackageApiResultList) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

func ImportedRepo(url string) []string {
	// takes human readable repo url, returns a list of imported repos through go-search url
	foo := Foo{}
	getJson("http://go-search.org/api?action=package&id="+FitCharsToURL(url), &foo)
	return foo.Imported
}

func CommonRepo(urls []string) []string {
	lenUrls := len(urls)
	if lenUrls <= 0 {
		return []string{}
	}
	c := make(chan string, lenUrls)
	// put urls into a channel
	for _, elem := range urls {
		c <- elem
	}
	close(c)

	// build resultset from first element
	resultSet := ImportedRepo(<-c)
	for elem := range c {
		tempSet := ImportedRepo(elem)
		resultSet = arrayOperations.IntersectString(resultSet, tempSet)
	}
	return resultSet
}

func HumanFromRepo(repos []string) {
	for i, _ := range repos {
		repos[i] = strings.Join(strings.Split(repos[i], "/")[:2], "/") //
	}
}

func CommonHuman(urls []string) []string {
	lenUrls := len(urls)
	if lenUrls <= 0 {
		return []string{}
	}
	c := make(chan string, lenUrls)
	// put urls into a channel
	for _, elem := range urls {
		c <- elem
	}
	close(c)

	// build resultset from first element
	resultSet := ImportedRepo(<-c)
	HumanFromRepo(resultSet)
	resultSet = arrayOperations.DistinctString(resultSet)

	for elem := range c {
		tempSet := ImportedRepo(elem)
		HumanFromRepo(tempSet)
		tempSet = arrayOperations.DistinctString(resultSet)
		resultSet = arrayOperations.IntersectString(resultSet, tempSet)
	}
	return resultSet
}

func PackageApi(url string) PackageApiResult {
	// takes human readable repo url, returns a list of imported repos through go-search url
	packageApiResult := PackageApiResult{}
	getJson("http://go-search.org/api?action=package&id="+FitCharsToURL(url), &packageApiResult)
	return packageApiResult
}

func PostIntersectHuman(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// urlList should be given from http request
	input := r.FormValue("packages")
	if len(input) == 0 {
		return
	}
	urlList := strings.Split(input, ", ") //
	// if the last item is empty, get rid of it
	if urlList[len(urlList)-1] == "" {
		urlList = urlList[:len(urlList)-1]
	}
	// get the intersection of common imported repositories
	resultSet := CommonHuman(urlList)
	jsonOutput, err := json.Marshal(resultSet)

	if err != nil {
		//handle error appropriately here
		return
	}
	w.Write(jsonOutput)
}

func PostIntersectRepo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// urlList should be given from http request
	input := r.FormValue("packages")
	if len(input) == 0 {
		return
	}
	urlList := strings.Split(input, ", ")
	// if the last item is empty, get rid of it
	if urlList[len(urlList)-1] == "" {
		urlList = urlList[:len(urlList)-1]
	}

	// get the intersection of common imported repositories
	resultSet := CommonRepo(urlList)
	packageApiList := PackageApiResultList{}

	// process top 30 items
	for i, item := range resultSet {
		packageApiList = append(packageApiList, PackageApi(item))
		if i > 27 {
			break
		}
	}

	sort.Sort(packageApiList)
	jsonOutput, err := json.Marshal(packageApiList)

	if err != nil {
		//handle error appropriately here
		return
	}
	w.Write(jsonOutput)
}

func StarCount(url string) int {
	// takes human readable repo url, returns a list of imported repos through go-search url
	packageApiResult := PackageApiResult{}
	getJson("http://go-search.org/api?action=package&id="+FitCharsToURL(url), &packageApiResult)
	return packageApiResult.StarCount
}

func FitCharsToURL(query string) string {
	return (&url.URL{Path: query}).String()
}

func GetSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// urlList should be given from http request
	query := r.URL.Query().Get("query")
	if len(query) == 0 {
		//handle error appropriately here
		return
	}
	bar := Bar{}
	getJson("http://go-search.org/api?action=search&q="+FitCharsToURL(query), &bar)

	// bar has all the search result. Now I am getting additional information for the packages.
	customList := []CustomResult{}
	for i, item := range bar.Hits {
		customList = append(customList, CustomResult{item.Package, item.Author, StarCount(item.Package)})
		if i > 3 {
			break
		}
	}
	jsonOutput, err := json.Marshal(customList)

	if err != nil {
		//handle error appropriately here
		return
	}
	w.Write(jsonOutput)
}
