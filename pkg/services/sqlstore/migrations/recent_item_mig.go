package migrations

import (
	. "github.com/grafana/grafana/pkg/services/sqlstore/migrator"
)

func addRecentItemMigrations(mg *Migrator) {
	recentItemV1 := Table{
		Name: "recent_item",
		Columns: []*Column{
			{Name: "id", Type: DB_BigInt, Nullable: false, IsPrimaryKey: true, IsAutoIncrement: true},
			{Name: "uid", Type: DB_NVarchar, Length: 40, Nullable: false},
			{Name: "org_id", Type: DB_BigInt, Nullable: false},
			{Name: "user_id", Type: DB_BigInt, Nullable: false},
			{Name: "resource_type", Type: DB_NVarchar, Length: 40, Nullable: false},
			{Name: "resource_uid", Type: DB_NVarchar, Length: 255, Nullable: false},
			{Name: "title", Type: DB_NVarchar, Length: 255, Nullable: false},
			{Name: "url", Type: DB_NVarchar, Length: 2048, Nullable: false},
			{Name: "last_viewed_at", Type: DB_BigInt, Nullable: false},
		},
		Indices: []*Index{
			{Cols: []string{"uid"}, Type: UniqueIndex},
			{Cols: []string{"user_id", "org_id", "resource_type", "resource_uid"}, Type: UniqueIndex},
			{Cols: []string{"user_id", "org_id", "last_viewed_at"}},
		},
	}

	mg.AddMigration("create recent_item table v1", NewAddTableMigration(recentItemV1))
	mg.AddMigration("add unique index recent_item.uid", NewAddIndexMigration(recentItemV1, recentItemV1.Indices[0]))
	mg.AddMigration("add unique index recent_item.user_id-org_id-resource_type-resource_uid", NewAddIndexMigration(recentItemV1, recentItemV1.Indices[1]))
	mg.AddMigration("add index recent_item.user_id-org_id-last_viewed_at", NewAddIndexMigration(recentItemV1, recentItemV1.Indices[2]))
}
