package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/otofuto/LiveInterpreting/pkg/account"
	"github.com/otofuto/LiveInterpreting/pkg/database"
	"github.com/otofuto/LiveInterpreting/pkg/database/accounts"
	"github.com/otofuto/LiveInterpreting/pkg/database/directMessages"
	"github.com/otofuto/LiveInterpreting/pkg/database/errorData"
	"github.com/otofuto/LiveInterpreting/pkg/database/langs"
	"github.com/otofuto/LiveInterpreting/pkg/database/talkrooms"
	"github.com/otofuto/LiveInterpreting/pkg/database/trans"
	stripe "github.com/otofuto/LiveInterpreting/pkg/stripeHandler"
)

var port string

//WebSocket関係
var clients = make(map[*websocket.Conn]string)
var broadcast = make(chan SocketMessage)
var upgrader = websocket.Upgrader{}

type TempContext struct {
	Login    accounts.Accounts               `json:"login"`
	User     accounts.Accounts               `json:"user"`
	Users    []accounts.Accounts             `json:"users"`
	Message  string                          `json:"message"`
	Messages []directMessages.DirectMessages `json:"direct_messages"`
	Trans    trans.Trans                     `json:"trans"`
	Transes  []trans.Trans                   `json:"transes"`
	Talks    []talkrooms.TalkRooms           `json:"talkrooms"`
}

type SocketMessage struct {
	Message   string `json:"message"`
	From      int    `json:"from"`
	Id        int    `json:"id"`
	CreatedAt string `json:"created_at"`
	ChatId    string `json:"chat_id"`
	TransId   int    `json:"trans_id"`
}

func main() {
	port = os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	http.Handle("/st/", http.StripPrefix("/st/", http.FileServer(http.Dir("./static"))))
	http.HandleFunc("/", IndexHandle)
	http.HandleFunc("/favicon.ico", FaviconHandle)

	http.HandleFunc("/Account/", account.AccountHandle)
	http.HandleFunc("/AccountSocial/", account.AccountSocialHandle)
	http.HandleFunc("/Login/", account.LoginHandle)
	http.HandleFunc("/Logout/", account.LogoutHandle)
	http.HandleFunc("/PassForgot/", account.PassForgotHandle)

	http.HandleFunc("/Notifications/", NotificationsHandle)

	http.HandleFunc("/u/", UserHandle)
	http.HandleFunc("/mypage/", MypageHandle)
	http.HandleFunc("/home/", HomeHandle)
	http.HandleFunc("/Lang/", LangHandle)
	http.HandleFunc("/directmessages/", DMHandle)
	http.HandleFunc("/search/", SearchHandle)
	http.HandleFunc("/trans/", TransHandle)
	http.HandleFunc("/inbox/", InboxHandle)
	http.HandleFunc("/payment/", PaymentHandle)
	http.HandleFunc("/live/", LiveHandle)

	http.HandleFunc("/document/", documentHandle)

	http.HandleFunc("/ws/", SocketHandle)
	go handleMessages()

	log.Println("Listening on port: " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func IndexHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodGet {
		cookie, err := r.Cookie("accounttoken")
		if err != nil {
			temp := template.Must(template.ParseFiles("template/index.html"))
			if err := temp.Execute(w, ""); err != nil {
				log.Println(err)
				http.Error(w, "error", 500)
			}
			return
		}
		_, err = accounts.CheckToken(cookie.Value)
		if err != nil {
			temp := template.Must(template.ParseFiles("template/index.html"))
			if err := temp.Execute(w, ""); err != nil {
				log.Println(err)
				http.Error(w, "error", 500)
				return
			}
		} else {
			http.Redirect(w, r, "/home/", 302)
		}
	} else {
		http.Error(w, "method not allowed", 405)
	}
}

func FaviconHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/ico")
	file, err := os.Open("./static/materials/favicon.ico")
	if err != nil {
		http.Error(w, "failed to open the favicon", 500)
		return
	}
	defer file.Close()
	io.Copy(w, file)
}

func NotificationsHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodGet {
		login := account.LoginAccount(r)
		if login.Id == -1 {
			http.Error(w, "not logined", 403)
			return
		}
		notifs, err := login.GetNotifs()
		if err != nil {
			http.Error(w, "failed to fetch notifications", 500)
			log.Println(err)
			return
		}
		bytes, err := json.Marshal(notifs)
		if err != nil {
			http.Error(w, "failed to marshal objects to json", 500)
			return
		}
		fmt.Fprintf(w, string(bytes))
	} else if r.Method == http.MethodDelete {
		login := account.LoginAccount(r)
		if login.Id == -1 {
			http.Error(w, "not logined", 403)
			return
		}
		r.ParseMultipartForm(32 << 20)
		if isset(r, []string{
			"from",
			"to",
			"type",
			"date",
		}) {
			from, err := strconv.Atoi(r.FormValue("from"))
			if err != nil {
				http.Error(w, "from is not integer", 400)
				return
			}
			to, err := strconv.Atoi(r.FormValue("to"))
			if err != nil {
				http.Error(w, "to is not integer", 400)
				return
			}
			err = accounts.DeleteNotif(from, to, r.FormValue("type"), r.FormValue("date"))
			if err != nil {
				log.Println(err)
				http.Error(w, "failed to delete notif", 500)
				return
			}
			fmt.Fprintf(w, "true")
		} else {
			http.Error(w, "parametors not enough", 400)
		}
	} else {
		http.Error(w, "method not alloed.", 405)
	}
}

func UserHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodGet {
		accountId, err := strconv.Atoi(r.URL.Path[len("/u/"):])
		if err != nil {
			http.Error(w, "URLが正しくありません。", 400)
			return
		}
		ac := accounts.Accounts{
			Id: accountId,
		}
		if ac.Get() {
			if ac.Enabled == 0 {
				temp := template.Must(template.ParseFiles("template/disableduser.html"))

				if err := temp.Execute(w, ""); err != nil {
					log.Println(err)
					http.Error(w, "error", 500)
				}
				return
			}
			ac.Password = ""
			context := struct {
				Account    accounts.Accounts `json:"account"`
				Login      accounts.Accounts `json:"login"`
				IsFollow   bool              `json:"is_follow"`
				IsFollower bool              `json:"is_follower"`
			}{
				Account:    ac,
				Login:      accounts.Accounts{Id: -1},
				IsFollow:   false,
				IsFollower: false,
			}
			loginaccount := account.LoginAccount(r)
			if loginaccount.Id != -1 {
				context.Login = loginaccount
				context.IsFollow = accounts.CheckFollow(loginaccount.Id, ac.Id)
				context.IsFollower = accounts.CheckFollow(ac.Id, loginaccount.Id)
				loginaccount.UpdateLastLogin()
			}
			temp := template.Must(template.ParseFiles("template/user.html"))

			if err := temp.Execute(w, context); err != nil {
				log.Println(err)
				http.Error(w, "error", 500)
				return
			}
		} else {
			http.Error(w, "このユーザーは存在しません。", 404)
		}
	} else {
		http.Error(w, "method not allowed.", 405)
	}
}

func MypageHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodGet {
		login := account.LoginAccount(r)
		if login.Id == -1 {
			http.Redirect(w, r, "/", 303)
			return
		}
		msg := ""
		msgs := make([]directMessages.DirectMessages, 0)
		trs := make([]trans.Trans, 0)
		filename := r.URL.Path[len("/mypage/"):]
		if filename == "" {
			filename = "index"
		}
		if filename[len(filename)-1:] == "/" {
			filename = filename[:len(filename)-1]
		}
		users := make([]accounts.Accounts, 0)
		if filename == "index" {
			login.GetView(login.Id)
			msgs = login.GetDMs(10)
			trs = login.GetTranses(10, 0)
			db := database.Connect()
			defer db.Close()
			for _, dm := range msgs {
				u := accounts.Accounts{Id: dm.From}
				if dm.From == login.Id {
					u.Id = dm.To
				}
				u.GetLite(db)
				users = append(users, u)
			}
			for _, tr := range trs {
				u := accounts.Accounts{Id: tr.From}
				if tr.From == login.Id {
					u.Id = tr.To
				}
				u.GetLite(db)
				users = append(users, u)
			}
			users = append(users, login)
			bytes, err := json.Marshal(users)
			if err != nil {
				log.Println(err)
				http.Error(w, "failed to convert object to json", 500)
				return
			}
			msg = string(bytes)
		} else if filename == "profile" {
			if !login.Get() {
				http.Error(w, "failed to fetch account info", 500)
				return
			}
		}
		temp := template.Must(template.ParseFiles("template/mypage/" + filename + ".html"))
		if err := temp.Execute(w, TempContext{
			Login:    login,
			Message:  msg,
			Messages: msgs,
			Transes:  trs,
			Users:    users,
		}); err != nil {
			log.Println(err)
			http.Error(w, "HTTP 500 Internal server error", 500)
		}
		login.UpdateLastLogin()
	} else {
		http.Error(w, "method not allowed", 405)
	}
}

func HomeHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodGet {
		ac := account.LoginAccount(r)
		temp := template.Must(template.ParseFiles("template/home.html"))
		if err := temp.Execute(w, TempContext{
			Login: ac,
		}); err != nil {
			log.Println(err)
			http.Error(w, "error", 500)
			return
		}
		ac.UpdateLastLogin()
	} else {
		http.Error(w, "method not allowed", 405)
	}
}

func LangHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodGet {
		bytes, err := json.Marshal(langs.All())
		if err != nil {
			log.Println(err)
			http.Error(w, "error", 500)
			return
		}
		fmt.Fprintf(w, string(bytes))
	} else {
		http.Error(w, "method not allowed.", 405)
	}
}

func DMHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodGet {
		login := account.LoginAccount(r)
		if login.Id == -1 {
			http.Redirect(w, r, "/", 303)
			return
		}
		accountId, err := strconv.Atoi(r.URL.Path[len("/directmessages/"):])
		if err != nil {
			http.Error(w, "<h1>このページはまだ作成されていません。</h1>", 404)
		} else {
			ac := accounts.Accounts{
				Id: accountId,
			}
			if ac.Get() {
				ac.Password = ""
				ac.Email = ""
				dms, err := directMessages.List(login.Id, ac.Id)
				if err != nil {
					http.Error(w, "failed to list your direct messages.", 500)
					return
				}
				context := struct {
					Account accounts.Accounts               `json:"account"`
					Login   accounts.Accounts               `json:"login"`
					DM      []directMessages.DirectMessages `json:"dm"`
				}{
					Account: ac,
					Login:   login,
					DM:      dms,
				}
				temp := template.Must(template.ParseFiles("template/dm.html"))
				if err := temp.Execute(w, context); err != nil {
					log.Println(err)
					http.Error(w, "error", 500)
					return
				}
				login.UpdateLastLogin()
				for _, dm := range dms {
					if !dm.Read {
						dm.SetRead()
					}
				}
			} else {
				http.Error(w, "このユーザーは存在しません。", 404)
			}
		}
	} else if r.Method == http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		r.ParseMultipartForm(32 << 20)
		if r.FormValue("message") == "" {
			http.Error(w, "parameter 'message' is not allowed empty.", 400)
			return
		}
		login := account.LoginAccount(r)
		if login.Id == -1 {
			http.Error(w, "not logined", 403)
			return
		}
		accountId, err := strconv.Atoi(r.URL.Path[len("/directmessages/"):])
		if err != nil {
			http.Error(w, "accountId was not designationed.", 400)
			return
		}
		ac := accounts.Accounts{
			Id: accountId,
		}
		if ac.Get() {
			dm := directMessages.DirectMessages{
				From:    login.Id,
				To:      ac.Id,
				Message: r.FormValue("message"),
			}
			newId := dm.Insert()
			if newId == -1 {
				http.Error(w, "insert failed", 500)
				return
			}
			chatId := "dm"
			if ac.Id < login.Id {
				chatId += strconv.Itoa(ac.Id) + "_" + strconv.Itoa(login.Id)
			} else {
				chatId += strconv.Itoa(login.Id) + "_" + strconv.Itoa(ac.Id)
			}
			msg := SocketMessage{
				Message:   dm.Message,
				From:      dm.From,
				Id:        newId,
				CreatedAt: dm.CreatedAt,
				ChatId:    chatId,
			}
			for client, id := range clients {
				if id == chatId {
					client.WriteJSON(msg)
				}
			}
			fmt.Fprintf(w, strconv.Itoa(newId))
		} else {
			http.Error(w, "user not found.", 404)
		}
	} else if r.Method == http.MethodPut {
		after := r.URL.Path[len("/directmessages/"):]
		if strings.Index(after, "/") > 0 {
			from, err := strconv.Atoi(after[:strings.Index(after, "/")])
			if err != nil {
				http.Error(w, "not integer", 400)
				return
			}
			after = after[strings.Index(after, "/")+1:]
			if strings.Index(after, "/") > 0 {
				to, err := strconv.Atoi(after[:strings.Index(after, "/")])
				if err != nil {
					http.Error(w, "not integer", 400)
					return
				}
				after = after[strings.Index(after, "/")+1:]
				id, err := strconv.Atoi(after)
				if err != nil {
					http.Error(w, "not integer", 400)
					return
				}
				dm := directMessages.DirectMessages{
					From: from,
					To:   to,
					Id:   id,
				}
				ac := account.LoginAccount(r)
				if ac.Id == to {
					if dm.SetRead() {
						fmt.Fprintf(w, "true")
					} else {
						http.Error(w, "failed to update dm to read", 500)
					}
				} else {
					http.Error(w, "your account is not allowed.", 403)
				}
			} else {
				http.Error(w, "path: '/directmessages/from/to/chat_id'", 400)
			}
		} else {
			http.Error(w, "path: '/directmessages/from/to/chat_id'", 400)
		}
	} else {
		http.Error(w, "method not allowed", 405)
	}
}

func SearchHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	if r.Method == http.MethodGet {
		temp := template.Must(template.ParseFiles("template/search.html"))
		if err := temp.Execute(w, TempContext{
			Login: account.LoginAccount(r),
		}); err != nil {
			log.Fatal(err)
		}
	} else {
		http.Error(w, "method not allowed", 405)
	}
}

func TransHandle(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html")
		login := account.LoginAccount(r)
		if login.Id == -1 {
			http.Redirect(w, r, "/", 303)
			return
		}
		msg := ""
		msgs := make([]directMessages.DirectMessages, 0)
		talks := make([]talkrooms.TalkRooms, 0)
		filename := r.URL.Path[len("/trans/"):]
		if strings.HasSuffix(filename, "/") {
			filename = filename[:len(filename)-1]
		}
		ac := accounts.Accounts{Id: -1}
		if strings.HasPrefix(filename, "req/") {
			uid, err := strconv.Atoi(filename[len("req/"):])
			if err != nil {
				http.Error(w, "user id was not set", 400)
				return
			}
			ac.Id = uid
			if ac.Get() {
				filename = "req"
			} else {
				http.Redirect(w, r, "/home/", 303)
				return
			}
		} else if strings.HasPrefix(filename, "reqedit/") {
			trid, err := strconv.Atoi(filename[len("reqedit/"):])
			if err != nil {
				http.Error(w, "page not found", 404)
				return
			}
			tr := trans.Trans{Id: trid}
			if !tr.Get() {
				http.Error(w, "trans not found", 404)
				return
			}
			if tr.From != login.Id {
				http.Redirect(w, r, "/home/", 303)
				return
			}
			if tr.RequestCancel != 0 {
				http.Error(w, "既にキャンセルされています。", 400)
				return
			}
			ac.Id = tr.To
			if !ac.Get() {
				http.Error(w, "failed to get account data", 500)
				return
			}
			bytes, err := json.Marshal(tr)
			if err != nil {
				http.Error(w, "failed to convert trans object to json", 500)
				return
			}
			msg = string(bytes)
			filename = "reqedit"
		} else if strings.HasPrefix(filename, "estimate/") {
			trid, err := strconv.Atoi(filename[len("estimate/"):])
			if err != nil {
				http.Error(w, "page not found", 404)
				return
			}
			tr := trans.Trans{Id: trid}
			if !tr.Get() {
				http.Error(w, "trans not found", 404)
				return
			}
			if tr.To != login.Id {
				http.Redirect(w, r, "/home/", 303)
				return
			}
			ac.Id = tr.From
			if !ac.Get() {
				http.Error(w, "failed to get user(from) data", 500)
				return
			}
			msgobj := struct {
				Trans trans.Trans       `json:"trans"`
				From  accounts.Accounts `json:"from"`
				To    accounts.Accounts `json:"to"`
				Langs []langs.Langs     `json:"langs"`
			}{
				Trans: tr,
				From:  login,
				To: accounts.Accounts{
					Id:   ac.Id,
					Name: ac.Name,
				},
				Langs: langs.All(),
			}
			bytes, err := json.Marshal(msgobj)
			if err != nil {
				http.Error(w, "failed to convert object to json", 500)
				return
			}
			msg = string(bytes)
			filename = "estimate"
		} else if strings.HasPrefix(filename, "estedit/") {
			trid, err := strconv.Atoi(filename[len("estedit/"):])
			if err != nil {
				http.Error(w, "page not found", 404)
				return
			}
			tr := trans.Trans{Id: trid}
			if !tr.Get() {
				http.Error(w, "trans not found", 404)
				return
			}
			if tr.To != login.Id {
				http.Redirect(w, r, "/home/", 303)
				return
			}
			if !tr.ResponseType.Valid {
				http.Error(w, "まだ見積を送っていません。", 400)
				return
			}
			if tr.ResponseType.Int64 == 1 {
				http.Error(w, "既にキャンセルされています。", 400)
				return
			}
			if tr.BuyDate.Valid {
				http.Error(w, "購入されたあとでは変更できません。", 400)
				return
			}
			ac.Id = tr.From
			if !ac.Get() {
				http.Error(w, "failed to get user(from) data", 500)
				return
			}
			msgobj := struct {
				Trans trans.Trans       `json:"trans"`
				From  accounts.Accounts `json:"from"`
				To    accounts.Accounts `json:"to"`
				Langs []langs.Langs     `json:"langs"`
			}{
				Trans: tr,
				From:  login,
				To: accounts.Accounts{
					Id:   ac.Id,
					Name: ac.Name,
				},
				Langs: langs.All(),
			}
			bytes, err := json.Marshal(msgobj)
			msg = string(bytes)
			filename = "estedit"
		} else if strings.HasPrefix(filename, "buy/") {
			trid, err := strconv.Atoi(filename[len("buy/"):])
			if err != nil {
				http.Error(w, "page not found", 404)
				return
			}
			tr := trans.Trans{Id: trid}
			if !tr.Get() {
				http.Error(w, "trans not found", 404)
				return
			}
			if tr.From != login.Id {
				http.Redirect(w, r, "/home/", 303)
				return
			}
			ac.Id = tr.To
			if !ac.Get() {
				http.Error(w, "failed to get user(from) data", 500)
				return
			}
			msgobj := struct {
				Trans trans.Trans       `json:"trans"`
				From  accounts.Accounts `json:"from"`
				To    accounts.Accounts `json:"to"`
				Langs []langs.Langs     `json:"langs"`
			}{
				Trans: tr,
				To:    login,
				From: accounts.Accounts{
					Id:   ac.Id,
					Name: ac.Name,
				},
				Langs: langs.All(),
			}
			bytes, err := json.Marshal(msgobj)
			if err != nil {
				http.Error(w, "failed to convert trans object to json", 500)
				return
			}
			msg = string(bytes)
			filename = "buy"
		} else if strings.HasPrefix(filename, "talkroom/") {
			trid, err := strconv.Atoi(filename[len("talkroom/"):])
			if err != nil {
				http.Error(w, "page not found", 404)
				return
			}
			tr := trans.Trans{Id: trid}
			if !tr.Get() {
				http.Error(w, "trans not found", 404)
				return
			}
			if tr.To != login.Id && tr.From != login.Id {
				http.Redirect(w, r, "/home/", 303)
				return
			}
			if !tr.BuyDate.Valid {
				http.Redirect(w, r, "/trans/"+strconv.Itoa(tr.Id), 303)
				return
			}
			if login.Id == tr.From {
				ac.Id = tr.To
			} else {
				ac.Id = tr.From
			}
			if !ac.Get() {
				http.Error(w, "failed to get user(from) data", 500)
				return
			}
			talks = talkrooms.List(tr.Id)
			filename = "talkroom"
		} else {
			trid, err := strconv.Atoi(filename)
			if err != nil {
				http.Error(w, "page not found", 404)
				return
			}
			tr := trans.Trans{Id: trid}
			if !tr.Get() {
				http.Error(w, "trans not found", 404)
				return
			}
			if tr.From != login.Id && tr.To != login.Id {
				http.Redirect(w, r, "/home/", 303)
				return
			}
			ac.Id = tr.To
			if !ac.Get() {
				http.Error(w, "failed to get user(to) data", 500)
				return
			}
			from := accounts.Accounts{Id: tr.From}
			db := database.Connect()
			defer db.Close()
			if !from.GetLite(db) {
				http.Error(w, "failed to get user(from) data", 500)
				return
			}
			msgobj := struct {
				Trans trans.Trans       `json:"trans"`
				From  accounts.Accounts `json:"from"`
				To    accounts.Accounts `json:"to"`
				Langs []langs.Langs     `json:"langs"`
			}{
				Trans: tr,
				From:  from,
				To: accounts.Accounts{
					Id:   ac.Id,
					Name: ac.Name,
				},
				Langs: langs.All(),
			}
			bytes, err := json.Marshal(msgobj)
			if err != nil {
				http.Error(w, "failed to convert trans object to json", 500)
				return
			}
			msg = string(bytes)
			filename = "index"
		}
		temp := template.Must(template.ParseFiles("template/trans/" + filename + ".html"))
		if err := temp.Execute(w, TempContext{
			Login:    login,
			User:     ac,
			Message:  msg,
			Messages: msgs,
			Talks:    talks,
		}); err != nil {
			log.Println(err)
			http.Error(w, "HTTP 500 Internal server error", 500)
		}
	} else if r.Method == http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		login := account.LoginAccount(r)
		if login.Id == -1 {
			http.Error(w, "not logined", 403)
			return
		}

		mode := r.URL.Path[len("/trans/"):]
		if mode == "" {
			http.Error(w, "なんだてめぇ", 404)
			return
		}
		if strings.HasPrefix(mode, "req/") {
			if login.StripeCustomer == "" {
				http.Error(w, "login user is not registered credit card yet", 403)
				return
			}
			uid, err := strconv.Atoi(mode[len("req/"):])
			if err != nil {
				http.Error(w, "user id is not integer", 400)
				return
			}
			if !accounts.CheckId(uid) {
				http.Error(w, "user not found", 404)
				return
			}
			r.ParseMultipartForm(32 << 20)
			if isset(r, []string{
				"live_start",
				"live_time",
				"lang",
				"request_type",
				"request_title",
				"request",
				"budget_range",
				"estimate_limit_date",
			}) {
				livetimestr := r.FormValue("live_time")
				if strings.Index(r.FormValue("live_time"), ":") >= 0 {
					hour, err := strconv.Atoi(r.FormValue("live_time")[:strings.Index(r.FormValue("live_time"), ":")])
					if err != nil {
						http.Error(w, "live_time is invalid format", 400)
						return
					}
					min, err := strconv.Atoi(r.FormValue("live_time")[strings.Index(r.FormValue("live_time"), ":")+1:])
					if err != nil {
						http.Error(w, "live_time is invalid format", 400)
						return
					}
					livetimestr = strconv.Itoa(hour*60 + min)
				}
				livetime, err := strconv.Atoi(livetimestr)
				if err != nil {
					http.Error(w, "live_time is not integer", 400)
					return
				}
				lang, err := strconv.Atoi(r.FormValue("lang"))
				if err != nil {
					http.Error(w, "lang is not integer", 400)
					return
				}
				reqtype, err := strconv.Atoi(r.FormValue("request_type"))
				if err != nil {
					http.Error(w, "request_type is not integer", 400)
					return
				}
				budgetrange, err := strconv.Atoi(r.FormValue("budget_range"))
				if err != nil {
					http.Error(w, "budget_range is not integer", 400)
					return
				}
				tra := trans.Trans{
					From:              login.Id,
					To:                uid,
					LiveStart:         sql.NullString{Valid: true, String: r.FormValue("live_start")},
					LiveTime:          sql.NullInt64{Valid: true, Int64: int64(livetime)},
					Lang:              lang,
					RequestType:       reqtype,
					RequestTitle:      r.FormValue("request_title"),
					Request:           r.FormValue("request"),
					BudgetRange:       budgetrange,
					EstimateLimitDate: sql.NullString{Valid: true, String: r.FormValue("estimate_limit_date")},
				}
				err = tra.Insert()
				if err != nil {
					log.Println(err)
					http.Error(w, "failed to regist", 500)
					return
				}
				n := accounts.Notif{
					Type: "trans/req",
					Text: tra.RequestTitle,
					From: login.Id,
					To:   uid,
					Id:   tra.Id,
				}
				err = n.Insert()
				if err != nil {
					log.Println(err)
					errorData.Insert("failed to insert notif "+strconv.Itoa(login.Id)+" to "+strconv.Itoa(uid), err.Error())
				}
				bytes, err := json.Marshal(tra)
				if err != nil {
					fmt.Fprintf(w, "true")
					return
				}
				fmt.Fprintf(w, string(bytes))
			} else {
				http.Error(w, "parameters not enough", 400)
			}
		} else if strings.HasPrefix(mode, "estimate/") {
			trid, err := strconv.Atoi(mode[len("estimate/"):])
			if err != nil {
				http.Error(w, "trans id is not set", 404)
				return
			}
			r.ParseMultipartForm(32 << 20)
			if isset(r, []string{
				"price",
				"response",
			}) {
				tr := trans.Trans{Id: trid}
				if !tr.Get() {
					http.Error(w, "failed to get trans data", 404)
					return
				}
				if tr.To != login.Id {
					http.Error(w, "dame", 403)
					return
				}
				price, err := strconv.Atoi(r.FormValue("price"))
				if err != nil {
					http.Error(w, "price is not integer", 400)
					return
				}
				tr.Price = sql.NullInt64{Int64: int64(price), Valid: true}
				tr.Response = sql.NullString{String: r.FormValue("response"), Valid: true}
				tr.ResponseType = sql.NullInt64{Int64: 0, Valid: true}
				tr.EstimateDate = sql.NullString{String: time.Now().Format("2006-01-02 15:04:05"), Valid: true}
				err = tr.Update()
				if err != nil {
					log.Println(err)
					http.Error(w, "failed to update trans", 500)
					return
				}
				respHead := strings.Replace(tr.Response.String, "\n", " ", -1)
				if len([]rune(respHead)) > 16 {
					respHead = string([]rune(respHead)[:16]) + "…"
				}
				n := accounts.Notif{
					Type: "trans/res",
					Text: respHead,
					From: tr.To,
					To:   tr.From,
					Id:   tr.Id,
				}
				err = n.Insert()
				if err != nil {
					log.Println(err)
					errorData.Insert("failed to insert notif "+strconv.Itoa(login.Id)+" to "+strconv.Itoa(tr.From), err.Error())
				}
				bytes, err := json.Marshal(tr)
				if err != nil {
					http.Error(w, "failed to convert trans object to json", 500)
					return
				}
				fmt.Fprintf(w, string(bytes))
			} else {
				http.Error(w, "parameter not enough", 400)
			}
		} else if strings.HasPrefix(mode, "buy/") {
			trid, err := strconv.Atoi(mode[len("buy/"):])
			if err != nil {
				http.Error(w, "trans id is not set", 404)
				return
			}
			tr := trans.Trans{Id: trid}
			if !tr.Get() {
				http.Error(w, "failed to get trans data", 404)
				return
			}
			if tr.From != login.Id {
				http.Error(w, "dame", 403)
				return
			}
			if tr.BuyDate.Valid {
				http.Error(w, "already buyed", 400)
				return
			}
			err = stripe.Payment(login.StripeCustomer, tr.Price.Int64)
			if err != nil {
				log.Println("main.go TransHandle(w http.ResponseWriter, r *http.Request) POST")
				log.Println(err)
				http.Error(w, "failed to payment", 500)
				return
			}
			tr.BuyDate = sql.NullString{String: time.Now().Format("2006-01-02 15:04:05"), Valid: true}
			err = tr.Update()
			if err != nil {
				log.Println(err)
				http.Error(w, "failed to update trans", 500)
				return
			}
			n := accounts.Notif{
				Type: "trans/buy",
				Text: tr.RequestTitle,
				From: tr.From,
				To:   tr.To,
				Id:   tr.Id,
			}
			err = n.Insert()
			if err != nil {
				log.Println(err)
				errorData.Insert("failed to insert notif "+strconv.Itoa(login.Id)+" to "+strconv.Itoa(tr.From), err.Error())
			}
			fmt.Fprintf(w, "true")
		} else if strings.HasPrefix(mode, "talkroom/") {
			trid, err := strconv.Atoi(mode[len("talkroom/"):])
			if err != nil {
				http.Error(w, "trans id is not set", 404)
				return
			}
			tr := trans.Trans{Id: trid}
			if !tr.Get() {
				http.Error(w, "failed to get trans data", 404)
				return
			}
			if tr.To != login.Id && tr.From != login.Id {
				http.Error(w, "誰だてめぇは", 403)
				return
			}
			if !tr.BuyDate.Valid {
				http.Error(w, "this trans is not buyed yet", 400)
				return
			}
			r.ParseMultipartForm(32 << 20)
			if strings.TrimSpace(r.FormValue("message")) == "" {
				http.Error(w, "message is empty", 400)
				return
			}
			talk := talkrooms.TalkRooms{
				TransId: tr.Id,
				From:    login.Id,
				To:      tr.To,
				Message: r.FormValue("message"),
			}
			if tr.To == login.Id {
				talk.To = tr.From
			}
			err = talk.Insert()
			if err != nil {
				log.Println("main.go TransHandle(w http.ResponseWriter, r *http.Request)")
				log.Println(err)
				http.Error(w, "failed to insert message", 500)
				return
			}
			chatId := "talk"
			if tr.From > tr.To {
				chatId += strconv.Itoa(tr.To) + "_" + strconv.Itoa(tr.From)
			} else {
				chatId += strconv.Itoa(tr.From) + "_" + strconv.Itoa(tr.To)
			}
			msg := SocketMessage{
				Message:   talk.Message,
				From:      talk.From,
				Id:        talk.Id,
				CreatedAt: talk.CreatedAt,
				ChatId:    chatId,
				TransId:   talk.TransId,
			}
			for client, id := range clients {
				if id == chatId {
					client.WriteJSON(msg)
				}
			}
			fmt.Fprintf(w, "true")
		} else {
			http.Error(w, "user not designation", 404)
		}
	} else if r.Method == http.MethodPut {
		w.Header().Set("Content-Type", "application/json")
		mode := r.URL.Path[len("/trans/"):]
		if strings.HasPrefix(mode, "talkrooms/") {
			str := r.URL.Path[len("/trans/talkrooms/"):]
			if strings.Index(str, "/") > 0 {
				trid, err := strconv.Atoi(str[:strings.Index(str, "/")])
				if err != nil {
					http.Error(w, "trans_id is not integer", 400)
					return
				}
				id, err := strconv.Atoi(str[strings.Index(str, "/")+1:])
				if err != nil {
					http.Error(w, "id is not integer", 400)
					return
				}
				talk := talkrooms.TalkRooms{
					TransId: trid,
					Id:      id,
				}
				err = talk.SetRead()
				if err != nil {
					log.Println("main.go TransHandle(w http.ResponseWriter, r *http.Request)")
					log.Println(err)
					http.Error(w, "talk's read flag change failed", 500)
					return
				}
			} else {
				http.Error(w, "path is ends with trans_id/id", 404)
			}
		} else if strings.HasPrefix(mode, "req/") {
			trid, err := strconv.Atoi(mode[len("req/"):])
			if err != nil {
				http.Error(w, "trans id is not set", 404)
				return
			}
			r.ParseMultipartForm(32 << 20)
			if isset(r, []string{
				"live_start",
				"live_time",
				"lang",
				"request_type",
				"request_title",
				"request",
				"budget_range",
				"estimate_limit_date",
			}) {
				livetimestr := r.FormValue("live_time")
				if strings.Index(r.FormValue("live_time"), ":") >= 0 {
					hour, err := strconv.Atoi(r.FormValue("live_time")[:strings.Index(r.FormValue("live_time"), ":")])
					if err != nil {
						http.Error(w, "live_time is invalid format", 400)
						return
					}
					min, err := strconv.Atoi(r.FormValue("live_time")[strings.Index(r.FormValue("live_time"), ":")+1:])
					if err != nil {
						http.Error(w, "live_time is invalid format", 400)
						return
					}
					livetimestr = strconv.Itoa(hour*60 + min)
				}
				livetime, err := strconv.Atoi(livetimestr)
				if err != nil {
					http.Error(w, "live_time is not integer", 400)
					return
				}
				lang, err := strconv.Atoi(r.FormValue("lang"))
				if err != nil {
					http.Error(w, "lang is not integer", 400)
					return
				}
				reqtype, err := strconv.Atoi(r.FormValue("request_type"))
				if err != nil {
					http.Error(w, "request_type is not integer", 400)
					return
				}
				budgetrange, err := strconv.Atoi(r.FormValue("budget_range"))
				if err != nil {
					http.Error(w, "budget_range is not integer", 400)
					return
				}
				login := account.LoginAccount(r)
				if login.Id == -1 {
					http.Error(w, "not logined", 403)
					return
				}
				tr := trans.Trans{Id: trid}
				if !tr.Get() {
					http.Error(w, "failed to get trans object", 500)
					return
				}
				if tr.EstimateDate.Valid {
					http.Error(w, "got estimate", 409)
					return
				}
				if tr.From != login.Id {
					http.Error(w, "who are you majide", 403)
					return
				}
				tr.LiveStart = sql.NullString{Valid: true, String: r.FormValue("live_start")}
				tr.LiveTime = sql.NullInt64{Valid: true, Int64: int64(livetime)}
				tr.Lang = lang
				tr.RequestType = reqtype
				tr.RequestTitle = r.FormValue("request_title")
				tr.Request = r.FormValue("request")
				tr.BudgetRange = budgetrange
				tr.EstimateLimitDate = sql.NullString{Valid: true, String: r.FormValue("estimate_limit_date")}
				err = tr.Update()
				if err != nil {
					log.Println(err)
					http.Error(w, "failed to regist", 500)
					return
				}
				n := accounts.Notif{
					Type: "trans/reqedit",
					Text: tr.RequestTitle,
					From: tr.From,
					To:   tr.To,
					Id:   tr.Id,
				}
				err = n.Insert()
				if err != nil {
					log.Println(err)
					errorData.Insert("failed to insert notif "+strconv.Itoa(tr.From)+" to "+strconv.Itoa(tr.To), err.Error())
				}
				bytes, err := json.Marshal(tr)
				if err != nil {
					fmt.Fprintf(w, "true")
					return
				}
				fmt.Fprintf(w, string(bytes))
			} else {
				http.Error(w, "parameters not enough", 400)
			}
		} else if strings.HasPrefix(mode, "estimate/") {
			trid, err := strconv.Atoi(mode[len("estimate/"):])
			if err != nil {
				http.Error(w, "trans id is not set", 404)
				return
			}
			r.ParseMultipartForm(32 << 20)
			if isset(r, []string{
				"price",
				"response",
			}) {
				tr := trans.Trans{Id: trid}
				if !tr.Get() {
					http.Error(w, "failed to get trans data", 404)
					return
				}
				if tr.BuyDate.Valid {
					http.Error(w, "this trans is already buyed", 409)
					return
				}
				login := account.LoginAccount(r)
				if tr.To != login.Id {
					http.Error(w, "dame", 403)
					return
				}
				price, err := strconv.Atoi(r.FormValue("price"))
				if err != nil {
					http.Error(w, "price is not integer", 400)
					return
				}
				tr.Price = sql.NullInt64{Int64: int64(price), Valid: true}
				tr.Response = sql.NullString{String: r.FormValue("response"), Valid: true}
				tr.EstimateDate = sql.NullString{String: time.Now().Format("2006-01-02 15:04:05"), Valid: true}
				err = tr.Update()
				if err != nil {
					log.Println(err)
					http.Error(w, "failed to update trans", 500)
					return
				}
				respHead := strings.Replace(tr.Response.String, "\n", " ", -1)
				if len([]rune(respHead)) > 16 {
					respHead = string([]rune(respHead)[:16]) + "…"
				}
				n := accounts.Notif{
					Type: "trans/estedit",
					Text: respHead,
					From: tr.To,
					To:   tr.From,
					Id:   tr.Id,
				}
				err = n.Insert()
				if err != nil {
					log.Println(err)
					errorData.Insert("failed to insert notif "+strconv.Itoa(login.Id)+" to "+strconv.Itoa(tr.From), err.Error())
				}
				bytes, err := json.Marshal(tr)
				if err != nil {
					http.Error(w, "failed to convert trans object to json", 500)
					return
				}
				fmt.Fprintf(w, string(bytes))
			} else {
				http.Error(w, "parameter not enough", 400)
			}
		} else {
			http.Error(w, "誰やねんおまえ", 404)
		}
	} else if r.Method == http.MethodDelete {
		w.Header().Set("Content-Type", "application/json")
		mode := r.URL.Path[len("/trans/"):]
		if strings.HasPrefix(mode, "estimate/") {
			trid, err := strconv.Atoi(mode[len("estimate/"):])
			if err != nil {
				http.Error(w, "trans id is not set", 404)
				return
			}
			tr := trans.Trans{Id: trid}
			if !tr.Get() {
				http.Error(w, "failed to get trans object", 500)
				return
			}
			login := account.LoginAccount(r)
			if login.Id == -1 {
				http.Error(w, "not logined", 403)
				return
			}
			if login.Id != tr.To {
				http.Error(w, "account not allowed", 403)
				return
			}
			r.ParseMultipartForm(32 << 20)
			if isset(r, []string{"response"}) {
				tr.Response = sql.NullString{Valid: true, String: r.FormValue("response")}
				tr.EstimateDate = sql.NullString{Valid: true, String: time.Now().Format("2006-01-02 15:04:05")}
				tr.ResponseType = sql.NullInt64{Valid: true, Int64: 1}
				err = tr.Update()
				if err != nil {
					http.Error(w, "failed to update", 500)
					return
				}
				res := strings.Replace(r.FormValue("response"), "\n", " ", -1)
				if len([]rune(res)) > 16 {
					res = string([]rune(res)[:16]) + "…"
				}
				n := accounts.Notif{
					Type: "trans/rescancel",
					Text: res,
					From: tr.To,
					To:   tr.From,
					Id:   tr.Id,
				}
				err = n.Insert()
				if err != nil {
					log.Println(err)
					errorData.Insert("failed to insert notif "+strconv.Itoa(tr.From)+" to "+strconv.Itoa(tr.To), err.Error())
				}
				bytes, err := json.Marshal(tr)
				if err != nil {
					http.Error(w, "failed to convert trans object to json", 500)
					return
				}
				fmt.Fprintf(w, string(bytes))
			} else {
				tr.Response = sql.NullString{Valid: false}
				tr.EstimateDate = sql.NullString{Valid: false}
				tr.ResponseType = sql.NullInt64{Valid: false}
				err = tr.Update()
				if err != nil {
					http.Error(w, "failed to update", 500)
					return
				}
				n := accounts.Notif{
					Type: "trans/estdel",
					Text: tr.RequestTitle,
					From: tr.To,
					To:   tr.From,
					Id:   tr.Id,
				}
				err = n.Insert()
				if err != nil {
					log.Println(err)
					errorData.Insert("failed to insert notif "+strconv.Itoa(tr.From)+" to "+strconv.Itoa(tr.To), err.Error())
				}
				bytes, err := json.Marshal(tr)
				if err != nil {
					http.Error(w, "failed to convert trans object to json", 500)
					return
				}
				fmt.Fprintf(w, string(bytes))
			}
		} else if strings.HasPrefix(mode, "req/") {
			trid, err := strconv.Atoi(mode[len("req/"):])
			if err != nil {
				http.Error(w, "trans id is not set", 404)
				return
			}
			tr := trans.Trans{Id: trid}
			if !tr.Get() {
				http.Error(w, "failed to get trans object", 500)
				return
			}
			login := account.LoginAccount(r)
			if login.Id == -1 {
				http.Error(w, "not logined", 403)
				return
			}
			if login.Id != tr.From {
				http.Error(w, "account not allowed", 403)
				return
			}
			tr.RequestCancel = 1
			err = tr.Update()
			if err != nil {
				http.Error(w, "failed to update", 500)
				return
			}
			n := accounts.Notif{
				Type: "trans/reqcancel",
				Text: tr.RequestTitle,
				From: tr.From,
				To:   tr.To,
				Id:   tr.Id,
			}
			err = n.Insert()
			if err != nil {
				log.Println(err)
				errorData.Insert("failed to insert notif "+strconv.Itoa(tr.From)+" to "+strconv.Itoa(tr.To), err.Error())
			}
			bytes, err := json.Marshal(tr)
			if err != nil {
				http.Error(w, "failed to convert trans object to json", 500)
				return
			}
			fmt.Fprintf(w, string(bytes))
		} else {
			http.Error(w, "なにさ", 404)
		}
	} else {
		http.Error(w, "method not allowed.", 405)
	}
}

