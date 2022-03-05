# gssg
gssg is my custom static site generator written as a replacement for Hugo for my portfolio site.

[See portfolio site source here](https://github.com/rytc/rytcio)

## Build

After downloading the source, run `go build -o gssg.exe` on Windows or `go build -o gssg` on Linux.

Tested with Go version 1.17.2.

## Use
Copy the gssg binary to an empty directory and run `gssg init`. This will setup the directory structure with basic files. After that, run `gssg build` to generate the site, then run `gssg server` to run the site on `localhost:1313`

## How it works
The directory structure is setup as follows:

- **./templates:** The site template for site-wide header and footer, or specific content like a blog post
- **./pages:** HTML pages with templating logic
- **./static:** Static content such as images, css, javascript. 
- **./content:** data/content that is specially processed for use in pages and templates
  - **./content/blog:** Markdown blog posts
  - **./content/projects:** YAML files describing projects
- **./public:** The final generated site HTML

## TODOs
- If a file exists in the "public/" directory that no longer exists in the "static" directory, delete it
- Document template functions and available data

