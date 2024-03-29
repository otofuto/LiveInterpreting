package account

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"image"
	_ "image/jpeg"
	"image/png"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/nfnt/resize"
	"github.com/otofuto/LiveInterpreting/pkg/database/accounts"
	"github.com/otofuto/LiveInterpreting/pkg/database/errorData"
	"github.com/otofuto/LiveInterpreting/pkg/database/reports"
)

func AccountHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodPost {
		r.ParseMultipartForm(32 << 20)
		sex, err := strconv.Atoi(r.FormValue("sex"))
		if err != nil {
			http.Error(w, "sex is type of int", 400)
			return
		}
		hw, err := strconv.Atoi(r.FormValue("hourly_wage"))
		if err != nil && r.FormValue("user_type") == "interpreter" {
			http.Error(w, "hourly_wage is type of int", 400)
			return
		}
		ac := accounts.Accounts{
			UserType:    r.FormValue("user_type"),
			Name:        r.FormValue("name"),
			Description: r.FormValue("description"),
			Email:       r.FormValue("email"),
			Sex:         sex,
			Password:    r.FormValue("password"),
			Url1:        r.FormValue("url1"),
			Url2:        r.FormValue("url2"),
			Url3:        r.FormValue("url3"),
			HourlyWage:  hw,
			WageComment: r.FormValue("wage_comment"),
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
		newId, eatoken := ac.Insert()
		if newId == -1 {
			http.Error(w, "insert failed", 500)
			return
		}

		file, fileHeader, err := r.FormFile("icon_image")
		if err == nil {
			defer file.Close()

			img, _, err := image.Decode(file)
			if err != nil {
				log.Println("画像取得失敗")
				ac.Delete()
				http.Error(w, "image.Decode failed", 500)
				return
			}

			//正方形にトリム
			img = ToSquare(img)
			//140角にリサイズ
			img = resize.Resize(300, 300, img, resize.Lanczos3)

			ac.Get()
			ac.IconImage = strconv.Itoa(newId) + "_" + fileHeader.Filename + ".png"
			ac.Update()

			pr, pw := io.Pipe()
			go func() {
				err = png.Encode(pw, img)
				if err != nil {
					log.Println(err)
					http.Error(w, "png encode error", 500)
					return
				}
				pw.Close()
			}()

			sess := session.Must(session.NewSession(&aws.Config{
				Credentials: credentials.NewStaticCredentials(os.Getenv("IAM_ACCESSKEY"), os.Getenv("IAM_SECRETKEY"), ""),
				Region:      aws.String(os.Getenv("S3_REGION")),
			}))

			uploader := s3manager.NewUploader(sess)
			_, err = uploader.Upload(&s3manager.UploadInput{
				Bucket: aws.String(os.Getenv("S3_BUCKET")),
				Key:    aws.String("accounts/" + ac.IconImage),
				Body:   pr,
			})
			if err != nil {
				log.Println("S3アップロード失敗")
				log.Println(err)
				ac.Delete()
				http.Error(w, "upload failed", 500)
				return
			}
		}

		auth := smtp.PlainAuth("", os.Getenv("MAIL_ADDRESS"), os.Getenv("MAIL_PASS"), os.Getenv("MAIL_SERVER"))

		accessUrl := r.Header.Get("Referer")[:strings.Index(r.Header.Get("Referer"), "//")+2] + r.Host + "/emailauth/" + strconv.Itoa(newId) + "/?t=" + eatoken
		msg := []byte("" +
			"From: Live interpreting<" + os.Getenv("MAIL_ADDRESS") + ">\r\n" +
			"To: " + ac.Name + "<" + ac.Email + ">\r\n" +
			encodeHeader("Subject", "Live interpreting仮登録のご案内") +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/html; charset=\"utf-8\"\r\n" +
			"Content-Transfer-Encoding: base64\r\n" +
			"\r\n" +
			encodeBody(
				"<p>Live interpretingにご登録いただきありがとうございます。</p>"+
					"<p>下記URLへアクセスし、メールアドレスの認証を行って下さい。</p>"+
					"<p><a href=\""+accessUrl+"\">"+accessUrl+"</a></p>"+
					"<p><br></p>"+
					"<p><br></p>"+
					"<p>このメールは配信専用です。<br>ご返信頂いても確認および返信は出来かねますのでご了承ください。<p>"+
					"<p><br></p>"+
					"<p>Live interpreting</p>"+
					"\r\n") +
			"\r\n")

		err = smtp.SendMail(os.Getenv("MAIL_SERVER")+":587", auth, os.Getenv("MAIL_ADDRESS"), []string{ac.Email}, msg)
		if err != nil {
			log.Println(err)
			log.Println(accessUrl)
			http.Error(w, "登録に成功しましたが、確認メールの送信に失敗しました。", 500)
			return
		}

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
			ac := LoginAccount(r)
			search := r.FormValue("search")
			search = strings.Replace(search, "?", "", -1)
			userType := r.FormValue("user_type")
			if search == "" && userType == "" && r.FormValue("langs") == "" {
				http.Error(w, "user_type or search or langs value is must", 400)
				return
			}
			if userType != "influencer" && userType != "interpreter" && userType != "" {
				http.Error(w, "user_type value is not allowed", 400)
				return
			}
			acs := accounts.Search(search, userType, r.FormValue("langs"), r.FormValue("sort"), r.FormValue("wage"), ac.Id)
			bytes, err := json.Marshal(acs)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Fprintf(w, string(bytes))
		} else if strings.HasPrefix(mode, "img/") {
			id, err := strconv.Atoi(r.URL.Path[len("/Account/img/"):])
			if err != nil {
				http.Error(w, "id is not integer: "+r.URL.Path[len("/Accounts/img/"):], 400)
				return
			}
			ac := accounts.Accounts{Id: id}
			if ac.Get() {
				if ac.IconImage == "" {
					file, err := os.Open("./materials/defaulticon.png")
					if err != nil {
						http.Error(w, "icon image was not registered.", 404)
						return
					}
					defer file.Close()
					w.Header().Set("Content-Type", "image/png")
					io.Copy(w, file)
					return
				}
				sess := session.Must(session.NewSession(&aws.Config{
					Credentials: credentials.NewStaticCredentials(os.Getenv("IAM_ACCESSKEY"), os.Getenv("IAM_SECRETKEY"), ""),
					Region:      aws.String(os.Getenv("S3_REGION")),
				}))

				svc := s3.New(sess)
				obj, err := svc.GetObject(&s3.GetObjectInput{
					Bucket: aws.String(os.Getenv("S3_BUCKET")),
					Key:    aws.String("accounts/" + ac.IconImage),
				})
				if err != nil {
					log.Println(err)
					http.Error(w, "failed to fetch account-icon", 404)
				} else {
					w.Header().Set("Content-Type", "image/png")
					io.Copy(w, obj.Body)
					obj.Body.Close()
				}
			} else {
				http.Error(w, "account not found", 400)
			}
		} else if strings.HasPrefix(mode, "name/") {
			uid, err := strconv.Atoi(mode[len("name/"):])
			if err != nil {
				http.Error(w, "user id is not integer", 400)
				return
			}
			ac := accounts.Accounts{Id: uid}
			if ac.Get() {
				bytes, err := json.Marshal(accounts.Accounts{
					Id:   uid,
					Name: ac.Name,
				})
				if err != nil {
					http.Error(w, "failed to convert object to json", 500)
					return
				}
				fmt.Fprintf(w, string(bytes))
			} else {
				http.Error(w, "account not found", 404)
				return
			}
		} else {
			http.Error(w, "なに？", 404)
		}
	} else if r.Method == http.MethodPut {
		r.ParseMultipartForm(32 << 20)
		ac := LoginAccount(r)
		if ac.Id == -1 {
			http.Error(w, "not logined", 403)
		} else {
			mode := r.URL.Path[len("/Account/"):]
			if strings.HasPrefix(mode, "passreset") {
				newPass := r.FormValue("pass")
				if newPass == "" {
					http.Error(w, "pass is not allowed empty.", 400)
					return
				}
				ac.Password = newPass
				if ac.PassUpdate() {
					fmt.Fprintf(w, "true")
				} else {
					http.Error(w, "failed to update", 500)
				}
			} else {
				ac.Get()
				if !accounts.CheckMail(ac.Email, ac.Id) {
					http.Error(w, "email already registered", 400)
					return
				}
				hw, err := strconv.Atoi(r.FormValue("hourly_wage"))
				if err != nil {
					http.Error(w, "hourly_wage is not integer", 400)
					return
				}
				ac.Name = r.FormValue("name")
				ac.Description = r.FormValue("description")
				ac.Email = r.FormValue("email")
				ac.Url1 = r.FormValue("url1")
				ac.Url2 = r.FormValue("url2")
				ac.Url3 = r.FormValue("url3")
				ac.HourlyWage = hw
				ac.WageComment = r.FormValue("wage_comment")

				if r.FormValue("langs") != "" {
					err := ac.SetLangs(r.FormValue("langs"))
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
					img = resize.Resize(300, 300, img, resize.Lanczos3)

					oldImage := ac.IconImage
					ac.IconImage = strconv.Itoa(ac.Id) + "_" + fileHeader.Filename + ".png"

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
						Region:      aws.String(os.Getenv("S3_REGION")),
					}))

					uploader := s3manager.NewUploader(sess)
					_, err = uploader.Upload(&s3manager.UploadInput{
						Bucket: aws.String(os.Getenv("S3_BUCKET")),
						Key:    aws.String("accounts/" + ac.IconImage),
						Body:   pr,
					})
					if err != nil {
						fmt.Println("S3アップロード失敗")
						fmt.Println(err)
						ac.Delete()
						http.Error(w, "upload failed", 500)
						os.Exit(1)
						return
					}

					if oldImage != ac.IconImage {
						svc := s3.New(sess)
						input := &s3.DeleteObjectInput{
							Bucket: aws.String(os.Getenv("S3_BUCKET")),
							Key:    aws.String("accounts/" + oldImage),
						}
						_, err := svc.DeleteObject(input)
						if err != nil {
							errorData.Insert("Delete file from s3 failed.", "accounts/"+oldImage)
						}
					}
				}

				ac.Update()

				bytes, err := json.Marshal(ac)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Fprintf(w, string(bytes))
			}
		}
	} else if r.Method == http.MethodDelete {
		mode := r.URL.Path[len("/Account/"):]
		if mode == "delete" {
			cookie, err := r.Cookie("liveinteadmin")
			if err == nil {
				if cookie.Value == "admin" {
					aid, err := strconv.Atoi(r.FormValue("id"))
					if err != nil {
						http.Error(w, "id is not integer", 400)
						return
					}
					ac := accounts.Accounts{Id: aid}
					if ac.Get() {
						if ac.Delete() {
							fmt.Fprintf(w, "true")
						} else {
							fmt.Fprintf(w, "false")
						}
						return
					} else {
						http.Error(w, "account nto found", 404)
						return
					}
				} else {
					http.Error(w, "admin not logined", 403)
				}
			} else {
				http.Error(w, "admin not logined", 403)
			}
		} else {
			doadmin := false
			cookie, err := r.Cookie("liveinteadmin")
			if err == nil {
				if cookie.Value == "admin" {
					doadmin = true
					aid, err := strconv.Atoi(r.FormValue("id"))
					if err != nil {
						http.Error(w, "id is not integer", 400)
						return
					}
					ac := accounts.Accounts{Id: aid}
					if ac.Get() {
						if ac.Disabled() {
							fmt.Fprintf(w, "true")
						} else {
							fmt.Fprintf(w, "false")
						}
						return
					} else {
						http.Error(w, "account nto found", 404)
						return
					}
				}
			}
			if !doadmin {
				ac := LoginAccount(r)
				if ac.Id == -1 {
					http.Error(w, "not logined", 403)
					return
				}
				ac.Get()
				_, err := accounts.Login(ac.Email, r.FormValue("password"))
				if err != nil {
					http.Error(w, "unmatched password", 400)
					return
				}
				if ac.Disabled() {
					fmt.Fprintf(w, "true")
				} else {
					fmt.Fprintf(w, "false")
				}
			}
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
		ac := LoginAccount(r)
		if ac.Id == -1 {
			http.Error(w, "not logined", 403)
		} else {
			social := accounts.AccountSocial{
				Id:       ac.Id,
				TargetId: targetId,
				Action:   action,
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

		ac := LoginAccount(r)
		if ac.Id == -1 {
			http.Error(w, "not logined", 403)
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
		ac := LoginAccount(r)
		if ac.Id == -1 {
			http.Error(w, "not logined", 403)
		} else {
			social := accounts.AccountSocial{
				Id:       ac.Id,
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
				ret.Set(x2, y2, img.At(x2, y2+cutHeight))
			}
		}
		return ret
	} else {
		br := image.Point{y, y}
		cutWidth := (x - y) / 2
		ret := image.NewRGBA(image.Rectangle{tl, br})
		for y2 := 0; y2 < x; y2++ {
			for x2 := 0; x2 < x; x2++ {
				ret.Set(x2, y2, img.At(x2+cutWidth, y2))
			}
		}
		return ret
	}
}

func LoginHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodPost {
		r.ParseMultipartForm(32 << 20)
		email := r.FormValue("email")
		pass := r.FormValue("password")
		ac, err := accounts.Login(email, pass)
		if err != nil {
			if err.Error() == "no auth" {
				fmt.Fprintf(w, "{\"result\": \"no auth\"}")
			} else {
				fmt.Println(err)
				http.Error(w, "failed", 400)
			}
			return
		}
		token := ac.CreateToken()

		cookie := &http.Cookie{
			Name:     "accounttoken",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			MaxAge:   3600 * 24 * 7,
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

	if r.Method == http.MethodPost {
		cookie, err := r.Cookie("accounttoken")
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Failed to get cookie 'accounttoken'.", 403)
			return
		}
		accounts.DeleteToken(cookie.Value)
		cookie.MaxAge = -1
		cookie.Value = ""
		http.SetCookie(w, cookie)
		fmt.Fprintf(w, "logout")
	} else {
		http.Error(w, "method not allowed", 405)
	}
}

func EmailauthHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	if r.Method == http.MethodGet {
		uid_str := r.URL.Path[len("/emailauth/"):]
		if strings.Index(uid_str, "/") <= 0 {
			http.Error(w, "page not found", 404)
			return
		}
		uid_str = uid_str[:strings.Index(uid_str, "/")]
		uid, err := strconv.Atoi(uid_str)
		if err != nil {
			http.Error(w, "page not found", 404)
			return
		}
		result, err := accounts.EmailAuth(uid, r.FormValue("t"))
		if err != nil {
			log.Println(err)
			http.Error(w, "failed to update user", 500)
			return
		}

		login := accounts.Accounts{Id: -1}
		if result {
			login.Id = uid
			if !login.Get() {
				http.Error(w, "account not found", 404)
				return
			}

			token := login.CreateToken()

			cookie := &http.Cookie{
				Name:     "accounttoken",
				Value:    token,
				Path:     "/",
				HttpOnly: true,
				MaxAge:   3600 * 24 * 7,
			}
			http.SetCookie(w, cookie)
		}
		temp := template.Must(template.ParseFiles("template/emailauth.html"))
		if err := temp.Execute(w, struct {
			Result bool
			Login  accounts.Accounts
		}{
			Result: result,
			Login:  login,
		}); err != nil {
			log.Println(err)
			http.Error(w, "render error", 500)
		}
	} else if r.Method == http.MethodPost {
		r.ParseMultipartForm(32 << 20)
		email := r.FormValue("email")
		pass := r.FormValue("password")
		ac, err := accounts.Login(email, pass)
		if err == nil {
			fmt.Fprintf(w, "false")
			return
		}

		if err.Error() == "no auth" {
			eatoken, err := ac.SetEmailauthToken()
			ac.Email = email
			if err != nil {
				log.Println(err)
				http.Error(w, "account update error", 500)
				return
			}
			auth := smtp.PlainAuth("", os.Getenv("MAIL_ADDRESS"), os.Getenv("MAIL_PASS"), os.Getenv("MAIL_SERVER"))

			accessUrl := r.Header.Get("Referer")[:strings.Index(r.Header.Get("Referer"), "//")+2] + r.Host + "/emailauth/" + strconv.Itoa(ac.Id) + "/?t=" + eatoken
			msg := []byte("" +
				"From: Live interpreting<" + os.Getenv("MAIL_ADDRESS") + ">\r\n" +
				"To: " + ac.Name + "<" + ac.Email + ">\r\n" +
				encodeHeader("Subject", "Live interpreting仮登録のご案内") +
				"MIME-Version: 1.0\r\n" +
				"Content-Type: text/html; charset=\"utf-8\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n" +
				"\r\n" +
				encodeBody(
					"<p>Live interpretingにご登録いただきありがとうございます。</p>"+
						"<p>下記URLへアクセスし、メールアドレスの認証を行って下さい。</p>"+
						"<p><a href=\""+accessUrl+"\">"+accessUrl+"</a></p>"+
						"<p><br></p>"+
						"<p><br></p>"+
						"<p>このメールは配信専用です。<br>ご返信頂いても確認および返信は出来かねますのでご了承ください。<p>"+
						"<p><br></p>"+
						"<p>Live interpreting</p>"+
						"\r\n") +
				"\r\n")

			err = smtp.SendMail(os.Getenv("MAIL_SERVER")+":587", auth, os.Getenv("MAIL_ADDRESS"), []string{ac.Email}, msg)
			if err != nil {
				log.Println(err)
				log.Println(accessUrl)
				http.Error(w, "failed to send email", 500)
				return
			}
			fmt.Fprintf(w, "true")
		} else {
			log.Println(err)
			http.Error(w, "failed to check email and pass", 500)
			return
		}
	} else {
		http.Error(w, "method not allowed", 405)
	}
}

func ReportsHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodPost {
		login := LoginAccount(r)
		if login.Id == -1 {
			http.Error(w, "not logined", 403)
			return
		}
		r.ParseMultipartForm(32 << 20)
		mode := r.URL.Path[len("/Reports/"):]
		if mode == "reason" {
			rs := make([]reports.Reasons, 0)
			err := json.NewDecoder(r.Body).Decode(&rs)
			if err != nil {
				log.Println(err)
				http.Error(w, "failed to decode json request", 500)
				return
			}
			err = reports.CreateReasons(rs)
			if err != nil {
				log.Println(err)
				http.Error(w, "failed to insert", 500)
				return
			}
			fmt.Fprintf(w, "true")
		} else {
			aid, err := strconv.Atoi(r.FormValue("id"))
			if err != nil {
				http.Error(w, "id is not integer", 400)
				return
			}
			reason, err := strconv.Atoi(r.FormValue("reason"))
			if err != nil {
				http.Error(w, "reason is not integer", 400)
				return
			}
			repo := reports.Reports{
				Account: aid,
				Sender:  login.Id,
				Reason: reports.Reasons{
					Id: reason,
				},
			}
			err = repo.Insert()
			if err != nil {
				log.Println(err)
				http.Error(w, "insert failed", 500)
				return
			}
			fmt.Fprintf(w, "true")
		}
	} else if r.Method == http.MethodGet {
		mode := r.URL.Path[len("/Reports/"):]
		if mode == "reason" {
			rs, err := reports.GetReasons()
			if err != nil {
				log.Println(err)
				http.Error(w, "failed to fetch", 500)
				return
			}
			bytes, _ := json.Marshal(rs)
			fmt.Fprintf(w, string(bytes))
		} else {
			http.Error(w, "api not found", 404)
		}
	} else {
		http.Error(w, "medhot not allowed", 405)
	}
}

//文字化け対策参考:
//http://psychedelicnekopunch.com/archives/1922

func encodeHeader(code string, subject string) string {
	// UTF8 文字列を指定文字数で分割する
	b := bytes.NewBuffer([]byte(""))
	strs := []string{}
	length := 13
	for k, c := range strings.Split(subject, "") {
		b.WriteString(c)
		if k%length == length-1 {
			strs = append(strs, b.String())
			b.Reset()
		}
	}
	if b.Len() > 0 {
		strs = append(strs, b.String())
	}
	// MIME エンコードする
	b2 := bytes.NewBuffer([]byte(""))
	b2.WriteString(code + ":")
	for _, line := range strs {
		b2.WriteString(" =?utf-8?B?")
		b2.WriteString(base64.StdEncoding.EncodeToString([]byte(line)))
		b2.WriteString("?=\r\n")
	}
	return b2.String()
}

// 本文を 76 バイト毎に CRLF を挿入して返す
func encodeBody(body string) string {
	b := bytes.NewBufferString(body)
	s := base64.StdEncoding.EncodeToString(b.Bytes())
	b2 := bytes.NewBuffer([]byte(""))
	for k, c := range strings.Split(s, "") {
		b2.WriteString(c)
		if k%76 == 75 {
			b2.WriteString("\r\n")
		}
	}
	return b2.String()
}

func PassForgotHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodPost {
		r.ParseMultipartForm(32 << 20)
		ac := accounts.Accounts{Email: r.FormValue("email")}
		if ac.GetFromEmail() {
			token := accounts.PassResetToken(ac.Id)
			auth := smtp.PlainAuth("", os.Getenv("MAIL_ADDRESS"), os.Getenv("MAIL_PASS"), os.Getenv("MAIL_SERVER"))

			rootUrl := r.Header.Get("Referer")[:strings.Index(r.Header.Get("Referer"), "//")+2] + r.Host
			msg := []byte("" +
				"From: Live interpreting<" + os.Getenv("MAIL_ADDRESS") + ">\r\n" +
				"To: " + ac.Name + "<" + ac.Email + ">\r\n" +
				encodeHeader("Subject", "パスワードをリセットしてください") +
				"MIME-Version: 1.0\r\n" +
				"Content-Type: text/html; charset=\"utf-8\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n" +
				"\r\n" +
				encodeBody(
					"<p>いつもLive interpretingをご利用いただきありがとうございます。</p>"+
						"<p>下記URLへアクセスし、パスワードの再設定を行ってください。</p>"+
						"<p><a href=\""+rootUrl+"/PassForgot/?t="+token+"\">"+rootUrl+"/PassForgot/?t="+token+"</a></p>"+
						"<p><br></p>"+
						"<p><br></p>"+
						"<p>このメールは配信専用です。<br>ご返信頂いても確認および返信は出来かねますのでご了承ください。<p>"+
						"<p><br></p>"+
						"<p>Live interpreting</p>"+
						"\r\n") +
				"\r\n")

			err := smtp.SendMail(os.Getenv("MAIL_SERVER")+":587", auth, os.Getenv("MAIL_ADDRESS"), []string{ac.Email}, msg)
			if err != nil {
				log.Println(err)
				log.Println(rootUrl + "/PassForgot/?t=" + token)
				http.Error(w, "failed to send email", 500)
				return
			}

			fmt.Fprintf(w, "true")
		} else {
			http.Error(w, "email is not regitered", 400)
		}
	} else if r.Method == http.MethodGet {
		if r.FormValue("t") == "" {
			http.Error(w, "token is necessary", 400)
			return
		}
		ac := accounts.CheckPassResetToken(r.FormValue("t"))
		if ac.Id != -1 {
			token := ac.CreateToken()

			cookie := &http.Cookie{
				Name:     "accounttoken",
				Value:    token,
				Path:     "/",
				HttpOnly: true,
				MaxAge:   3600 * 24 * 7,
			}
			http.SetCookie(w, cookie)

			w.Header().Set("Content-Type", "text/html")
			temp := template.Must(template.ParseFiles("template/mypage/pass.html"))

			if err := temp.Execute(w, struct {
				Login accounts.Accounts
			}{
				Login: ac,
			}); err != nil {
				log.Fatal(err)
			}
		} else {
			http.Error(w, "トークンが無効です。再度トークンを発行してお試しください。", 400)
		}
	} else {
		http.Error(w, "method not allowed", 405)
	}
}

func LoginAccount(r *http.Request) accounts.Accounts {
	cookie, err := r.Cookie("accounttoken")
	if err != nil {
		return accounts.Accounts{Id: -1}
	}
	ac, err := accounts.CheckToken(cookie.Value)
	if err != nil {
		return accounts.Accounts{Id: -1}
	}
	return ac
}
