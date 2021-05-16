package errorData

import (
	"log"
	"github.com/otofuto/LiveInterpreting/pkg/database"
)

func Insert(title string, message string) {
	db := database.Connect()
	defer db.Close()

	ins, err := db.Prepare("insert into `error_datas` (`title`, `message`) values (?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer ins.Close()
	ins.Exec(title, message)
}