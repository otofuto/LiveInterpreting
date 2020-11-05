package langs

import (
	//"fmt"
	"log"
	"github.com/otofuto/LiveInterpreting/pkg/database"
)

type Langs struct {
	Id int `json:"id"`
	Lang string `json:"lang"`
}

func All() []Langs {
	db := database.Connect()
	defer db.Close()

	rows, err := db.Query("select `id`, `lang` from `langs` order by `id`")
	if err != nil {
		log.Fatal(err)
	}

	var ls []Langs
	for rows.Next() {
		var l Langs
		rows.Scan(&l.Id, &l.Lang)
		ls = append(ls, l)
	}
	return ls
}