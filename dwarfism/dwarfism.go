package dwarfism

import (
	"database/sql"
	"fmt"
	"html/template"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"../../../../sessions"
	"../../../tools"
	"github.com/gidoBOSSftw5731/log"
)

const (
	allowedChars = "-123456789ABCDEFGHJKLMNOPRSTUVWXYZ_abcdefghijkmnopqrstuvwxyz" // 60 chars
)

type userInfo struct {
	username string
	ipAddr   string
}

// ShortPage is a function to return the homepage of ShortPage
// outputURL should be in html <a> tags
func ShortPage(resp http.ResponseWriter, req *http.Request, config tools.Config, outputURL string) {
	pageTemplate := template.New("short page templated.")
	content, err := ioutil.ReadFile("server/selector/modules/dwarfism-2.0/shortPage.html")
	page := string(content)
	if err != nil {
		log.Errorf("Failed to parse template: %v", err)
		tools.ErrorHandler(resp, req, 404, "Parsing error, please try again with fewer cosmic rays")
		return
	}
	pageTemplate, err = pageTemplate.Parse(page)
	if err != nil {
		log.Errorf("Failed to parse template: %v", err)
		return
	}
	req.ParseForm()
	//field := req.FormValue("fn")
	//fmt.Println(field)
	tData := tools.TData{
		config.RecaptchaPubKey,
		config.URLPrefix,
		outputURL}
	//upload(resp, req)
	//log.Traceln("Form data: ", field, "\ntData: ", tData)
	err = pageTemplate.Execute(resp, tData)
	if err != nil {
		log.Errorf("template execute error: %v", err)
		return

	}
}

// Biggify redirects the user to the larger url
func Biggify(resp http.ResponseWriter, req *http.Request, config tools.Config, url string) {
	if url == "" {
		http.Redirect(resp, req, "/dwarfism2.0", 301)
	}

	db, err := sql.Open("mysql", fmt.Sprintf("%s@tcp(127.0.0.1:3306)/ImgSrvr", config.SQLAcc))
	if err != nil {
		log.Error("Oh noez, could not connect to database")
		tools.ErrorHandler(resp, req, 500, "I dont know what I'm doing")
		return
	}
	log.Debug("Oi, mysql did thing")
	defer db.Close()

	var longURL string
	err = db.QueryRow("SELECT origURL FROM shortlinks WHERE shortURL=?", url).Scan(&longURL)
	switch {
	case err == sql.ErrNoRows:
		tools.ErrorHandler(resp, req, 404, "idk maybe check for typos?")
	case err != nil:
		tools.ErrorHandler(resp, req, 500, "no clue, but it probably was your fault")
		return
	default:
	}

	if !strings.HasPrefix(longURL, "http") {
		longURL = "http://" + longURL
	}

	http.Redirect(resp, req, longURL, 301)
}

// ShortResp responds to the form from ShortPage
func ShortResp(resp http.ResponseWriter, req *http.Request, config tools.Config) {
	req.ParseForm()
	var (
		shortURL, longURL, username string
	)
	isLoggedIn, _ := sessions.Verify(resp, req, config.SQLAcc, &username)

	longURL = req.FormValue("lURL")
	if longURL == "" {
		tools.ErrorHandler(resp, req, 400, "You need to  specify a url!")
		return
	}

	if !strings.HasPrefix(longURL, "http") {
		longURL = "http://" + longURL
	}

	db, err := sql.Open("mysql", fmt.Sprintf("%s@tcp(127.0.0.1:3306)/ImgSrvr", config.SQLAcc))
	if err != nil {
		log.Error("Oh noez, could not connect to database")
		tools.ErrorHandler(resp, req, 500, "I dont know what I'm doing")
		return
	}
	log.Debug("Oi, mysql did thing")
	defer db.Close()

	if isLoggedIn {
		err = db.QueryRow("SELECT * FROM shortlinks WHERE shortURL = ?", req.FormValue("sURL")).Scan()
		//log.Traceln(err)
		if err == sql.ErrNoRows {
			shortURL = req.FormValue("sURL")
		} else {
			tools.ErrorHandler(resp, req, 403, "Shortlink taken, please use a new one")
			return
		}
	}
	if shortURL == "" {
		var x int
		rand.Seed(time.Now().UnixNano())
		allowedCharsSplit := strings.Split(allowedChars, "")
		for i := 0; i < 6; i++ {
			x = rand.Intn(len(allowedChars) - 1) // Not helpful name, but this generates a randon number from 0 to 84 to locate what we need for the session
			shortURL += allowedCharsSplit[x]     // Using x to navigate the split for one character
		}
	}

	err = Shorten(longURL, shortURL, db, userInfo{username, req.RemoteAddr})
	if err != nil {
		tools.ErrorHandler(resp, req, 500, "I'm pressing the buttons but it's not goin' anywhere!")
		log.Errorln("Error while shortening: ", err)
	}

	fullShortURL := config.BaseURL + config.URLPrefix + "d/" + shortURL

	ShortPage(resp, req, config, fullShortURL)
}

// Shorten is a function to add a shortURL to the DB
func Shorten(origURL, shortURL string, db *sql.DB, user userInfo) error {
	if user.username == "" {
		user.username = "anon"
	}

	_, err := db.Exec("INSERT INTO shortlinks VALUES(?, ?, ?, ?)", origURL, shortURL, user.username, user.ipAddr)

	return err
}
