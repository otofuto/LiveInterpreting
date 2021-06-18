package accounts

import (
	"log"
	"fmt"
	"errors"
	"time"
	"strconv"
	"encoding/json"
	"github.com/otofuto/LiveInterpreting/pkg/database"
	"github.com/otofuto/LiveInterpreting/pkg/database/langs"
	"github.com/otofuto/LiveInterpreting/pkg/database/errorData"
	"golang.org/x/crypto/bcrypt"
)

type Accounts struct {
	Id int `json:"id"`
	Name string `json:"name"`
	Email string `json:"email"`
	Password string `json:"password"`
	IconImage string `json:"icon_image"`
	Description string `json:"description"`
	Sex int `json:"sex"`
	UserType string `json:"user_type"`
	Url1 string `json:"url1"`
	Url2 string `json:"url2"`
	Url3 string `json:"url3"`
	HourlyWage string `json:"hourly_wage"`
	Langs []langs.Langs `json:"langs"`
	CreatedAt string `json:"created_at"`
	Enabled int `json:"enabled"`
	LastLogined string `json:"last_logined"`
}

type AccountTokens struct {
	Id int `json:"id"`
	Token string `json:"token"`
	CreatedAt string `json:"created_at"`
}

type AccountSocial struct {
	Id int `json:"id"`
	TargetId int `json:"target_id"`
	Action int `json:"action"`
	CreatedAt string `json:"created_at"`
}

type Notif struct {
	Type string `json:"type"`
	Text string `json:"text"`
	Date string `json:"date"`
	From int `json:"from"`
}

func (ac *Accounts) Insert() int {
	if CheckMail(ac.Email, -1) == false {
		return -2
	}

	db := database.Connect()
	defer db.Close()

	ins, err := db.Prepare("insert into `accounts` (`name`, `email`, `password`, `icon_image`, `description`, `sex`, `user_type`, `url1`, `url2`, `url3`, `hourly_wage`) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Println(err)
		return -1
	}
	ac.Password = passHash(ac.Password)
	ins.Exec(&ac.Name, &ac.Email, &ac.Password, &ac.IconImage, &ac.Description, &ac.Sex, &ac.UserType, &ac.Url1, &ac.Url2, &ac.Url3, &ac.HourlyWage)
	ins.Close()

	rows, err := db.Query("select last_insert_id()")
	if err != nil {
		log.Println(err)
	}
	defer rows.Close()
	if rows.Next() {
		newId := -1
		rows.Scan(&newId)
		ac.Id = newId
		for _, v := range ac.Langs {
			ins, err = db.Prepare("insert into `account_langs` (`id`, `lang_id`) values (?, ?)")
			if err != nil {
				ac.Delete()
				log.Println(err)
				return -3
			}
			ins.Exec(newId, v.Id)
			ins.Close()
		}
		ac.Enabled = 1
		return newId
	}
	return -1
}

func (ac *Accounts) SetLangs(jsonstring string) error {
	var ids []string
	err := json.Unmarshal([]byte(jsonstring), &ids)
	if err != nil {
		return errors.New(err.Error())
	}
	var ls []langs.Langs
	for _, v := range ids {
		id, _ := strconv.Atoi(v)
		ls = append(ls, langs.Langs { Id: id })
	}
	ac.Langs = ls
	return nil
}

func (ac *Accounts) Update() bool {
	if !CheckMail(ac.Email, ac.Id) {
		return false
	}

	db := database.Connect()
	defer db.Close()

	upd, err := db.Prepare("update `accounts` set `name` = ?, `email` = ?, `icon_image` = ?, `description` = ?, `sex` = ?, `user_type` = ?, `url1` = ?, `url2` = ?, `url3` = ?, `hourly_wage` = ? where `id` = ?")
	if err != nil {
		log.Println(err)
		return false
	}
	defer upd.Close()
	upd.Exec(&ac.Name, &ac.Email, &ac.IconImage, &ac.Description, &ac.Sex, &ac.UserType, &ac.Url1, &ac.Url2, &ac.Url3, &ac.HourlyWage, &ac.Id)
	r, err := db.Query("delete from `account_langs` where `id` = " + strconv.Itoa(ac.Id))
	if err != nil {
		log.Println(err)
		return false
	}
	defer r.Close()
	for _, v := range ac.Langs {
		ins, err := db.Prepare("insert into `account_langs` (`id`, `lang_id`) values (?, ?)")
		if err != nil {
			ac.Delete()
			log.Println(err)
			return false
		}
		ins.Exec(ac.Id, v.Id)
		ins.Close()
	}
	return true
}

