package main

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
)

var user2psd map[string]string = make(map[string]string)

func main() {

	t, err := template.New("demo").Parse(`Hello, {{.Username}}! Main Page: [{{.MainPage}}]`)
	template.Must(t, err)
	args1 := map[string]string{"Username": "Hypermind", "MainPage": "http://hypermind.com.cn/go"}
	err = t.Execute(os.Stdout, args1)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", handler)
	http.HandleFunc("/mon", monitor)
	http.HandleFunc("/login", login)
	// err := http.ListenAndServe(":9092", nil)
	// if err != nil {
	// 	log.Fatal("ListenAndServe: ", err)
	// }
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hi i love %s", r.URL.Path[1:])
}

func initData() {
	user2psd["u1"] = "p1"
	user2psd["u2"] = "p2"
	user2psd["u3"] = "p3"
}

func loginCheck(username string, password string) (err error) {
	bInvalidUsername := false
	bInvalidPassword := false
	err = nil
	for k, v := range user2psd {
		if username == k {
			bInvalidUsername = true
			if password == v {
				bInvalidPassword = true
				break
			} else {
				bInvalidPassword = false
			}
		}
	}
	if bInvalidUsername == false {
		fmt.Println(username, " doesn't exists.")
		err = errors.New(username + " doesn't exists.")
		//return UNAME_NO_EXIST
	} else if bInvalidPassword == false {
		fmt.Println("wrong password")
		err = errors.New("wrong password")
		//return PASSWORD_WRONG
	}
	return err
}

func monitor(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		fmt.Println("get monitor")
		// t, _ := template.ParseFiles("mon.html")
		// t.Execute(w, nil)
	} else {
		fmt.Println("post monitor")
		r.ParseForm()

		username := template.HTMLEscapeString(r.Form.Get("username"))
		password := template.HTMLEscapeString(r.Form.Get("password"))
		fmt.Println(username, password)
		if err := loginCheck(username, password); err != nil {
			t, _ := template.ParseFiles("login_fail.html")
			t.Execute(w, err.Error())
		} else {
			fmt.Println("login successful!")
			t, _ := template.ParseFiles("mon.html")
			t.Execute(w, nil)
		}
	}
}
func login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		fmt.Println("get login")
		t, _ := template.ParseFiles("login.html")
		t.Execute(w, nil)
	} else {
		fmt.Println("post login")
	}
}
