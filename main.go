package main

import (
	"fmt"
	"log"
	"strings"
	"net/http"
	"html/template"
	"strconv"
	"os"
	"io"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/otofuto/LiveInterpreting/pkg/account"
	"github.com/otofuto/LiveInterpreting/pkg/database/accounts"
	"github.com/otofuto/LiveInterpreting/pkg/database/langs"
	"github.com/otofuto/LiveInterpreting/pkg/database/directMessages"
)

var port string

//WebSocket関係
var clients = make(map[*websocket.Conn]string)
var broadcast = make(chan SocketMessage)
var upgrader = websocket.Upgrader{}

type SocketMessage struct {
	Message string `json:"message"`
	From int `json:"from"`
	Id int `json:"id"`
	CreatedAt string `json:"created_at"`
	ChatId string `json:"chat_id"`
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

	http.HandleFunc("/u/", UserHandle)
	http.HandleFunc("/edit/", EditHandle)
	http.HandleFunc("/home/", HomeHandle)
	http.HandleFunc("/Lang/", LangHandle)
	http.HandleFunc("/directmessages/", DMHandle)

	http.HandleFunc("/ws/", SocketHandle)
	go handleMessages()

	log.Println("Listening on port: " + port)
	log.Fatal(http.ListenAndServe(":" + port, nil))
}

func IndexHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodGet {
		cookie, err := r.Cookie("accounttoken")
		if err != nil {
			temp := template.Must(template.ParseFiles("template/index.html"))
			if err := temp.Execute(w, "");
			err != nil {
				log.Fatal(err)
			}
			return
		}
		_, err = accounts.CheckToken(cookie.Value)
		if err != nil {
			temp := template.Must(template.ParseFiles("template/index.html"))
			if err := temp.Execute(w, "");
			err != nil {
				log.Fatal(err)
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

func UserHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodGet {
		accountId, err := strconv.Atoi(r.URL.Path[len("/u/"):])
		if err != nil {
			http.Error(w, "URLが正しくありません。", 400)
			return
		}
		ac := accounts.Accounts {
			Id: accountId,
		}
		if ac.Get() {
			if ac.Enabled == 0 {
				temp := template.Must(template.ParseFiles("template/disableduser.html"))

				if err := temp.Execute(w, ""); err != nil {
					log.Fatal(err)
				}
				return
			}
			ac.Password = "";
			context := struct {
				Account accounts.Accounts `json:"account"`
				Login accounts.Accounts `json:"login"`
				IsFollow bool `json:"is_follow"`
				IsFollower bool `json:"is_follower"`
			}{
				Account: ac,
				Login: accounts.Accounts{ Id: -1 },
				IsFollow: false,
				IsFollower: false,
			}
			cookie, err := r.Cookie("accounttoken")
			if err == nil {
				loginaccount, err := accounts.CheckToken(cookie.Value)
				if err == nil {
					context.Login = loginaccount
					context.IsFollow = accounts.CheckFollow(loginaccount.Id, ac.Id)
					context.IsFollower = accounts.CheckFollow(ac.Id, loginaccount.Id)
				}
			}
			temp := template.Must(template.ParseFiles("template/user.html"))

			if err := temp.Execute(w, context);
			err != nil {
				log.Fatal(err)
			}
		} else {
			http.Error(w, "このユーザーは存在しません。", 404)
		}
	} else {
		http.Error(w, "method not allowed.", 405)
	}
}

func EditHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodGet {
		ac := account.LoginAccount(r)
		if ac.Id == -1 {
			http.Error(w, "not logined", 403)
			return
		}
		mode := r.URL.Path[len("/edit/"):]
		ac.Get()
		if strings.HasPrefix(mode, "pass") {
			temp := template.Must(template.ParseFiles("template/passreset.html"))

			if err := temp.Execute(w, ac); err != nil {
				log.Fatal(err)
			}
		} else {
			temp := template.Must(template.ParseFiles("template/edit.html"))

			if err := temp.Execute(w, ac); err != nil {
				log.Fatal(err)
			}
		}
	} else {
		http.Error(w, "method not allowed.", 405)
	}
}

func HomeHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodGet {
		cookie, err := r.Cookie("accounttoken")
		if err != nil {
			fmt.Println(err)
			http.Redirect(w, r, "/", 302)
			return
		}
		ac, err := accounts.CheckToken(cookie.Value)
		if err != nil {
			http.Redirect(w, r, "/", 302)
			fmt.Println(err)
		} else {
			temp := template.Must(template.ParseFiles("template/home.html"))
			if err := temp.Execute(w, ac); err != nil {
				log.Fatal(err)
			}
		}
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
			log.Fatal(err)
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
			http.Error(w, "not logined", 403)
			return
		}
		accountId, err := strconv.Atoi(r.URL.Path[len("/directmessages/"):])
		if err != nil {
			http.Error(w, "<h1>このページはまだ作成されていません。</h1>", 404)
		} else {
			ac := accounts.Accounts {
				Id: accountId,
			}
			if ac.Get() {
				ac.Password = "";
				ac.Email = "";
				dms, err := directMessages.List(login.Id, ac.Id)
				if err != nil {
					http.Error(w, "failed to list your direct messages.", 500)
					return
				}
				context := struct {
					Account accounts.Accounts `json:"account"`
					Login accounts.Accounts `json:"login"`
					DM []directMessages.DirectMessages `json:"dm"`
				}{
					Account: ac,
					Login: login,
					DM: dms,
				}
				temp := template.Must(template.ParseFiles("template/dm.html"))

				if err := temp.Execute(w, context);
				err != nil {
					log.Fatal(err)
				}
			} else {
				http.Error(w, "このユーザーは存在しません。", 404)
			}
		}
	} else if r.Method == http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
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
		ac := accounts.Accounts {
			Id: accountId,
		}
		if ac.Get() {
			dm := directMessages.DirectMessages {
				From: login.Id,
				To: ac.Id,
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
			msg := SocketMessage {
				Message: dm.Message,
				From: dm.From,
				Id: newId,
				CreatedAt: dm.CreatedAt,
				ChatId: chatId,
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
			after = after[strings.Index(after, "/") + 1:]
			if strings.Index(after, "/") > 0 {
				to, err := strconv.Atoi(after[:strings.Index(after, "/")])
				if err != nil {
					http.Error(w, "not integer", 400)
					return
				}
				after = after[strings.Index(after, "/") + 1:]
				id, err := strconv.Atoi(after)
				if err != nil {
					http.Error(w, "not integer", 400)
					return
				}
				dm := directMessages.DirectMessages {
					From: from,
					To: to,
					Id: id,
				}
				if dm.Read() {
					fmt.Fprintf(w, "true")
				} else {
					http.Error(w, "failed to update dm to read", 500)
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

func SocketHandle(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func (r2 * http.Request) bool { return true }
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}

	defer ws.Close();

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
		msg := <- broadcast
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