func (ac *Accounts) PassUpdate() bool {
	db := database.Connect()
	defer db.Close()

	db.Query("delete from `pass_reset` where `id` = " + strconv.Itoa(ac.Id))
	upd, err := db.Prepare("update `accounts` set `password` = ? where `id` = ?")
	if err != nil {
		log.Println(err)
		return false
	}
	defer upd.Close()
	upd.Exec(passHash(ac.Password), &ac.Id)
	return true
}

func (ac *Accounts) Delete() bool {
	db := database.Connect()
	defer db.Close()

	upd1, err := db.Prepare("delete from `accounts` where `id` = ?")
	if err != nil {
		log.Println(err)
		return false
	}
	upd1.Exec(&ac.Id)
	upd1.Close()
	upd2, err := db.Prepare("delete from `account_tokens` where `id` = ?")
	if err != nil {
		log.Println(err)
		return false
	}
	upd2.Exec(&ac.Id)
	upd2.Close()
	upd3, err := db.Prepare("delete from `account_langs` where `id` = ?")
	if err != nil {
		log.Println(err)
		return false
	}
	upd3.Exec(&ac.Id)
	upd3.Close()
	return true
}

func (ac *Accounts) Disabled() bool {
	db := database.Connect()
	defer db.Close()

	upd, err := db.Prepare("update `accounts` set `enabled` = 0 where `id` = ?")
	if err != nil {
		log.Println(err)
		return false
	}
	upd.Exec(&ac.Id)
	upd.Close()
	upd2, err := db.Prepare("delete from `account_tokens` where `id` = ?")
	if err != nil {
		log.Println(err)
		return false
	}
	upd2.Exec(&ac.Id)
	upd2.Close()
	ac.Enabled = 0
	return true
}

func (ac *Accounts) Get() bool {
	db := database.Connect()
	defer db.Close()

	rows, err := db.Query("select `name`, `email`, `password`, `icon_image`, `description`, `sex`, `user_type`, `url1`, `url2`, `url3`, `hourly_wage`, `created_at`, `enabled`, `last_logined` from `accounts` where `id` = " + strconv.Itoa(ac.Id))
	if err != nil {
		log.Println(err)
		return false
	}
	defer rows.Close()
	if rows.Next() {
		err = rows.Scan(&ac.Name, &ac.Email, &ac.Password, &ac.IconImage, &ac.Description, &ac.Sex, &ac.UserType, &ac.Url1, &ac.Url2, &ac.Url3, &ac.HourlyWage, &ac.CreatedAt, &ac.Enabled, &ac.LastLogined)
		if err != nil {
			log.Println(err)
			return false
		}
		rows2, err := db.Query("select langs.id, langs.lang from `account_langs` left outer join `langs` on `lang_id` = langs.id where `account_langs`.id = " + strconv.Itoa(ac.Id))
		if err != nil {
			log.Println(err)
			return false
		}
		defer rows2.Close()
		ac.Langs = make([]langs.Langs, 0)
		for rows2.Next() {
			var l langs.Langs
			err = rows2.Scan(&l.Id, &l.Lang)
			ac.Langs = append(ac.Langs, l)
		}
		return true
	}
	return false
}

func (ac *Accounts) GetView(loginid int) {
	if ac.Get() {
		ac.Password = ""
		ac.Email = ""
	} else {
		ac.Id = -1
	}
}

func CheckId(uid int) bool {
	db := database.Connect()
	defer db.Close()

	if uid < 0 {
		return false
	}

	sql := "select id from `accounts` where `enabled` = 1 and `id` = " + strconv.Itoa(uid)
	rows, err := db.Query(sql)
	if err != nil {
		log.Println(err)
		errorData.Insert("failed to select query in accounts.go at CheckId()", sql)
		return false
	}
	defer rows.Close()

	if rows.Next() {
		return true
	}
	return false
}

