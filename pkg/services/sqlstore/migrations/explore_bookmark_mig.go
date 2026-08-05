package migrations

import (
	. "github.com/grafana/grafana/pkg/services/sqlstore/migrator"
)

func addExploreBookmarkMigrations(mg *Migrator) {
	exploreBookmarkV1 := Table{
		Name: "explore_bookmark",
		Columns: []*Column{
			{Name: "id", Type: DB_BigInt, Nullable: false, IsPrimaryKey: true, IsAutoIncrement: true},
			{Name: "uid", Type: DB_NVarchar, Length: 40, Nullable: false},
			{Name: "org_id", Type: DB_BigInt, Nullable: false},
			{Name: "user_id", Type: DB_BigInt, Nullable: false},
			{Name: "name", Type: DB_NVarchar, Length: 255, Nullable: false},
			{Name: "datasource_uid", Type: DB_NVarchar, Length: 40, Nullable: false},
			{Name: "queries", Type: DB_Text, Nullable: false},
			{Name: "time_from", Type: DB_NVarchar, Length: 255, Nullable: false},
			{Name: "time_to", Type: DB_NVarchar, Length: 255, Nullable: false},
			{Name: "created_at", Type: DB_BigInt, Nullable: false},
		},
		Indices: []*Index{
			{Cols: []string{"uid"}, Type: UniqueIndex},
			{Cols: []string{"org_id", "user_id", "created_at"}},
		},
	}

	mg.AddMigration("create explore_bookmark table v1", NewAddTableMigration(exploreBookmarkV1))
	mg.AddMigration("add unique index explore_bookmark.uid", NewAddIndexMigration(exploreBookmarkV1, exploreBookmarkV1.Indices[0]))
	mg.AddMigration("add index explore_bookmark.org_id-user_id-created_at", NewAddIndexMigration(exploreBookmarkV1, exploreBookmarkV1.Indices[1]))
}
