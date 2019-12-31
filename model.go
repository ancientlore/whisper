package main

/*
Goals of this
- Render menu in groups
- Include most recent article in home page
- Include most recent comic in home page
- Render list of articles (pagination of some sort?)
- Render list of comics somehow
- Include most recent N article(s) in menu
- Inlucde most recent N comic(c) in menu
- Dynamically detect new files
*/

type Page struct {
	Name     string
	Filename string
	// Date

}

type Section struct {
	Name  string // Title
	Pages []Page // Sorted by reverse date

}

func (obj Section) Filename() string {
	if obj.Pages != nil && len(obj.Pages) > 0 {
		return obj.Pages[0].Filename
	}
	return ""
}

type Site struct {
	// All tags to pages map
	// All sections

}
