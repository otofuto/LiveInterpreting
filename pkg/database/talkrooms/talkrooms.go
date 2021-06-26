package talkrooms

import (
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/otofuto/LiveInterpreting/pkg/database"
)

type TalkRooms struct {
	TransId   int    `json:"trans_id"`
	Id        int    `json:"id"`
	From      int    `json:"from"`
	To        int    `json:"to"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
	Read      bool   `json:"read"`
}

func (tr *TalkRooms) Insert() error {
	if tr.TransId == 0 {
		log.Println("talkrooms.go (tr *TalkRooms) Insert()")
		return errors.New("tr.trans_id is not set")
	}
	if tr.From == 0 {
		log.Println("talkrooms.go (tr *TalkRooms) Insert()")
		return errors.New("tr.from is not set")
	}
	if tr.To == 0 {
		log.Println("talkrooms.go (tr *TalkRooms) Insert()")
		return errors.New("tr.to is not set")
	}
	if strings.TrimSpace(tr.Message) == "" {
		log.Println("talkrooms.go (tr *TalkRooms) Insert()")
		return errors.New("tr.message is empty")
	}

	db := database.Connect()
	defer db.Close()

	sql := "insert into `talkrooms` (`trans_id`, `id`, `from`, `to`, `message`) select ?, (select count(*) from `talkrooms` where `trans_id` = " + strconv.Itoa(tr.TransId) + "), ?, ?, ?"
	ins, err := db.Prepare(sql)
	if err != nil {
		log.Println("talkrooms.go (tr *TalkRooms) Insert()")
		log.Println(err)
		log.Println(sql)
		return err
	}
	defer ins.Close()
	msg := strings.TrimSpace(tr.Message)
	_, err = ins.Exec(&tr.TransId, &tr.From, &tr.To, &msg)
	if err != nil {
		log.Println("talkrooms.go (tr *TalkRooms) Insert()")
		log.Println(err)
		log.Println(sql)
		return err
	}

	sql = "select `id`, `created_at` from `talkrooms` where `id` = (select max(`id`) from `talkrooms` where `trans_id` = " + strconv.Itoa(tr.TransId) + ") and `trans_id` = " + strconv.Itoa(tr.TransId)
	rows, err := db.Query(sql)
	if err != nil {
		log.Println("talkrooms.go (tr *TalkRooms) Insert()")
		log.Println(err)
		return err
	}
	defer rows.Close()
	if rows.Next() {
		err = rows.Scan(&tr.Id, &tr.CreatedAt)
		if err != nil {
			log.Println("talkrooms.go (tr *TalkRooms) Insert()")
			log.Println(err)
			return err
		}
		return nil
	}
	return errors.New("failed to fetch new id")
}

func (tr *TalkRooms) SetRead() error {
	if tr.TransId == 0 {
		log.Println("talkrooms.go (tr *TalkRooms) Insert()")
		return errors.New("tr.trans_id is not set")
	}

	db := database.Connect()
	defer db.Close()

	sql := "update `talkrooms` set `read` = 1 where `trans_id` = " + strconv.Itoa(tr.TransId) + " and `id` = " + strconv.Itoa(tr.Id)
	upd, err := db.Prepare(sql)
	if err != nil {
		log.Println("talkrooms.go (tr *TalkRooms) SetRead()")
		log.Println(err)
		log.Println(sql)
		return err
	}
	defer upd.Close()
	_, err = upd.Exec()
	if err != nil {
		log.Println("talkrooms.go (tr *TalkRooms) SetRead()")
		log.Println(err)
		log.Println(sql)
		return err
	}
	return nil
}

func List(trid int) []TalkRooms {
	db := database.Connect()
	defer db.Close()

	ret := make([]TalkRooms, 0)

	sql := "select `trans_id`, `id`, `from`, `to`, `message`, `created_at` from `talkrooms` where `trans_id` = " + strconv.Itoa(trid) + " order by `id`"
	rows, err := db.Query(sql)
	if err != nil {
		log.Println("talkrooms.go List(trid int)")
		log.Println(err)
		log.Println(sql)
		return ret
	}
	defer rows.Close()
	for rows.Next() {
		var talk TalkRooms
		err = rows.Scan(&talk.TransId, &talk.Id, &talk.From, &talk.To, &talk.Message, &talk.CreatedAt)
		if err != nil {
			log.Println("talkrooms.go List(trid int)")
			log.Println(err)
		} else {
			ret = append(ret, talk)
		}
	}
	return ret
}
