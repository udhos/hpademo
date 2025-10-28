#!/bin/bash

#go install github.com/shurcooL/goexec@latest
#go get github.com/shurcooL/go-goon@latest
go install github.com/sno6/gommand@latest

cat<<EOF

navigate to http://localhost:8080/index.html, open the JavaScript debug console

EOF

#goexec 'http.ListenAndServe(`:8080`, http.FileServer(http.Dir(`www`)))'
gommand 'http.Handle("/", http.FileServer(http.Dir("www"))); fmt.Println(http.ListenAndServe(":8080", nil))'
