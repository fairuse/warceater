# warceater
WARCeater - a reader of WARC forum scrapes (archive.org) to create a searchable full-text index of them.

It was an experiment in which I wanted to see if reconstructing a forum from its archived state (WARC files as found on archive.org), was feasible.

The idea is first to parse the WARC files, use a set of html parsers and xpath/css selectors to extract 'forum post' objects, store these as JSON objects. 
The phase two indexes these JSON objects into a disk-based full-text search index (bleve search), so that we can easily find posts by content or by ID.
The last phase puts a simple web UI on top, that reformats the JSON objects into pages, so that threads can be shown in a sensible way.

It all worked, but it needs some real maintenance and refactoring if you want to use it for something that supports multiple formats.

This was a very early project to learn some Go basics, so the code quality will not be great.
