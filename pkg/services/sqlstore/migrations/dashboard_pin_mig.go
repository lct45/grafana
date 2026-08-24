package migrations

import (
	. "github.com/grafana/grafana/pkg/services/sqlstore/migrator"
)

func addDashboardPinMigrations(mg *Migrator) {
	dashboardPinV1 := Table{
		Name: "dashboard_pin",
		Columns: []*Column{
			{Name: "id", Type: DB_BigInt, Nullable: false, IsPrimaryKey: true, IsAutoIncrement: true},
			{Name: "org_id", Type: DB_BigInt, Nullable: false},
			{Name: "user_id", Type: DB_BigInt, Nullable: false},
			{Name: "dashboard_uid", Type: DB_NVarchar, Length: 40, Nullable: false},
			{Name: "note", Type: DB_NVarchar, Length: 256, Nullable: true},
			{Name: "sort_order", Type: DB_Int, Nullable: false},
			{Name: "created_at", Type: DB_BigInt, Nullable: false},
		},
		Indices: []*Index{
			{Cols: []string{"org_id", "user_id", "dashboard_uid"}, Type: UniqueIndex},
			{Cols: []string{"org_id", "user_id", "sort_order"}},
		},
	}

	mg.AddMigration("create dashboard_pin table v1", NewAddTableMigration(dashboardPinV1))
	mg.AddMigration("add unique index dashboard_pin.org_id-user_id-dashboard_uid", NewAddIndexMigration(dashboardPinV1, dashboardPinV1.Indices[0]))
	mg.AddMigration("add index dashboard_pin.org_id-user_id-sort_order", NewAddIndexMigration(dashboardPinV1, dashboardPinV1.Indices[1]))
}
