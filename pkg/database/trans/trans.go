package trans

import (
	"database/sql"
	"errors"
	"log"
	"strconv"

	"github.com/otofuto/LiveInterpreting/pkg/database"
)

type Trans struct {
	Id                int            `json:"id"`
	From              int            `json:"from"`
	To                int            `json:"to"`
	LiveStart         sql.NullString `json:"live_start"`
	LiveTime          sql.NullInt64  `json:"live_time"`
	Lang              int            `json:"lang"`
	RequestType       int            `json:"request_type"`
	RequestTitle      string         `json:"request_title"`
	Request           string         `json:"request"`
	RequestDate       string         `json:"request_date"`
	BudgetRange       int            `json:"budget_range"`
	RequestCancel     int            `json:"request_cancel"`
	EstimateLimitDate sql.NullString `json:"estimate_limit_date"`
	Price             sql.NullInt64  `json:"price"`
	EstimateDate      sql.NullString `json:"estimate_date"`
	ResponseType      sql.NullInt64  `json:"response_type"`
	Response          sql.NullString `json:"response"`
	BuyDate           sql.NullString `json:"buy_date"`
	FinishedDate      sql.NullString `json:"finished_date"`
	CancelDate        sql.NullString `json:"cancel_date"`
	FromEval          sql.NullInt64  `json:"from_eval"`
	FromComment       sql.NullString `json:"from_comment"`
	ToEval            sql.NullInt64  `json:"to_eval"`
	ToComment         sql.NullString `json:"to_comment"`
}

func (tr *Trans) Insert() error {
	db := database.Connect()
	defer db.Close()

	ins, err := db.Prepare("insert into `trans` (`from`, `to`, `live_start`, `live_time`, `lang`, `request_type`, `request_title`, `request`, `budget_range`, `estimate_limit_date`, `price`) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Println(err)
		return errors.New("insert trans failed at trans.Insert")
	}
	defer ins.Close()
	_, err = ins.Exec(&tr.From, &tr.To, &tr.LiveStart, &tr.LiveTime, &tr.Lang,
		&tr.RequestType, &tr.RequestTitle, &tr.Request, &tr.BudgetRange,
		&tr.EstimateLimitDate, &tr.Price)
	if err != nil {
		log.Println(err)
		return errors.New("insert trans failed at trans.Insert")
	}

	rows, err := db.Query("select last_insert_id()")
	if err != nil {
		log.Println(err)
		return errors.New("select last_insert_id failed at trans.Insert")
	}
	defer rows.Close()
	if rows.Next() {
		rows.Scan(&tr.Id)
		if tr.Id == 0 {
			tr.Id = -1
			return errors.New("insert error: select last_insert_id() = 0")
		}

		return nil
	} else {
		return errors.New("cannot get last_insert_id at trans.Insert")
	}
}

func (tr *Trans) Update() error {
	db := database.Connect()
	defer db.Close()

	upd, err := db.Prepare(
		"update `trans` set " +
			"`from` = ?, `to` = ?, `live_start` = ?, `live_time` = ?, `lang` = ?, " +
			"`request_type` = ?, `request` = ?, `request_cancel` = ?, `price` = ?, " +
			"`request_title` = ?, `budget_range` = ?, `estimate_limit_date` = ?, " +
			"`estimate_date` = ?, `response_type` = ?, `response` = ?, `buy_date` = ?, " +
			"`finished_date` = ?, `cancel_date` = ?, `from_eval` = ?, `from_comment` = ?, " +
			"`to_eval` = ?, `to_comment` = ? where `id` = ?")
	if upd != nil {
		log.Println(err)
		return errors.New("failed to update trans at trans.Update")
	}
	defer upd.Close()
	_, err = upd.Exec(&tr.From, &tr.To, &tr.LiveStart, &tr.LiveTime, &tr.Lang,
		&tr.RequestType, &tr.Request, &tr.RequestCancel, &tr.Price,
		&tr.RequestTitle, &tr.BudgetRange, &tr.EstimateLimitDate,
		&tr.EstimateDate, &tr.ResponseType, &tr.Response, &tr.BuyDate,
		&tr.FinishedDate, &tr.CancelDate, &tr.FromEval, &tr.FromComment,
		&tr.ToEval, &tr.ToComment, &tr.Id)
	if err != nil {
		return err
	}
	return nil
}

func (tr *Trans) Get() bool {
	db := database.Connect()
	defer db.Close()

	sql := "select `from`, `to`, `live_start`, `live_time`, `lang`, `request_type`, " +
		"`request_title`, `request`, `request_date`, `budget_range`, `request_cancel`, `estimate_limit_date`, " +
		"`price`, `estimate_date`, `response_type`, `response`, `buy_date`, `finished_date`, " +
		"`cancel_date`, `from_eval`, `from_comment`, `to_eval`, `to_comment` from `trans` " +
		"where `id` = " + strconv.Itoa(tr.Id)
	rows, err := db.Query(sql)
	if err != nil {
		log.Println(err)
		return false
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&tr.From, &tr.To, &tr.LiveStart, &tr.LiveTime, &tr.Lang, &tr.RequestType,
			&tr.RequestTitle, &tr.Request, &tr.RequestDate, &tr.BudgetRange, &tr.RequestCancel, &tr.EstimateLimitDate,
			&tr.Price, &tr.EstimateDate, &tr.ResponseType, &tr.Response, &tr.BuyDate, &tr.FinishedDate,
			&tr.CancelDate, &tr.FromEval, &tr.FromComment, &tr.ToEval, &tr.ToComment)
		if err != nil {
			log.Println(err)
			return false
		}
		return true
	}
	return false
}
