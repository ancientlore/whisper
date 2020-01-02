# whisper

A web server mainly oriented at media serving for small websites.

## Conventions

In general, _whisper_ serves static content from the location it's found - making it easy to structure your site how you want. There is special handling for certain content like Markdown files.

See the [example](example) folder for a sample site layout. In general, I want to use conventions instead of configuration files. Conventions used by this server include:

* The `template` folder holds HTML templates, using Go's `html/template` package. These templates are used for rendering content but never served directly.
* A `sitemap.txt` can be created as a template. See the [example](example) for details.
* The default page for a folder is a Markdown file called `index.md`.

## Markdown

Web pages are generally written in Markdown and use HTML templates to render into the site. The default template to use is called `default`; you must have a `default` and a `notfound` template because these are referenced directly in the code as defaults. Templates are stored in the `template` folder.

Markdown may contain *front matter* which is in TOML format. The front matter is delimited by `+++` at the start and end. For example:

    +++
    # This is my front matter
    title = "My glorious page"
    +++
    # This is my Heading
    This is my [Markdown](https://en.wikipedia.org/wiki/Markdown).

Front matter may include:

Name     | Type             | Description
---------|------------------|------------------------------------------
title    | string           | Title of page
date     | time             | Publish date
tags     | array of strings | Tags for the articles (not used yet)
template | string           | Override the template to render this file

Front matter is used for sorting and constructing lists of articles.

## Templates

_whisper_ uses standard Go templates from the `html/template` package. Templates are passed the following data:

    // frontMatter holds data scraped from a Markdown page.
    type frontMatter struct {
        Title    string    `toml:"title" comment:"Title of this page"`
        Date     time.Time `toml:"date" comment:"Date the article appears"`
        Template string    `toml:"template" comment:"The name of the template to use"`
        Tags     []string  `toml:"tags" comment:"Tags to assign to this article"`
    }

    // pageInfo has information about the current page.
    type pageInfo struct {
        Path     string
        Filename string
    }

    // data is what is passed to makedown templates.
    type data struct {
        FrontMatter frontMatter
        Page        pageInfo
        Content     template.HTML
    }

`Page` is information about the current page, and `FrontMatter` is the front-matter form the current Markdown file. `Content` contains the HTML version of the Markdown file.

Functions are added to the template for your convenience.

Function                       | Description
-------------------------------|------------
`dir(path string) []file`      | Return the contents of the given folder, excluding special files and subfolders.
`join(parts ...string) string` | The same as path.Join

`file` is defined as:

    type file struct {
        FrontMatter frontMatter
        Filename    string
    }

If `file` is not a Markdown file, then `FrontMatter.Title` is set to the file name and `FrontMatter.Date` is set to the modification time. The array is sorted by reverse date (most recent items first).

Because _whisper_ caches files that are generated, the process of building a page isn't repeated unnecessarily.

## Non-Goals

* It's not a goal to make templates reusable. I expect templates need editing for new sites.
* It's not a goal to automate creation of the menu.
* It's not a goal to be a fully-featured server. I run [Caddy](https://caddyserver.com/) in front of it.
