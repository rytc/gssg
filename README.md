# gssg
gssg is my custom static site generator written as a replacement of Hugo for my portfolio site.

[See portfolio site source here](https://github.com/rytc/rytcio)

Currently a WIP

## How it works
Run `gssg init` to setup the initial directory structure in an empty directory.

The directory structure is setup as following:

- **./templates:** The site template for site-wide header and footer go
- **./pages:** HTML pages with templating logic
- **./static:** Static content such as images, css, javascript. 
- **./content:** data/content that is specially processed for use in pages and templates
  - **./content/blog:** Markdown blog posts (Not yet implemented)
  - **./content/projects:** YAML files describing projects
- **./public:** The final generated site HTML

Running `gssg build` parses the templates, pages, and content then generates a static site and places it into `./public`. The content in `./static` gets copied directly to `./public`

`gssg server` runs a local server to test the site.

## TODOs
- Add directory watching while the local server is running so that the user doesn't have to close the server process, run `gssg build` then run `gssg server` again
- Blog post support