func (ac *Accounts) GetFromEmail() bool {
	db := database.Connect()
	defer db.Close()

	sql := "select `id`, `name`, `password`, `icon_image`, `description`, `sex`, `user_type`, `url1`, `url2`, `url3`, `hourly_wage`, `created_at`, `enabled`, `last_logined` from `accounts` where `enabled` = 1 and `email` = '" + database.Escape(ac.Email) + "'"
	rows, err := db.Query(sql)
	if err != nil {
		log.Println(err)
		errorData.Insert("failed to select query in accounts.go at GetFromEmail()", sql)
		return false
	}
	defer rows.Close()
	if rows.Next() {
		err = rows.Scan(&ac.Id, &ac.Name, &ac.Password, &ac.IconImage, &ac.Description, &ac.Sex, &ac.UserType, &ac.Url1, &ac.Url2, &ac.Url3, &ac.HourlyWage, &ac.CreatedAt, &ac.Enabled, &ac.LastLogined)
		if err != nil {
			log.Println(err)
		}
		sql = "select langs.id, langs.lang from `account_langs` left outer join `langs` on `lang_id` = langs.id where `account_langs`.id = " + strconv.Itoa(ac.Id)
		rows2, err := db.Query(sql)
		if err != nil {
			log.Println(err)
			errorData.Insert("failed to select query in accounts.go at GetFromEmail()", sql)
			return false
		}
		defer rows2.Close()
		ac.Langs = make([]langs.Langs, 0)
		for rows2.Next() {
			var l langs.Langs
			err = rows2.Scan(&l.Id, &l.Lang)
			ac.Langs = append(ac.Langs, l)
		}
		return true
	}
	return false
}

func CheckMail(mail string, id int) bool {
	db := database.Connect()
	defer db.Close()

	sql := "select `id` from `accounts` where `email` = '" + database.Escape(mail) + "' and `id` != " + strconv.Itoa(id)
	rows, err := db.Query(sql)
	if err != nil {
		log.Println(err)
		errorData.Insert("failed to select query in accounts.go at CheckMail()", sql)
		return false
	}
	defer rows.Close()
	if rows.Next() {
		return false
	}
	return true
}

func Login(email string, pass string) (Accounts, error) {
	ac := Accounts { Id: -1 }
	db := database.Connect()
	defer db.Close()

	rows, err := db.Query("select `id`, `name`, `password`, `icon_image`, `description`, `sex`, `user_type`, `url1`, `url2`, `url3`, `hourly_wage`, `created_at`, `enabled`, `last_logined` from `accounts` where `enabled` = 1 and `email` = '" + database.Escape(email) + "'")
	if err != nil {
		log.Println(err)
		return Accounts{}, errors.New("failed to select login")
	}
	defer rows.Close()
	if rows.Next() {
		err = rows.Scan(&ac.Id, &ac.Name, &ac.Password, &ac.IconImage, &ac.Description, &ac.Sex, &ac.UserType, &ac.Url1, &ac.Url2, &ac.Url3, &ac.HourlyWage, &ac.CreatedAt, &ac.Enabled, &ac.LastLogined)
		if err != nil {
			log.Println(err)
			return ac, errors.New("failed select in login")
		}
		if !checkPass(ac.Password, pass) {
			ac.Password = ""
			ac.Email = ""
			return ac, errors.New("unmatched password")
		}
		ac.Password = ""
		ac.Email = ""
	} else {
		return Accounts{}, errors.New("account not found")
	}

	return ac, nil
}

func (ac *Accounts) CreateToken() string {
	db := database.Connect();
	defer db.Close()

	token := passHash(ac.Email + time.Now().Format("yyyyMMddHHmmss"))
	accounttoken := AccountTokens {
		Id: ac.Id,
		Token: token,
	}
	ins, err := db.Prepare("insert into `account_tokens` (`id`, `token`) values (?, ?)")
	if err != nil {
		log.Println(err)
		return ""
	}
	defer ins.Close()
	ins.Exec(&accounttoken.Id, &accounttoken.Token)

	return token
}

