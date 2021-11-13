package lives

import (
	"database/sql"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/otofuto/LiveInterpreting/pkg/database"
	"github.com/otofuto/LiveInterpreting/pkg/database/trans"
)

type Lives struct {
	Id              int            `json:"id"`
	TransId         int            `json:"trans_id"`
	LiverId         int            `json:"liver_id"`
	LiverName       string         `json:"liver_name"`
	InterpreterId   int            `json:"interpreter_id"`
	InterpreterName string         `json:"interpreter_name"`
	Start           string         `json:"start"`
	End             string         `json:"end"`
	LangId          int            `json:"lang_id"`
	LangName        string         `json:"lang_name"`
	Url             string         `json:"url"`
	Title           string         `json:"title"`
	Image           sql.NullString `json:"image"`
}

func CreateLive(tr trans.Trans) (Lives, error) {
	if tr.Id <= 0 {
		return Lives{}, errors.New("trans data is empty")
	}
	db := database.Connect()
	defer db.Close()

	sql := "insert into `lives` (trans, liver, interpreter, start, end, lang, url, title, image) values (?, ?, ?, ?, ?, ?, '', '', null)"
	ins, err := db.Prepare(sql)
	if err != nil {
		log.Println("lives.go CreateLive(tr trans.Trans) db.Prepare()")
		return Lives{}, err
	}
	defer ins.Close()
	t, _ := time.Parse("2006-01-02 15:04:05", tr.LiveStart.String)
	t = t.Add(time.Duration(tr.LiveTime.Int64) * time.Minute)
	end := t.Format("2006-01-02 15:04:05")
	result, err := ins.Exec(&tr.Id, &tr.From, &tr.To, &tr.LiveStart, &end, &tr.Lang)
	if err != nil {
		log.Println("lives.go CreateLive(tr trans.Trans) ins.Exec()")
		return Lives{}, err
	}
	newid64, err := result.LastInsertId()
	if err != nil {
		log.Println("lives.go CreateLive(tr trans.Trans) result.LastInsertId()")
		return Lives{}, err
	}
	liv := Lives{
		Id:            database.Int64ToInt(newid64),
		TransId:       tr.Id,
		LiverId:       tr.From,
		InterpreterId: tr.To,
		Start:         tr.LiveStart.String,
		End:           end,
		LangId:        tr.Lang,
	}
	return liv, nil
}

func GetLives(db *sql.DB, count, offset int) ([]Lives, error) {
	ret := make([]Lives, 0)
	sql := "select lives.id, lives.trans, lives.liver, livers.`name`, lives.interpreter, interpreters.`name`, lives.start, lives.end, lives.lang, langs.lang, lives.url, lives.title, lives.image " +
		"from lives left outer join accounts as `livers` on livers.id = lives.liver left outer join accounts as `interpreters` on interpreters.id = lives.interpreter left outer join langs on langs.id = lives.lang " +
		"limit " + strconv.Itoa(count) + " offset " + strconv.Itoa(offset)
	rows, err := db.Query(sql)
	if err != nil {
		log.Println("lives.go GetLives(db *sql.DB, count, offset int)")
		return ret, err
	}
	defer rows.Close()
	for rows.Next() {
		var liv Lives
		err = rows.Scan(&liv.Id, &liv.TransId, &liv.LiverId, &liv.LiverName, &liv.InterpreterId, &liv.InterpreterName, &liv.Start, &liv.End, &liv.LangId, &liv.LangName, &liv.Url, &liv.Title, &liv.Image)
		if err != nil {
			log.Println("lives.go GetLives(db *sql.DB, count, offset int) rows.Scan()")
			return ret, err
		}
		ret = append(ret, liv)
	}
	return ret, nil
}

func GetFromTrans(db *sql.DB, trid int) (Lives, error) {
	sql := "select lives.id, lives.trans, lives.liver, livers.`name`, lives.interpreter, interpreters.`name`, lives.start, lives.end, lives.lang, langs.lang, lives.url, lives.title, lives.image " +
		"from lives left outer join accounts as `livers` on livers.id = lives.liver left outer join accounts as `interpreters` on interpreters.id = lives.interpreter left outer join langs on langs.id = lives.lang " +
		"where lives.trans = " + strconv.Itoa(trid)
	rows, err := db.Query(sql)
	if err != nil {
		log.Println("lives.go GetFromTrans(db *sql.DB, trid int) db.Query()")
		return Lives{}, err
	}
	defer rows.Close()
	var liv Lives
	if rows.Next() {
		err = rows.Scan(&liv.Id, &liv.TransId, &liv.LiverId, &liv.LiverName, &liv.InterpreterId, &liv.InterpreterName, &liv.Start, &liv.End, &liv.LangId, &liv.LangName, &liv.Url, &liv.Title, &liv.Image)
		if err != nil {
			log.Println("lives.go GetFromTrans(db *sql.DB, trid int) rows.Scan()")
			return liv, err
		}
	}
	return liv, nil
}

func Get(livid int) (Lives, error) {
	db := database.Connect()
	defer db.Close()

	sql := "select lives.id, lives.trans, lives.liver, livers.`name`, lives.interpreter, interpreters.`name`, lives.start, lives.end, lives.lang, langs.lang, lives.url, lives.title, lives.image " +
		"from lives left outer join accounts as `livers` on livers.id = lives.liver left outer join accounts as `interpreters` on interpreters.id = lives.interpreter left outer join langs on langs.id = lives.lang " +
		"where lives.id = " + strconv.Itoa(livid)
	rows, err := db.Query(sql)
	if err != nil {
		log.Println("lives.go Get(livid int) db.Query()")
		return Lives{}, err
	}
	defer rows.Close()
	var liv Lives
	if rows.Next() {
		err = rows.Scan(&liv.Id, &liv.TransId, &liv.LiverId, &liv.LiverName, &liv.InterpreterId, &liv.InterpreterName, &liv.Start, &liv.End, &liv.LangId, &liv.LangName, &liv.Url, &liv.Title, &liv.Image)
		if err != nil {
			log.Println("lives.go Get(livid int) rows.Scan()")
			return liv, err
		}
	}
	return liv, nil
}

func (liv *Lives) Update() error {
	db := database.Connect()
	defer db.Close()

	sql := "update `lives` set `title` = ?, `url` = ?, `start` = ?, `end` = ?, `lang` = ? where `id` = ?"
	upd, err := db.Prepare(sql)
	if err != nil {
		log.Println("lives.go (liv *Lives) Update() db.Prepare()")
		return err
	}
	defer upd.Close()
	_, err = upd.Exec(&liv.Title, &liv.Url, &liv.Start, &liv.End, &liv.LangId, &liv.Id)
	if err != nil {
		log.Println("lives.go (liv *Lives) Update() upd.Exec()")
		return err
	}
	return nil
}
