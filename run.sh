#!/bin/bash

go install github.com/shurcooL/goexec@latest
go install go get github.com/shurcooL/go-goon@latest

cat<<EOF
navigate to http://localhost:8080/index.html, open the JavaScript debug console
EOF

goexec 'http.ListenAndServe(`:8080`, http.FileServer(http.Dir(`www`)))'
