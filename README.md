# whisper

A web server mainly oriented at media serving for small websites.

## Conventions

See the [example](example) folder for a sample site layout. Where possible, conventions are used instead of configuration files. Conventions used by this server include:

* The `static` folder holds static content like images or JavaScript.
* The `template` folder holds HTML templates, using Go's `html/template` package.
* Certain files in the root are served directly if present: `ads.txt`, `favicon.ico`, `manifest.json`, and `robots.txt`.
* A sitemap is automatically created and rendered as `/sitemap.txt`.
* The default page for a folder is `index.md`.

## Markdown

Web pages are generally written in Markdown and use HTML templates to render into the site. The default template to use is called `default`; templates are stored in the `template` folder.

Markdown may contain *front matter* which is in TOML format. The front matter is delimited by `+++` at the start and end. For example:

    +++
    # This is my front matter
    title = "My glorious page"
    +++
    # This is my Heading
    This is my [Markdown](https://en.wikipedia.org/wiki/Markdown).

Front matter may include:

Name     | Type             | Description
---------|------------------|----------------------
title    | string           | Title of page
date     | time             | Publish date
tags     | array of strings | Tags for the article
template | string           | Override the template

