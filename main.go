package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/codegangsta/negroni"
)

type filterJson struct {
	Name     string
	Patterns []string
}

type filterListJson struct {
	Name       string
	FilterList []string
}

type filter struct {
	name          string
	patterns      []string
	filterChannel chan string
	returnChannel chan filterResult
	broker        broker
}

type filterList struct {
	name string
	list []filter
}

type filterResult struct {
	Name   string
	Result bool
}

type categorizer struct {
	filters []filter
	broker  broker
}

func NewCategorizer() Categorizer {
	meatFilter := filter{
		name:          "meat",
		patterns:      []string{"chuck", "mignon", "roast", "steak"},
		filterChannel: make(chan string),
	}

	beefFilter := filter{
		name:          "beef",
		patterns:      []string{"london broil", "chuck", "tri-tip", "beef"},
		filterChannel: make(chan string),
	}

	b := broker{filterChannels: make([]chan string, 0)}

	return &categorizer{filter: []filter{meatFilter, beefFilter}, broker: b}
}

func (c *categorizer) Initialize() {
	for _, f := range c.filters {
		c.broker.AddFilter(f)
	}
}

type broker struct {
	filterChannels   []chan string
	currentInProcess int
}

func (b *broker) AddFilter(f filter) {
	b.filterChannels = append(b.filterChannels, f.filterChannel)

}

type categoryResult struct {
	Item    string
	TagList []string
}

func (b *broker) Filter(item string) {

	for _, c := range b.filterChannels {
		fmt.Println("about to send ", item)
		c <- item
		b.currentInProcess++
	}

}

func (f *filter) ListenOnChannel() {

	go func() {
		select {
		case c := <-f.filterChannel:
			result := matchPattern(c, f.patterns)
			f.returnChannel <- filterResult{Name: f.name, Result: result}
		}
	}()

}

func (f *filter) setFilterChan(fc chan string) {
	f.filterChannel = fc
}

func matchPattern(s string, patterns []string) bool {

	for _, p := range patterns {
		if strings.Contains(strings.ToLower(s), strings.ToLower(p)) {
			return true
		}
	}

	return false
}

func Serve() {
	mux := http.NewServeMux()
	mux.Handle("/categorize", c)

	n := negroni.Classic()
	n.UseHandler(mux)
	n.Run(":3001")

}

func (c *categorizer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	returnChan := make(chan filterResult)

	meatFilter.ListenOnChannel()
	beefFilter.ListenOnChannel()

	item := r.FormValue("item")
	b.Filter(item)

	tagList := make([]string, 0)

	for {
		select {
		case result := <-returnChan:
			tagList = append(tagList, result.Name)
			b.currentInProcess--
			if b.currentInProcess == 0 {
				fmt.Printf("the tag list is %v\n", tagList)
				b, err := json.Marshal(&categoryResult{Item: item, TagList: tagList})
				if err != nil {
					fmt.Println("oh no")
					return
				}
				w.Write(b)
				return
			}
		}
	}
}

func main() {
	c := &categorizer{}
	c.Serve()

}
