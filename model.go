package main

import "time"

/*
Goals of this
- Render menu in groups
- Include most recent article in home page
- Include most recent comic in home page
- Render list of articles (pagination of some sort?)
- Render list of comics somehow
- Include most recent N article(s) in menu
- Include most recent N comic(c) in menu
- Dynamically detect new files
*/

type page struct {
	Title    string
	Filename string
	Date     time.Time
}

type folder struct {
	Title    string
	Filename string
	Pages    []page // Sorted by reverse date
}

type site struct {
	TaggedPages map[string][]page
	// All sections

}
