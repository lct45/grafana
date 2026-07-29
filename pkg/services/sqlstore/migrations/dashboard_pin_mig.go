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
			{Name: "sort_order", Type: DB_Int, Nullable: false, Default: "0"},
			{Name: "note", Type: DB_NVarchar, Length: 500, Nullable: true},
			{Name: "created", Type: DB_DateTime, Nullable: false},
			{Name: "updated", Type: DB_DateTime, Nullable: false},
		},
		Indices: []*Index{
			{Cols: []string{"user_id", "org_id", "dashboard_uid"}, Type: UniqueIndex},
			{Cols: []string{"user_id", "org_id", "sort_order"}, Type: IndexType},
		},
	}

	mg.AddMigration("create dashboard_pin table v1", NewAddTableMigration(dashboardPinV1))
	mg.AddMigration("add unique index dashboard_pin.user_id_org_id_dashboard_uid", NewAddIndexMigration(dashboardPinV1, dashboardPinV1.Indices[0]))
	mg.AddMigration("add index dashboard_pin.user_id_org_id_sort_order", NewAddIndexMigration(dashboardPinV1, dashboardPinV1.Indices[1]))
}
