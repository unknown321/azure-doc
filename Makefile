REPODIR=architecture-center
REPOURL=https://github.com/MicrosoftDocs/architecture-center.git
COMMIT=f3151c568c5895b5b37145672194ec421e2a1c33
BINARY=azure-doc

architecture-center/docs/networking/guide/images/ipv4-exhaustion-load-balancer-l4.png:
	find "${REPODIR}/docs" -type f -name "*.svg" -exec bash -c ' \
		echo "$${1}" ; \
		rsvg-convert "$${1}" -o "$${1%.*}.png" \
		' _ {} \;

svg2png: architecture-center/docs/networking/guide/images/ipv4-exhaustion-load-balancer-l4.png

architecture-center:
	mkdir -p "$(REPODIR)"
	git init "$(REPODIR)"
	cd "$(REPODIR)" && \
		git remote add origin $(REPOURL) && \
		git fetch --depth 1 origin $(COMMIT) && \
		git checkout $(COMMIT)

Azure_Architecture_Center.epub: architecture-center svg2png
	./azure-doc

$(BINARY):
	go build .

build: $(BINARY)

clean:
	@-rm $(BINARY) *.epub $(BINARY)-linux-amd64

test:
	@go test -v .

release: clean build Azure_Architecture_Center.epub
	strip $(BINARY)
	mv $(BINARY) $(BINARY)-linux-amd64

.DEFAULT_GOAL=Azure_Architecture_Center.epub