func InboxHandle(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html")
		login := account.LoginAccount(r)
		if login.Id == -1 {
			http.Redirect(w, r, "/", 303)
			return
		}
		msg := ""
		notifs, err := login.Inbox()
		if err != nil {
			http.Error(w, "failed to fetch inbox", 500)
			log.Println(err)
			return
		}
		acc := make([]accounts.Accounts, 0)
		db := database.Connect()
		defer db.Close()
		for _, n := range notifs {
			ac := accounts.Accounts{Id: n.From}
			if !ac.GetLite(db) {
				http.Error(w, "failed to get accounts into", 500)
				return
			}
			acc = append(acc, ac)
		}
		bytes, err := json.Marshal(struct {
			Accounts []accounts.Accounts `json:"accounts"`
			Notifs   []accounts.Notif    `json:"notifs"`
		}{
			Accounts: acc,
			Notifs:   notifs,
		})
		if err != nil {
			log.Println(err)
			http.Error(w, "failed to convert object to json", 500)
			return
		}
		msg = string(bytes)
		temp := template.Must(template.ParseFiles("template/inbox.html"))
		if err := temp.Execute(w, TempContext{
			Login:   login,
			Message: msg,
		}); err != nil {
			log.Println(err)
			http.Error(w, "HTTP 500 Internal server error", 500)
		}
	} else {
		http.Error(w, "method not allowed.", 405)
	}
}

func PaymentHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	login := account.LoginAccount(r)
	if login.Id == -1 {
		http.Redirect(w, r, "/", 303)
		return
	}

	if r.Method == http.MethodGet {
		type StripeInfo struct {
			PublicKey string `json:"pk"`
			SecretKey string `json:"sk"`
		}
		filename := r.URL.Path[len("/payment/"):]
		if strings.HasSuffix(filename, "/") {
			filename = filename[:len(filename)-1]
		}
		bytes, _ := json.Marshal(StripeInfo{
			PublicKey: os.Getenv("STRIPE_API_KEY"),
			SecretKey: stripe.GetClientSecret(),
		})
		temp := template.Must(template.ParseFiles("template/payment/" + filename + ".html"))
		if err := temp.Execute(w, TempContext{
			Message: string(bytes),
			Login:   login,
		}); err != nil {
			log.Println(err)
			http.Error(w, "HTTP 500 Internal server error", 500)
			return
		}
	} else if r.Method == http.MethodPost {
		r.ParseMultipartForm(32 << 20)
		if login.Get() {
			mode := r.URL.Path[len("/payment/"):]
			if strings.HasSuffix(mode, "/") {
				mode = mode[:len(mode)-1]
			}
			if mode == "card" {
				if login.StripeCustomer != "" {
					err := stripe.DeleteCustomer(login.StripeCustomer)
					if err != nil {
						log.Println("main.go PaymentHandle(w http.ResponseWriter, r *http.Request) Method: POST")
						log.Println(err)
						http.Error(w, "failed to delete old customer", 500)
						return
					}
				}
				cusId := stripe.CreateCustomer(login.Email, login.Name, r.FormValue("token"))
				if cusId == "" {
					http.Error(w, "failed to create customer of stripe", 500)
					return
				}
				if err := login.SetCustomerId(cusId); err != nil {
					http.Error(w, "failed to set customer id to your account", 500)
					return
				}
				fmt.Fprintf(w, "true")
			} else {
				http.Error(w, "different mode", 404)
			}
		} else {
			http.Error(w, "failed to get account info", 500)
			return
		}
	} else if r.Method == http.MethodDelete {
		if login.Get() {
			mode := r.URL.Path[len("/payment/"):]
			if strings.HasSuffix(mode, "/") {
				mode = mode[:len(mode)-1]
			}
			if mode == "card" {
				if login.StripeCustomer != "" {
					err := stripe.DeleteCustomer(login.StripeCustomer)
					if err != nil {
						log.Println("main.go PaymentHandle(w http.ResponseWriter, r *http.Request) Method: POST")
						log.Println(err)
						http.Error(w, "failed to delete old customer", 500)
						return
					}
					if err := login.SetCustomerId(""); err != nil {
						http.Error(w, "failed to set customer id to your account", 500)
						return
					}
				}
				fmt.Fprintf(w, "true")
			} else {
				http.Error(w, "different mode", 404)
			}
		} else {
			http.Error(w, "failed to get account info", 500)
			return
		}
	} else {
		http.Error(w, "method not allowed.", 405)
	}
}

