package admin

import (
	cache "github.com/Focinfi/oncekv/cache/master"
	db "github.com/Focinfi/oncekv/db/master"
)

// Admin for oncekv admin
type Admin struct {
	CacheMaster *cache.Master
	DBMaster    *db.Master
}

// Default act as a default Admin
var Default = &Admin{
	CacheMaster: cache.Default,
	DBMaster:    db.Default,
}