func CheckToken(token string) (Accounts, error) {
	db := database.Connect()
	defer db.Close()

	del, err := db.Query("delete from `account_tokens` where `created_at` <= subtime(now(), '168:00:00') and `token` = '" + database.Escape(token) + "'")
	if err != nil {
		log.Println(err)
		return Accounts{}, errors.New("failed to delete old tokens")
	}
	defer del.Close()
	//168hour = 24h * 7days
	rows, err := db.Query("select `id` from `account_tokens` where `created_at` > subtime(now(), '168:00:00') and `token` = '" + database.Escape(token) + "'")
	if err != nil {
		log.Println(err)
		return Accounts{}, errors.New("select account_tokens failed")
	}
	defer rows.Close()
	if rows.Next() {
		var ac Accounts
		err = rows.Scan(&ac.Id)
		if err != nil {
			log.Println(err)
			return Accounts{}, errors.New("failed scan row in tokens")
		}
		if ac.Get() {
			ac.Password = ""
			ac.Email = ""
			return ac, nil
		} else {
			return Accounts{}, errors.New("Account not found")
		}
	} else {
		return Accounts{}, errors.New("There is not the token.")
	}
}

func DeleteToken(token string) {
	db := database.Connect()
	defer db.Close()

	del, _ := db.Query("delete from `account_tokens` where `token` = '" + database.Escape(token) + "'")
	del.Close()
}

func passHash(pass string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), 10)
	if err != nil {
		log.Println(err)
		return ""
	}
	return string(hash)
}

func PassResetToken(id int) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(time.Now().Format("yyyyMMddHHmmss")), 10)
	if err != nil {
		log.Println(err)
		return ""
	}
	token := string(hash)[8:28]
	db := database.Connect()
	defer db.Close()

	del, _ := db.Query("delete from `pass_reset` where `id` = " + strconv.Itoa(id))
	defer del.Close()
	ins, err := db.Prepare("insert into `pass_reset` (`id`, `token`) values (?, ?)")
	if err != nil {
		log.Println(err)
		return ""
	}
	defer ins.Close()
	ins.Exec(id, token)
	return token
}

func CheckPassResetToken(token string) Accounts {
	db := database.Connect()
	defer db.Close()

	//24時間以上経っている場合は削除する
	del, _ := db.Query("delete from `pass_reset` where `created_at` <= subtime(now(), '24:00:00')")
	defer del.Close()

	rows, err := db.Query("select `id` from `pass_reset` where `token` = '" + database.Escape(token) + "'")
	if err != nil {
		log.Println(err)
		return Accounts { Id: -1 }
	}
	defer rows.Close()

	if rows.Next() {
		var ac Accounts
		err = rows.Scan(&ac.Id)
		if err != nil {
			return Accounts { Id: -1 }
		}
		ac.Get()
		return ac
	}
	return Accounts { Id: -1 }
}

func checkPass(hash string, pass string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pass))
	return err == nil
}

func Search(search string, user_type string, id int) []Accounts {
	db := database.Connect()
	defer db.Close()

	if user_type != "" {
		user_type = "`user_type` = '" + user_type + "'"
	}

	if search == "new" {
		if user_type != "" {
			user_type += " and "
		}
		search = "`created_at` >= subtime(now(), '168:00:00')"
	} else if search == "follow" {
		if user_type != "" {
			user_type += " and "
		}
		search = "`id` in (select `target_id` from `account_social` where `id` = " + strconv.Itoa(id) + " order by `created_at` desc)"
	} else if search == "follower" {
		if user_type != "" {
			user_type += " and "
		}
		search = "`id` in (select `id` from `account_social` where `target_id` = " + strconv.Itoa(id) + " order by `created_at` desc)"
	} else {
		search = ""
	}

	rows, err := db.Query("select `id`, `name`, `icon_image`, `description`, `sex`, `user_type`, `url1`, `url2`, `url3`, `hourly_wage`, `created_at`, `enabled`, `last_logined` from `accounts` where `enabled` = 1 and " + user_type + search)
	if err != nil {
		log.Println(err)
		return make([]Accounts, 0)
	}
	defer rows.Close()
	acs := make([]Accounts, 0)
	for rows.Next() {
		var ac Accounts
		err = rows.Scan(&ac.Id, &ac.Name, &ac.IconImage, &ac.Description, &ac.Sex, &ac.UserType, &ac.Url1, &ac.Url2, &ac.Url3, &ac.HourlyWage, &ac.CreatedAt, &ac.Enabled, &ac.LastLogined)
		if err != nil {
			log.Println(err)
			return make([]Accounts, 0)
		}
		rows2, err := db.Query("select langs.id, langs.lang from `account_langs` left outer join `langs` on `lang_id` = langs.id where `account_langs`.id = " + strconv.Itoa(ac.Id))
		if err != nil {
			log.Println(err)
			return make([]Accounts, 0)
		}
		for rows2.Next() {
			var l langs.Langs
			err = rows2.Scan(&l.Id, &l.Lang)
			ac.Langs = append(ac.Langs, l)
		}
		rows2.Close()
		acs = append(acs, ac)
	}
	return acs
}