func LiveHandle(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		trid, err := strconv.Atoi(r.URL.Path[len("/live/"):])
		if err == nil {
			tr := trans.Trans{Id: trid}
			if !tr.Get() {
				http.Redirect(w, r, "/home/", 302)
				return
			}
			liver := accounts.Accounts{Id: tr.From}
			if !liver.Get() {
				http.Error(w, "failed to fetch account info", 500)
				return
			}
			interpreter := accounts.Accounts{Id: tr.To}
			if !interpreter.Get() {
				http.Error(w, "failed to fetch account info", 500)
				return
			}
			transConx := struct {
				Liver       accounts.Accounts `json:"liver"`
				Interpreter accounts.Accounts `json:"interpreter"`
				Begin       string            `json:"begin"`
				Length      int               `json:"length"`
			}{
				Liver:       liver,
				Interpreter: interpreter,
				Begin:       tr.LiveStart.String,
				Length:      int(tr.LiveTime.Int64),
			}
			bytes, err := json.Marshal(transConx)
			if err != nil {
				http.Error(w, "failed to convert object to json", 500)
				return
			}
			temp := template.Must(template.ParseFiles("template/live.html"))
			if err := temp.Execute(w, TempContext{
				Login:   account.LoginAccount(r),
				Trans:   tr,
				Message: string(bytes),
			}); err != nil {
				log.Println(err)
				http.Error(w, "HTTP 500 Internal server error", 500)
			}
		} else {
			if strings.Index(r.Referer(), "/trans/") > 0 {
				trid_str := r.Referer()[strings.Index(r.Referer(), "/trans/")+len("/trans/"):]
				trid, err = strconv.Atoi(trid_str)
				if err == nil {
					http.Redirect(w, r, "/live/"+trid_str, 302)
				} else {
					http.Error(w, "page not found", 404)
				}
			} else {
				http.Error(w, "page not found", 404)
			}
		}
	} else {
		http.Error(w, "method not allowed", 405)
	}
}

