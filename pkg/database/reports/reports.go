package reports

import (
	"database/sql"
	"log"

	"github.com/otofuto/LiveInterpreting/pkg/database"
)

type Reports struct {
	Account        int `json:"account"`
	AccountName    string
	AccountEnabled bool    `json:"account_enabled"`
	ReportDate     string  `json:"report_date"`
	Sender         int     `json:"sender"`
	Reason         Reasons `json:"reason"`
}

type Reasons struct {
	Id     int    `json:"id"`
	Reason string `json:"reason"`
}

func (r *Reports) Insert() error {
	db := database.Connect()
	defer db.Close()

	sql := "insert into `report_accounts` (`account`, `sender`, `reason`) values (?, ?, ?)"
	ins, err := db.Prepare(sql)
	if err != nil {
		log.Println("reports.go (r *Reports) Insert() db.Prepare()")
		return err
	}
	defer ins.Close()
	_, err = ins.Exec(&r.Account, &r.Sender, &r.Reason.Id)
	if err != nil {
		log.Println("reports.go (r *Reports) Insert() ins.Exec()")
		return err
	}
	return nil
}

func (r *Reasons) Insert(db *sql.DB) error {
	sql := "insert into `report_reasons` (`id`, `reason`) values (?, ?)"
	ins, err := db.Prepare(sql)
	if err != nil {
		log.Println("reports.go (r *Reasons) Insert() db.Prepare()")
		return err
	}
	defer ins.Close()
	_, err = ins.Exec(&r.Id, &r.Reason)
	if err != nil {
		log.Println("reports.go (r *Reasons) Insert() ins.Exec()")
		return err
	}
	return nil
}

func CreateReasons(rs []Reasons) error {
	db := database.Connect()
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		log.Println("reports.go CreateReasons(rs []Reasons) db.Begin()")
		return err
	}
	del, err := db.Query("delete from `report_reasons`")
	if err != nil {
		log.Println("reports.go CreateReasons(rs []Reasons) db.Query()")
		tx.Rollback()
		return err
	}
	defer del.Close()
	for _, r := range rs {
		err := r.Insert(db)
		if err != nil {
			log.Println("reports.go CreateReasons(rs []Reasons) r.Insert()")
			tx.Rollback()
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		log.Println("reports.go CreateReasons(rs []Reasons) tx.Commit()")
		return err
	}
	return nil
}

func GetReasons() ([]Reasons, error) {
	db := database.Connect()
	defer db.Close()

	ret := make([]Reasons, 0)
	sql := "select `id`, `reason` from `report_reasons` order by `id`"
	rows, err := db.Query(sql)
	if err != nil {
		log.Println("reports.go GetReasons() db.Query()")
		return ret, err
	}
	defer rows.Close()
	for rows.Next() {
		var r Reasons
		err = rows.Scan(&r.Id, &r.Reason)
		if err != nil {
			log.Println("reports.go GetReasons() rows.Scan()")
			return ret, err
		}
		ret = append(ret, r)
	}
	return ret, nil
}

func All() ([]Reports, error) {
	db := database.Connect()
	defer db.Close()

	ret := make([]Reports, 0)
	sql := "select `account`, `accounts`.`enabled`, `report_date`, `sender`, `report_accounts`.`reason`, `report_reasons`.`reason`, `name` from `report_accounts` left outer join `report_reasons` on `report_accounts`.`reason` = `report_reasons`.`id` left outer join `accounts` on `account` = `accounts`.`id` order by `report_date` desc"
	rows, err := db.Query(sql)
	if err != nil {
		log.Println("reports.go All() db.Query()")
		return ret, err
	}
	defer rows.Close()
	for rows.Next() {
		var r Reports
		var rs Reasons
		err = rows.Scan(&r.Account, &r.AccountEnabled, &r.ReportDate, &r.Sender, &rs.Id, &rs.Reason, &r.AccountName)
		if err != nil {
			log.Println("reports.go All() rows.Scan()")
			return ret, err
		}
		r.Reason = rs
		ret = append(ret, r)
	}
	return ret, nil
}
