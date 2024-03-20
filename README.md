azure-doc
=========

Creates EPUB from [Azure Architecture Center](https://github.com/MicrosoftDocs/architecture-center)

See:
  - https://github.com/MicrosoftDocs/architecture-center/issues/1569
  - https://github.com/MicrosoftDocs/architecture-center/issues/2048

#### Optional dependencies:
  - rsvg-convert (`librsvg2-bin` in Debian/Ubuntu), converts svg to png - most ebook devices don't support svg 

#### Building

```shell
make build
```

#### Usage

```shell
Usage of ./azure-doc:
  -out string
    	output file (default "Azure_Architecture_Center.epub")
  -path docs
    	path to docs dir (default "./architecture-center/docs")
```

#### Example

```shell
# downloads documentation repo into ./architecture-center
# converts svg to png with rsvg-convert for ebook readers without svg support
# assembles the epub
make
```

#### TODO

  - [ ] correct cross-document links
  - [ ] fix absolute yml/md paths