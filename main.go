package main

import (
	"fmt"
	"log"
	"net/http"
	"html/template"
	"strconv"
	"strings"
	"os"
	"io"
	"encoding/json"
	"image"
	_ "image/jpeg"
	"image/png"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/nfnt/resize"
	"github.com/otofuto/LiveInterpreting/pkg/database/accounts"
	"github.com/otofuto/LiveInterpreting/pkg/database/langs"
)

var port string

func main() {
	port = os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("./static"))))
	http.HandleFunc("/Account/", AccountHandle)
	http.HandleFunc("/AccountSocial/", AccountSocialHandle)
	http.HandleFunc("/Login/", LoginHandle)
	http.HandleFunc("/Logout/", LogoutHandle)
	http.HandleFunc("/u/", UserHandle)
	http.HandleFunc("/edit/", EditHandle)
	http.HandleFunc("/home/", HomeHandle)
	http.HandleFunc("/Lang/", LangHandle)

	log.Println("Listening on port: " + port)
	log.Fatal(http.ListenAndServe(":" + port, nil))
}

func AccountHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

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
		if ac.UserType == "interpreter" {
			err = ac.SetLangs(r.FormValue("langs"))
			if err != nil {
				http.Error(w, "langs is not json", 400)
				return
			}
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
			/*save, err := os.Create("./static/img/accounts/" + ac.IconImage)
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
			}*/

			pr, pw := io.Pipe()
			go func() {
				err = png.Encode(pw, img)
				if err != nil {
					log.Fatal(err)
				}
				pw.Close()
			}()

			sess := session.Must(session.NewSession(&aws.Config{
				Credentials: credentials.NewStaticCredentials(os.Getenv("IAM_ACCESSKEY"), os.Getenv("IAM_SECRETKEY"), ""),
				Region: aws.String(os.Getenv("S3_REGION")),
			}))

			uploader := s3manager.NewUploader(sess)
			_, err = uploader.Upload(&s3manager.UploadInput{
				Bucket: aws.String(os.Getenv("S3_BUCKET")),
				Key: aws.String("accounts/" + ac.IconImage),
				Body: pr,
			})
			if err != nil {
				fmt.Println("S3アップロード失敗")
				fmt.Println(err)
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
	} else if r.Method == http.MethodGet {
		mode := r.URL.Path[len("/Account/"):]
		if strings.HasPrefix(mode, "CheckMail") {
			email := r.FormValue("email")
			accountId, err := strconv.Atoi(r.FormValue("id"))
			if err != nil {
				accountId = -1
			}
			if email != "" {
				if accounts.CheckMail(email, accountId) {
					fmt.Fprintf(w, "true")
				} else {
					fmt.Fprintf(w, "false")
				}
			} else {
				http.Error(w, "parameter 'email' is required", 400)
			}
		} else if strings.HasPrefix(mode, "Search") {
			search := r.FormValue("search")
			userType := r.FormValue("user_type")
			if search == "" && userType == "" {
				http.Error(w, "user_type or search value is must", 400)
				return
			}
			if userType != "influencer" && userType != "interpreter" && userType != ""{
				http.Error(w, "user_type value is not allowed", 400)
				return
			}
			acs := accounts.Search(search, userType)
			bytes, err := json.Marshal(acs)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Fprintf(w, string(bytes))
		} else if strings.HasPrefix(mode, "img/") {
			id, err := strconv.Atoi(r.URL.Path[len("/Account/img/"):])
			if err != nil {
				http.Error(w, "id is not integer: " + r.URL.Path[len("/Accounts/img/"):], 400)
				return
			}
			ac := accounts.Accounts { Id: id }
			if ac.Get() {
				sess := session.Must(session.NewSession(&aws.Config{
					Credentials: credentials.NewStaticCredentials(os.Getenv("IAM_ACCESSKEY"), os.Getenv("IAM_SECRETKEY"), ""),
					Region: aws.String(os.Getenv("S3_REGION")),
				}))

				svc := s3.New(sess)
				obj, err := svc.GetObject(&s3.GetObjectInput{
					Bucket: aws.String(os.Getenv("S3_BUCKET")),
					Key: aws.String("accounts/" + ac.IconImage),
				})
				if err != nil {
					fmt.Println(err)
					http.Error(w, "failed to fetch account-icon", 404)
				} else {
					w.Header().Set("Content-Type", "image/png")
					io.Copy(w, obj.Body)
					obj.Body.Close()
				}
			} else {
				http.Error(w, "account not found", 400)
			}
		}else {
			http.Error(w, "なに？", 404)
		}
	} else if r.Method == http.MethodPut {
		r.ParseMultipartForm(32 << 20)
		cookie, err := r.Cookie("accounttoken")
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Failed to get cookie 'accounttoken'.", 403)
			return
		}
		ac, err := accounts.CheckToken(cookie.Value)
		ac.Get()
		if err != nil {
			http.Error(w, "checktoken err", 403)
			fmt.Println(err)
		} else {
			if !accounts.CheckMail(ac.Email, ac.Id) {
				http.Error(w, "email already registered", 400)
				return
			}
			ac.Name = r.FormValue("name")
			ac.Description = r.FormValue("description")
			ac.Email = r.FormValue("email")

			if r.FormValue("langs") != "" {
				err = ac.SetLangs(r.FormValue("langs"))
				if err != nil {
					http.Error(w, "langs is not json", 400)
					return
				}
			}

			file, fileHeader, err := r.FormFile("icon_image")
			if err == nil {
				defer file.Close()

				img, _, err := image.Decode(file)
				if err != nil {
					fmt.Println("画像取得失敗")
					http.Error(w, "image.Decode failed", 500)
					return
				}

				//正方形にトリム
				img = ToSquare(img)
				//140角にリサイズ
				img = resize.Resize(140, 140, img, resize.Lanczos3)

				ac.IconImage = strconv.Itoa(ac.Id) + "_" + fileHeader.Filename + ".png"
				/*save, err := os.Create("./static/img/accounts/" + ac.IconImage)
				if err != nil {
					fmt.Println("ファイル確保失敗")
					http.Error(w, "upload failed", 500)
					return
				}
				defer save.Close()

				err = png.Encode(save, img)
				if err != nil {
					fmt.Println("ファイル保存失敗")
					http.Error(w, "upload failed", 500)
					return
				}*/

				pr, pw := io.Pipe()
				go func() {
					err = png.Encode(pw, img)
					if err != nil {
						log.Fatal(err)
					}
					pw.Close()
				}()

				sess := session.Must(session.NewSession(&aws.Config{
					Credentials: credentials.NewStaticCredentials(os.Getenv("IAM_ACCESSKEY"), os.Getenv("IAM_SECRETKEY"), ""),
					Region: aws.String(os.Getenv("S3_REGION")),
				}))

				uploader := s3manager.NewUploader(sess)
				_, err = uploader.Upload(&s3manager.UploadInput{
					Bucket: aws.String(os.Getenv("S3_BUCKET")),
					Key: aws.String("accounts/" + ac.IconImage),
					Body: pr,
				})
				if err != nil {
					fmt.Println("S3アップロード失敗")
					fmt.Println(err)
					ac.Delete()
					http.Error(w, "upload failed", 500)
					os.Exit(1)
					return
				}
			}

			ac.Update()

			bytes, err := json.Marshal(ac)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Fprintf(w, string(bytes))
		}
	} else {
		http.Error(w, "method not allowed.", 405)
	}
}

func AccountSocialHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodPost {
		r.ParseMultipartForm(32 << 20)
		targetId, err := strconv.Atoi(r.FormValue("target_id"))
		if err != nil {
			http.Error(w, "target_id is not integer", 400)
			return
		}
		action, err := strconv.Atoi(r.FormValue("action"))
		if err != nil {
			http.Error(w, "action is not integer", 400)
			return
		}
		if action != 0 && action != 1 {
			http.Error(w, "action value is not defined", 400)
			return
		}
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
			social := accounts.AccountSocial {
				Id: ac.Id,
				TargetId: targetId,
				Action: action,
			}
			if social.Insert() {
				fmt.Fprintf(w, "true")
			} else {
				fmt.Fprintf(w, "false")
			}
		}
	} else if r.Method == http.MethodGet {
		action, err := strconv.Atoi(r.URL.Path[len("/AccountSocial/"):])
		if err != nil {
			http.Error(w, "action is not integer", 400)
			return
		}

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
			socials, err := accounts.Social(ac.Id, action)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			bytes, err := json.Marshal(socials)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Fprintf(w, string(bytes))
		}
	} else if r.Method == http.MethodDelete {
		r.ParseMultipartForm(32 << 20)
		targetId, err := strconv.Atoi(r.FormValue("target_id"))
		if err != nil {
			http.Error(w, "target_id is not integer", 400)
			return
		}
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
			social := accounts.AccountSocial {
				Id: ac.Id,
				TargetId: targetId,
			}
			if social.Delete() {
				fmt.Fprintf(w, "true")
			} else {
				fmt.Fprintf(w, "false")
			}
		}
	} else {
		http.Error(w, "method not allowed", 405)
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
	w.Header().Set("Access-Control-Allow-Origin", "*")

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
	w.Header().Set("Access-Control-Allow-Origin", "*")

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
		cookie, err := r.Cookie("accounttoken")
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Failed to get cookie 'accounttoken'.", 403)
			return
		}
		ac, err := accounts.CheckToken(cookie.Value)
		ac.Get()
		if err != nil {
			http.Error(w, "checktoken err", 403)
			fmt.Println(err)
		} else {
			temp := template.Must(template.ParseFiles("template/edit.html"))

			if err := temp.Execute(w, ac);
			err != nil {
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
			if err := temp.Execute(w, ac);
			err != nil {
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