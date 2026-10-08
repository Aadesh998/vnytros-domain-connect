package db

import "gorm.io/gorm"

// DB is the mailforge handle on the database. It is not opened here: the api
// owns the pool and assigns it in mailforge/server.Init, so both halves of the
// process share one set of connections.
var DB *gorm.DB