func documentHandle(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html")
		cookie, err := r.Cookie("manage")
		if err != nil {
			temp := template.Must(template.ParseFiles("template/manage.html"))
			if err = temp.Execute(w, ""); err != nil {
				log.Fatal(err)
			}
		} else if cookie.Value == "ok" {
			temp := template.Must(template.ParseFiles("template/sekkei.html"))
			if err := temp.Execute(w, ""); err != nil {
				log.Fatal(err)
			}
		} else {
			temp := template.Must(template.ParseFiles("template/manage.html"))
			if err = temp.Execute(w, ""); err != nil {
				log.Fatal(err)
			}
		}
	} else if r.Method == http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		r.ParseMultipartForm(32 << 20)
		if r.FormValue("pass") == "v!a@7osV7RBmJto" || r.FormValue("pass") == "nozomikawaii" {
			cookie := &http.Cookie{
				Name:     "manage",
				Value:    "ok",
				Path:     "/",
				HttpOnly: true,
				MaxAge:   3600 * 24 * 3,
			}
			http.SetCookie(w, cookie)
			fmt.Fprintf(w, "true")
		} else {
			http.Error(w, "false", 400)
		}
	}
}

func SocketHandle(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r2 *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}

	defer ws.Close()

	clients[ws] = r.URL.Path[len("/ws/"):]

	for {
		var msg SocketMessage
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, ws)
			break
		}
		msg.ChatId = r.URL.Path[len("/ws/"):]
		broadcast <- msg
	}
}

func handleMessages() {
	for {
		msg := <-broadcast
		for client, id := range clients {
			if id == msg.ChatId {
				err := client.WriteJSON(msg)
				if err != nil {
					log.Printf("error: %v", err)
					client.Close()
					delete(clients, client)
				}
			}
		}
	}
}

//GETでは使えない
func isset(r *http.Request, keys []string) bool {
	for _, v := range keys {
		exist := false
		for k, _ := range r.MultipartForm.Value {
			if v == k {
				exist = true
			}
		}
		if !exist {
			return false
		}
	}
	return true
}
