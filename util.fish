#!/usr/bin/env fish

function util_lint
  echo -n "=> linting util "
  golangci-lint run ./... || return 1
end

function util_test
  go clean -testcache
  go test -v github.com/volodymyrprokopyuk/go-util/... -run Check
end

function util_build
  echo "=> updating util"
  go get -u ./... && go mod tidy
  echo "=> building util"
  CGO_ENABLED=0 go build ./...
end

function html_generate -a title document
  echo "=> documenting $document.html"
  pandoc --metadata title="$title" --css doc/pandoc.css $document.org \
    --to html --embed-resources --standalone --output $document.html
end

function util_document
  set font "Noto Sans"
  set size 12
  for dot in doc/diagram/*.dot
    set svg (basename $dot .dot)
    set path (dirname $dot)
    dot -Nfontname="$font" -Efontname="$font" -Gfontname="$font" \
      -Nfontsize=$size -Efontsize=$size -Gfontzise=$size \
      -Tsvg $dot -o $path/$svg.svg
  end
  html_generate "Go util" doc/go-util
end

function main
  set cmd $argv[1]
  set sub $argv[2]
  switch $cmd
  case lint
    util_lint
  case test
    util_test
  case build
    util_build
  case document
    util_document
  case '*'
    echo "unknown command $cmd" && return 1
  end
end

main $argv
