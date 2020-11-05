package main

import (
	"fmt"
	"log"
	"net/http"
	"html/template"
	"strconv"
	"strings"
	"os"
	//"io"
	"encoding/json"
	"image"
	_ "image/jpeg"
	"image/png"
	"github.com/nfnt/resize"
	"github.com/otofuto/LiveInterpreting/pkg/database/accounts"
)

var port string

func main() {
	port = os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("./static"))));
	http.HandleFunc("/Account/", AccountHandle)
	http.HandleFunc("/Login/", LoginHandle)
	http.HandleFunc("/Logout/", LogoutHandle)
	http.HandleFunc("/u/", UserHandle)
	http.HandleFunc("/home/", HomeHandle)

	log.Println("Listening on port: " + port)
	log.Fatal(http.ListenAndServe(":" + port, nil))
}

func AccountHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "localhost")

	if r.Method == http.MethodPost {
		r.ParseMultipartForm(32 << 20)
		sex, err := strconv.Atoi(r.FormValue("sex"))
		if err != nil {
			http.Error(w, "sex is type of int", 400)
			return;
		}
		ac := accounts.Accounts {
			UserType: r.FormValue("user_type"),
			Name: r.FormValue("name"),
			Description: r.FormValue("description"),
			Email: r.FormValue("email"),
			Sex: sex,
			Password: r.FormValue("password"),
		}
		if !accounts.CheckMail(ac.Email, -1) {
			http.Error(w, "email already registered", 400)
			return
		}
		newId := ac.Insert()
		if newId == -1 {
			http.Error(w, "insert failed", 500)
			return
		}

		file, fileHeader, err := r.FormFile("icon_image")
		if err == nil {
			defer file.Close()

			img, _, err := image.Decode(file)
			if err != nil {
				fmt.Println("画像取得失敗")
				ac.Delete()
				http.Error(w, "image.Decode failed", 500)
				return
			}

			//正方形にトリム
			img = ToSquare(img)
			//140角にリサイズ
			img = resize.Resize(140, 140, img, resize.Lanczos3)

			ac.Get()
			ac.IconImage = strconv.Itoa(newId) + "_" + fileHeader.Filename + ".png"
			ac.Update()
			save, err := os.Create("./static/img/accounts/" + ac.IconImage)
			if err != nil {
				fmt.Println("ファイル確保失敗")
				ac.Delete()
				http.Error(w, "upload failed", 500)
				return
			}
			defer save.Close()

			err = png.Encode(save, img)
			if err != nil {
				fmt.Println("ファイル保存失敗")
				ac.Delete()
				http.Error(w, "upload failed", 500)
				os.Exit(1)
				return
			}
		}

		token := ac.CreateToken()

		cookie := &http.Cookie {
			Name: "accounttoken",
			Value: token,
			Path: "/",
			HttpOnly: true,
			MaxAge: 3600 * 24 * 7,
		}
		http.SetCookie(w, cookie)

		bytes, err := json.Marshal(ac)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(w, string(bytes))
	}else if r.Method == http.MethodGet {
		mode := r.URL.Path[len("/Account/"):]
		if strings.HasPrefix(mode, "CheckMail") {
			email := r.FormValue("email")
			if email != "" {
				if accounts.CheckMail(email, -1) {
					fmt.Fprintf(w, "true")
				} else {
					fmt.Fprintf(w, "false")
				}
			} else {
				http.Error(w, "parameter 'email' is required", 400)
			}
		} else {
			http.Error(w, "なに？", 404)
		}
	} else {
		http.Error(w, "method not allowed.", 405)
	}
}

func ToSquare(img image.Image) image.Image {
	b := img.Bounds()
	x, y := b.Dx(), b.Dy()
	tl := image.Point{0, 0}

	if x == y {
		return img
	}

	if x < y {
		br := image.Point{x, x}
		cutHeight := (y - x) / 2
		ret := image.NewRGBA(image.Rectangle{tl, br})
		for y2 := 0; y2 < x; y2++ {
			for x2 := 0; x2 < x; x2++ {
				ret.Set(x2, y2, img.At(x2, y2 + cutHeight))
			}
		}
		return ret
	} else {
		br := image.Point{y, y}
		cutWidth := (x - y) / 2
		ret := image.NewRGBA(image.Rectangle{tl, br})
		for y2 := 0; y2 < x; y2++ {
			for x2 := 0; x2 < x; x2++ {
				ret.Set(x2, y2, img.At(x2 + cutWidth, y2))
			}
		}
		return ret
	}
}

func LoginHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "localhost")

	if r.Method == http.MethodPost {
		r.ParseMultipartForm(32 << 20)
		email := r.FormValue("email")
		pass := r.FormValue("password")
		ac, err := accounts.Login(email, pass)
		if err != nil {
			fmt.Println(err)
			http.Error(w, "failed", 400)
			return
		}
		token := ac.CreateToken()

		cookie := &http.Cookie {
			Name: "accounttoken",
			Value: token,
			Path: "/",
			HttpOnly: true,
			MaxAge: 3600 * 24 * 7,
		}
		http.SetCookie(w, cookie)
		bytes, err := json.Marshal(ac)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(w, string(bytes))
	} else if r.Method == http.MethodGet {
		cookie, err := r.Cookie("accounttoken")
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Failed to get cookie 'accounttoken'.", 403)
			return
		}
		ac, err := accounts.CheckToken(cookie.Value)
		if err != nil {
			http.Error(w, "checktoken err", 403)
			fmt.Println(err)
		} else {
			bytes, err := json.Marshal(ac)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Fprintf(w, string(bytes))
		}
	} else {
		http.Error(w, "method not allowed", 405)
	}
}

func LogoutHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "localhost")

	if r.Method == http.MethodPost {
		cookie, err := r.Cookie("accounttoken")
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Failed to get cookie 'accounttoken'.", 403)
			return
		}
		accounts.DeleteToken(cookie.Value)
		cookie.MaxAge = -1
		cookie.Value = "";
		http.SetCookie(w, cookie)
		fmt.Fprintf(w, "logout")
	} else {
		http.Error(w, "method not allowed", 405)
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
		ac := accounts.Accounts {
			Id: accountId,
		}
		if ac.Get() {
			ac.Password = "";
			temp := template.Must(template.ParseFiles("template/user.html"))

			if err := temp.Execute(w, ac);
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

func HomeHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Access-Control-Allow-Origin", "localhost")

	if r.Method == http.MethodGet {
		cookie, err := r.Cookie("accounttoken")
		if err != nil {
			fmt.Println(err)
			http.Redirect(w, r, "/", 301)
			return
		}
		ac, err := accounts.CheckToken(cookie.Value)
		if err != nil {
			http.Redirect(w, r, "/", 301)
			fmt.Println(err)
		} else {
			temp := template.Must(template.ParseFiles("template/home.html"))
			if err := temp.Execute(w, ac);
			err != nil {
				log.Fatal(err)
			}
		}
	} else {
		http.Error(w, "method not allowed", 405)
	}
}