package accounts

import (
	"log"
	//"fmt"
	"errors"
	"time"
	"strconv"
	"github.com/otofuto/LiveInterpreting/pkg/database"
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
	CreatedAt	string `json:"created_at"`
}

type AccountTokens struct {
	Id int `json:"id"`
	Token string `json:"token"`
	CreatedAt string `json:"created_at"`
}

func (ac *Accounts) Insert() int {
	if CheckMail(ac.Email, -1) == false {
		return -2
	}

	db := database.Connect()
	defer db.Close()

	ins, err := db.Prepare("insert into `accounts` (`name`, `email`, `password`, `icon_image`, `description`, `sex`, `user_type`) values (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Fatal(err)
		return -1
	}
	ac.Password = passHash(ac.Password)
	ins.Exec(&ac.Name, &ac.Email, &ac.Password, &ac.IconImage, &ac.Description, &ac.Sex, &ac.UserType)

	rows, err := db.Query("select last_insert_id()")
	if err != nil {
		log.Fatal(err)
	}
	if rows.Next() {
		newId := -1
		rows.Scan(&newId)
		ac.Id = newId
		return newId
	}
	return -1
}

func (ac *Accounts) Update() bool {
	if !CheckMail(ac.Email, ac.Id) {
		return false
	}

	db := database.Connect()
	defer db.Close()

	upd, err := db.Prepare("update `accounts` set `name` = ?, `email` = ?, `password` = ?, `icon_image` = ?, `description` = ?, `sex` = ?, `user_type` = ? where `id` = ?")
	if err != nil {
		log.Fatal(err)
		return false
	}
	upd.Exec(&ac.Name, &ac.Email, &ac.Password, &ac.IconImage, &ac.Description, &ac.Sex, &ac.UserType, &ac.Id)
	return true
}

func (ac *Accounts) Delete() bool {
	db := database.Connect()
	defer db.Close()

	upd, err := db.Prepare("delete from `accounts` where `id` = ?")
	if err != nil {
		log.Fatal(err)
		return false
	}
	upd.Exec(&ac.Id)
	return true
}

func (ac *Accounts) Get() bool {
	db := database.Connect()
	defer db.Close()

	rows, err := db.Query("select `name`, `email`, `password`, `icon_image`, `description`, `sex`, `user_type`, `created_at` from `accounts` where `id` = " + strconv.Itoa(ac.Id))
	if err != nil {
		log.Fatal(err)
	}
	if rows.Next() {
		err = rows.Scan(&ac.Name, &ac.Email, &ac.Password, &ac.IconImage, &ac.Description, &ac.Sex, &ac.UserType, &ac.CreatedAt)
		if err != nil {
			log.Fatal(err)
		}
		return true
	}
	return false
}

func CheckMail(mail string, id int) bool {
	db := database.Connect()
	defer db.Close()

	rows, err := db.Query("select `id` from `accounts` where `email` = '" + database.Escape(mail) + "' and `id` != " + strconv.Itoa(id))
	if err != nil {
		log.Fatal(err)
	}
	if rows.Next() {
		return false
	}
	return true
}

func Login(email string, pass string) (Accounts, error) {
	ac := Accounts { Id: -1 }
	db := database.Connect()
	defer db.Close()

	rows, err := db.Query("select `id`, `name`, `password`, `icon_image`, `description`, `sex`, `user_type`, `created_at` from `accounts` where `email` = '" + email + "'")
	if err != nil {
		log.Fatal(err)
	}
	if rows.Next() {
		err = rows.Scan(&ac.Id, &ac.Name, &ac.Password, &ac.IconImage, &ac.Description, &ac.Sex, &ac.UserType, &ac.CreatedAt)
		if err != nil {
			//log.Fatal(err)
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
		log.Fatal(err)
	}
	ins.Exec(&accounttoken.Id, &accounttoken.Token)

	return token
}

func passHash(pass string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), 10)
	if err != nil {
		log.Fatal(err)
	}
	return string(hash)
}

func checkPass(hash string, pass string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pass))
	return err == nil
}

func CheckToken(token string) (Accounts, error) {
	db := database.Connect()
	defer db.Close()

	db.Query("delete from `account_tokens` where `created_at` <= subtime(now(), '168:00:00') and `token` = '" + token + "'")
	//168hour = 24h * 7days
	rows, err := db.Query("select `id` from `account_tokens` where `created_at` > subtime(now(), '168:00:00') and `token` = '" + token + "'")
	if err != nil {
		log.Fatal(err)
		return Accounts{}, errors.New("select account_tokens failed")
	}
	if rows.Next() {
		var ac Accounts
		err = rows.Scan(&ac.Id)
		if err != nil {
			log.Fatal(err)
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

	db.Query("delete from `account_tokens` where `token` = '" + token + "'")
}