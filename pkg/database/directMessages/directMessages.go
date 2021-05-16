package directMessages

import (
	"log"
	"errors"
	"strconv"
	"github.com/otofuto/LiveInterpreting/pkg/database"
	"github.com/otofuto/LiveInterpreting/pkg/database/errorData"
)

type DirectMessages struct {
	From int `json:"from"`
	To int `json:"to"`
	Id int `json:"id"`
	Message string `json:"message"`
	CreatedAt string `json:"created_at"`
	Read bool `json:"read"`
}

func (dm *DirectMessages) Insert() int {
	db := database.Connect()
	defer db.Close()

	newId := 0
	rows, err := db.Query("select count(*) as cnt from `direct_messages` where `from` = " + strconv.Itoa(dm.From) + " and `to` = " + strconv.Itoa(dm.To))
	if err != nil {
		log.Println(err)
		return -1
	}
	defer rows.Close()
	if rows.Next() {
		err = rows.Scan(&newId)
		if err != nil {
			log.Println(err)
			return -1
		}
	} else {
		errorData.Insert("rows.Next() equals False on DM Insert", strconv.Itoa(dm.From) + " -> " + strconv.Itoa(dm.To))
		return -1
	}

	ins, err := db.Prepare("insert into `direct_messages` (`from`, `to`, `id`, `message`) values (?, ?, ?, ?)")
	if err != nil {
		log.Println(err)
		return -1
	}
	defer ins.Close()
	ins.Exec(&dm.From, &dm.To, newId, &dm.Message)

	rows2, err := db.Query("select `created_at` from `direct_messages` where `from` = " + strconv.Itoa(dm.From) + " and `to` = " + strconv.Itoa(dm.To) + " and `id` = " + strconv.Itoa(newId))
	if err != nil {
		log.Println(err)
		return -1
	}
	defer rows2.Close()
	if rows2.Next() {
		rows2.Scan(&dm.CreatedAt)
	}

	return newId
}

func (dm *DirectMessages) Get() bool {
	db := database.Connect()
	defer db.Close()

	rows, err := db.Query("select `from`, `to`, `id`, `message`, `created_at`, `read` from `direct_messages` where `from` = " + strconv.Itoa(dm.From) + " and `to` = " + strconv.Itoa(dm.To) + " and `id` = " + strconv.Itoa(dm.Id))
	if err != nil {
		log.Println(err)
		return false
	}
	defer rows.Close()
	if rows.Next() {
		err = rows.Scan(&dm.From, &dm.To, &dm.Id, &dm.Message, &dm.CreatedAt, &dm.Read)
		if err != nil {
			log.Println(err)
			return false
		}
		return true
	}
	return false
}

func List(me int, to int) ([]DirectMessages, error) {
	db := database.Connect()
	defer db.Close()

	rows, err := db.Query("select `from`, `to`, `id`, `message`, `created_at`, `read` from `direct_messages` where " +
		"(`from` = " + strconv.Itoa(me) + " and `to` = " + strconv.Itoa(to) + ") or " +
		"(`from` = " + strconv.Itoa(to) + " and `to` = " + strconv.Itoa(me) + ") order by `created_at`")
	if err != nil {
		log.Println(err)
		return make([]DirectMessages, 0), errors.New("select failed at directMessages.List")
	}
	defer rows.Close()
	var ret []DirectMessages
	for rows.Next() {
		var dm DirectMessages
		err = rows.Scan(&dm.From, &dm.To, &dm.Id, &dm.Message, &dm.CreatedAt, &dm.Read)
		if err != nil {
			log.Println(err)
			return make([]DirectMessages, 0), errors.New("rows scan failed at directMessages.List")
		}
		ret = append(ret, dm)
	}
	return ret, nil
}

func (dm *DirectMessages) SetRead() bool {
	db := database.Connect()
	defer db.Close()

	upd, err := db.Prepare("update `direct_messages` set `read` = 1 where `from` = ? and `to` = ? and `id` = ?")
	if err != nil {
		log.Println(err)
		return false
	}
	defer upd.Close()
	upd.Exec(&dm.From, &dm.To, &dm.Id)
	return true
}