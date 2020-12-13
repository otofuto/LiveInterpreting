package trans

import (
	"log"
	"github.com/otofuto/LiveInterpreting/pkg/database"
)

type Trans struct {
	Id int `json:"id"`
	From int `json:"from"`
	To int `json:"to"`
	LiveStart string `json:"live_start"`
	LiveTime int `json:"live_time"`
	Lang int `json:"lang"`
	RequestType int `json:"request_type"`
	Request string `json:"request"`
	RequestCancel int `json:"request_cancel"`
	Price int `json:"price"`
	EstimateDate string `json:"estimate_date"`
	ResponseType int `json:"response_type"`
	Response string `json:"response"`
	BuyDate string `json:"buy_date"`
	FinishedDate string `json:"finished_date"`
	CancelDate string `json:"cancel_date"`
	FromEval int `json:"from_eval"`
	FromComment string `json:"from_comment"`
	ToEval int `json:"to_eval"`
	ToComment string `json:"to_comment"`
}

func (tr *Trans) Insert() (int, error) {
	db := database.Connect()
	defer db.Close()

	ins, err := db.Prepare("insert into `trans` (`from`, `to`, `live_start`, `live_time`, `lang`, `request_type`, `request`, `price`, `estimate_date`) values (?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Println(err)
		return -1, errors.New("insert trans failed at trans.Insert")
	}
	defer ins.Close()
	ins.Exec(&tr.From, &tr.To, &tr.LiveStart, &tr.LiveTime, &tr.Lang, &tr.RequestType, &tr.Request, &tr.Price, &tr.EstimateDate)

	rows, err := db.Query("select last_insert_id()")
	if err != nil {
		log.Println(err)
		return -1, errors.New("select last_insert_id failed at trans.Insert")
	}
	defer rows.Close()
	if rows.Next() {
		var newId int
		rows.Scan(&newId)

		return newId, nil
	} else {
		return -1, errors.New("cannot get last_insert_id at trans.Insert")
	}
}

func (tr *Trans) Update() error {
	db := database.Connect()
	defer db.Close()

	upd, err := db.Prepare(
		"update `trans` set " +
		"`from` = ?, `to` = ?, `live_start` = ?, `live_time` = ?, `lang` = ?, " +
		"`request_type` = ?, `request` = ?, `request_cancel` = ?, `price` = ?, " +
		"`estimate_date` = ?, `response_type` = ?, `response` = ?, `buy_date` = ?, " +
		"`finished_date` = ?, `cancel_date` = ?, `from_eval` = ?, `from_comment` = ?, " +
		"`to_eval` = ?, `to_comment` = ? where `id` = ?")
	if upd != nil {
		log.Println(err)
		return errors.New("failed to update trans at trans.Update")
	}
	defer upd.Close()
	upd.Exec(&tr.From, &tr.To, &tr.LiveStart)
}