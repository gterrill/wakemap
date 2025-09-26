module github.com/gterrill/wakemap/core

go 1.22

// replace ensures the module is resolved to the local folder during dev
replace github.com/gterrill/wakemap/core => .

require github.com/mattn/go-sqlite3 v1.14.32
