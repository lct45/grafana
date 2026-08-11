package migrations

import (
	. "github.com/grafana/grafana/pkg/services/sqlstore/migrator"
)

func addDashboardHomePinMigrations(mg *Migrator) {
	dashboardHomePinV1 := Table{
		Name: "dashboard_home_pin",
		Columns: []*Column{
			{Name: "id", Type: DB_BigInt, Nullable: false, IsPrimaryKey: true, IsAutoIncrement: true},
			{Name: "uid", Type: DB_NVarchar, Length: 40, Nullable: false},
			{Name: "org_id", Type: DB_BigInt, Nullable: false},
			{Name: "user_id", Type: DB_BigInt, Nullable: false},
			{Name: "dashboard_uid", Type: DB_NVarchar, Length: 40, Nullable: false},
			{Name: "sort_order", Type: DB_Int, Nullable: false, Default: "0"},
			{Name: "note", Type: DB_NVarchar, Length: 255, Nullable: true},
			{Name: "created_at", Type: DB_BigInt, Nullable: false},
			{Name: "updated_at", Type: DB_BigInt, Nullable: false},
		},
		Indices: []*Index{
			{Cols: []string{"uid"}, Type: UniqueIndex},
			{Cols: []string{"org_id", "user_id", "dashboard_uid"}, Type: UniqueIndex},
			{Cols: []string{"org_id", "user_id", "sort_order"}},
		},
	}

	mg.AddMigration("create dashboard_home_pin table v1", NewAddTableMigration(dashboardHomePinV1))
	mg.AddMigration("add unique index dashboard_home_pin.uid", NewAddIndexMigration(dashboardHomePinV1, dashboardHomePinV1.Indices[0]))
	mg.AddMigration("add unique index dashboard_home_pin.org_id-user_id-dashboard_uid", NewAddIndexMigration(dashboardHomePinV1, dashboardHomePinV1.Indices[1]))
	mg.AddMigration("add index dashboard_home_pin.org_id-user_id-sort_order", NewAddIndexMigration(dashboardHomePinV1, dashboardHomePinV1.Indices[2]))
}
