# PageDownloader
This tool provides download list of web pages with using [sitemap](http://www.sitemaps.org/protocol.html), or it may download target page
## Usage
* `settings.cfg` and `url.cfg` uses JSON structure for initialize program
* To add link for sitemap or target page, add it as element of array to `urls.cfg` in section _Lists_ for sitemaps and _Pages_ for target links
## Settings
* `delay` - delay between processing last page and download next
* `multi_thread` - this setting in progress