func Social(accountId int, action int) ([]AccountSocial, error) {
	db := database.Connect()
	defer db.Close()

	if action != 0 && action != 1 {
		return make([]AccountSocial, 0), errors.New("action number is not defined")
	}

	rows, err := db.Query("select `id`, `target_id`, `action`, `created_at` from `account_social` where `id` = " + strconv.Itoa(accountId) + " and `action` = " + strconv.Itoa(action) + " and `target_id` not in (select `id` from `accounts` where `enabled` = 0) order by `created_at` desc")
	if err != nil {
		return make([]AccountSocial, 0), errors.New("failed to select query")
	}
	defer rows.Close()
	socials := make([]AccountSocial, 0)
	for rows.Next() {
		var s AccountSocial
		_ = rows.Scan(&s.Id, &s.TargetId, &s.Action, &s.CreatedAt)
		socials = append(socials, s)
	}
	return socials, nil
}

func (s *AccountSocial) Insert() bool {
	db := database.Connect()
	defer db.Close()

	if s.Action != 0 && s.Action != 1 {
		return false
	}

	ins, err := db.Prepare("insert into `account_social` (`id`, `target_id`, `action`) values (?, ?, ?)")
	if err != nil {
		fmt.Println(err)
		return false
	}
	defer ins.Close()
	ins.Exec(&s.Id, &s.TargetId, &s.Action)
	return true
}

func (s *AccountSocial) Delete() bool {
	db := database.Connect()
	defer db.Close()

	del, err := db.Prepare("delete from `account_social` where `id` = ? and `target_id` = ?")
	if err != nil {
		fmt.Println(err)
		return false
	}
	defer del.Close()
	del.Exec(&s.Id, &s.TargetId)
	return true
}

func CheckFollow(id int, targetId int) bool {
	db := database.Connect()
	defer db.Close()

	rows, err := db.Query("select * from `account_social` where `action` = 0 and `id` = " + strconv.Itoa(id) + " and `target_id` = " + strconv.Itoa(targetId))
	if err != nil {
		fmt.Println(err)
		return false
	}
	defer rows.Close()
	if rows.Next() {
		return true
	}
	return false
}

func (ac *Accounts) UpdateLastLogin() {
	db := database.Connect()
	defer db.Close()

	upd, err := db.Prepare("update `accounts` set `last_logined` = now() where `id` = ?")
	if err != nil {
		log.Println(err)
		return
	}
	defer upd.Close()
	upd.Exec(&ac.Id)
}

func (ac *Accounts) GetNotifs() ([]Notif, error) {
	db := database.Connect()
	defer db.Close()

	var notifs []Notif
	rows, err := db.Query("select `message`, `created_at`, `from` from `direct_messages` where `read` = 0 and `to` = " + strconv.Itoa(ac.Id) + " order by `created_at` desc")
	if err != nil {
		return notifs, errors.New("failed to get DM")
	}
	defer rows.Close()
	for rows.Next() {
		n := Notif { Type: "dm" }
		err = rows.Scan(&n.Text, &n.Date, &n.From)
		notifs = append(notifs, n)
	}
	return notifs, nil
}