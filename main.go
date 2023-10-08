package main

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jtagcat/teinetahvel/tahvel"
	bb "github.com/jtagcat/util/bbolt"
	ginutil "github.com/jtagcat/util/gin"
	"github.com/jtagcat/util/std"
	"github.com/rs/xid"
	"go.etcd.io/bbolt"
)

var (
	TIMEZONE, _ = time.LoadLocation("Europe/Tallinn")
	FOOTER_HTML = os.Getenv("FOOTER_HTML")
	TITLE       = os.Getenv("TITLE")
)

var AUTHSESSIONS = make(map[string]string)

func init() {
	go func() {
		for {
			time.Sleep(time.Minute)

			expireLine := time.Now().Add(-3 * time.Minute)

			for id := range AUTHSESSIONS {
				xid, err := xid.FromString(id)
				if err != nil {
					slog.Error("authSessions cleanup parsing xid", std.SlogErr(err))
					continue
				}

				if xid.Time().Before(expireLine) {
					delete(AUTHSESSIONS, id)
				}
			}
		}
	}()
}

func main() {
	ctx := context.Background()
	ctx, _ = signal.NotifyContext(ctx, os.Interrupt)

	slogLevel := new(slog.LevelVar)
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slogLevel})))

	if os.Getenv("DEBUG") == "1" {
		slogLevel.Set(slog.LevelDebug)
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.Default()
	router.LoadHTMLGlob("templates/*.html")

	db, err := bbolt.Open("data/teinetahvel.db", 0o600, nil)
	if err != nil {
		slog.Error("opening database", std.SlogErr(err), slog.String("path", "data/teinetahvel.db"))
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Update(func(tx *bbolt.Tx) error {
		for _, bucket := range []string{"crowdsourced_room_acl"} {
			if _, err := tx.CreateBucket([]byte(bucket)); err != nil {
				if !errors.Is(err, bbolt.ErrBucketExists) {
					return err
				}
			}
		}

		return nil
	}); err != nil {
		slog.Error("adding buckets to database", std.SlogErr(err))
		os.Exit(1)
	}

	authHandlers(ctx, router)
	mainHandlers(ctx, router, db)
	bookingHandlers(ctx, router)

	ginutil.RunWithContext(ctx, router)
}

func authed(c *gin.Context) bool {
	if ginutil.Cookie(c, "session") == "" {
		return false
	}

	return true
}

func authHandlers(gctx context.Context, r *gin.Engine) {
	// r.POST("/login", func(c *gin.Context) {
	r.POST("/login", ginutil.HandlerWithErr(func(c *gin.Context, g *ginutil.Context) (status int, err string) {
		idCode, phone := c.PostForm("idCode"), c.PostForm("phone")
		if idCode == "" || phone == "" {
			return http.StatusBadRequest, "idCode and phone must not be empty"
		}
		if !strings.HasPrefix(phone, "+") {
			phone = "+372" + phone
		}

		ctx, cancel := context.WithTimeout(gctx, time.Minute)
		authCode := make(chan string, 1)
		authSession := xid.New().String()

		go func() {
			tahvel, err := tahvel.AuthMid(ctx, idCode, phone, authCode)
			if err != nil {
				slog.Info("failed auth", std.SlogErr(err))
				delete(AUTHSESSIONS, authSession)
				cancel()
				return
			}

			AUTHSESSIONS[authSession] = tahvel.Session
		}()

		select {
		case <-ctx.Done():
			return http.StatusForbidden, ctx.Err().Error()
		case code := <-authCode:
			AUTHSESSIONS[authSession] = ""

			c.SetCookie("prefill", strings.Join([]string{idCode, phone}, ","), 60*60*24*15, "", "", !gin.IsDebugging(), true)
			c.Redirect(http.StatusFound, fmt.Sprintf("/login-wait?authSession=%s&code=%s", authSession, code))
			return
		}
	}))

	r.GET("/login-wait", func(c *gin.Context) {
		authSession, code := c.Query("authSession"), c.Query("code")

		session, ok := AUTHSESSIONS[authSession]
		if !ok {
			c.HTML(http.StatusForbidden, "error.html", gin.H{"err": "Mobiil-ID-ga sisselogimine ebaõnnestus"})
			return
		}

		if session == "" {
			c.HTML(http.StatusAccepted, "login-wait.html", gin.H{
				"code": code,
			})
			return
		}

		c.SetCookie("session", session, 60*60*3, "", "", !gin.IsDebugging(), true) // timeout from /hois_back/user
		c.Redirect(http.StatusTemporaryRedirect, "/search")

		slog.Info("successful login", slog.String("prefill", ginutil.Cookie(c, "prefill")))
	})

	r.GET("/logout", ginutil.HandlerWithErr(func(c *gin.Context, g *ginutil.Context) (int, string) {
		t := tahvel.Tahvel{Session: g.Cookie("session")}
		ctx, cancel := context.WithTimeout(gctx, 10*time.Second)
		defer cancel()

		c.SetCookie("session", "", -1, "", "", !gin.IsDebugging(), true)
		c.SetCookie("prefill", "", -1, "", "", !gin.IsDebugging(), true)

		if err := t.Logout(ctx); err != nil {
			return http.StatusBadGateway, err.Error()
		}

		return g.Redirect(http.StatusTemporaryRedirect, "/")
	}))
}

func mainHandlers(gctx context.Context, r *gin.Engine, db *bbolt.DB) {
	r.GET("/", func(c *gin.Context) {
		if authed(c) {
			c.Redirect(http.StatusTemporaryRedirect, "/search")
			return
		}

		pageVars := gin.H{
			"title":      TITLE,
			"footerHTML": template.HTML(FOOTER_HTML),
		}

		if idCode, phone, ok := strings.Cut(ginutil.Cookie(c, "prefill"), ","); ok {
			pageVars = gin.H{
				"prefillIdCode": idCode,
				"prefillPhone":  strings.TrimPrefix(phone, "+372"),
			}
		}

		c.HTML(http.StatusOK, "index.html", pageVars)
	})

	r.GET("/search", searchHandler(gctx, db))
	r.POST("/search", searchHandler(gctx, db))

	r.GET("/crowdsource", ginutil.HandlerWithErr(func(c *gin.Context, g *ginutil.Context) (int, string) {
		if !authed(c) {
			return g.Redirect(http.StatusTemporaryRedirect, "/")
		}

		t := tahvel.Tahvel{Session: g.Cookie("session")}
		ctx, cancel := context.WithTimeout(gctx, 5*time.Second)
		defer cancel()

		user, err := t.GetUser(ctx)
		if err != nil {
			c.SetCookie("session", "", -1, "", "", !gin.IsDebugging(), true)
			return g.Redirect(http.StatusTemporaryRedirect, "/")
		}
		c.SetCookie("session", t.Session, user.SessionTimeoutInSeconds, "", "", !gin.IsDebugging(), true)

		//

		roomId, access := c.Query("room"), c.Query("access") == "1"

		accessStr := "0"
		if access {
			accessStr = "1"
		}

		if err := bb.Put(db, []byte("crowdsourced_room_acl"), roomId+":"+user.ACLCompositeName(), accessStr); err != nil {
			return http.StatusInternalServerError, err.Error()
		}

		return g.Redirect(http.StatusTemporaryRedirect, "/search")
	}))
}

func searchHandler(gctx context.Context, db *bbolt.DB) gin.HandlerFunc {
	return ginutil.HandlerWithErr(func(c *gin.Context, g *ginutil.Context) (int, string) {
		if !authed(c) {
			return g.Redirect(http.StatusTemporaryRedirect, "/")
		}

		t := tahvel.Tahvel{Session: g.Cookie("session")}
		ctx, cancel := context.WithTimeout(gctx, 20*time.Second)
		defer cancel()

		user, err := t.GetUser(ctx)
		if err != nil {
			c.SetCookie("session", "", -1, "", "", !gin.IsDebugging(), true)
			return g.Redirect(http.StatusTemporaryRedirect, "/")
		}
		c.SetCookie("session", t.Session, user.SessionTimeoutInSeconds, "", "", !gin.IsDebugging(), true)

		//
		now := time.Now().In(TIMEZONE)

		bookings, err := t.Bookings(ctx, now)
		if err != nil {
			return http.StatusBadGateway, "listing bookings: " + err.Error()
		}

		pageVars := gin.H{
			"unknownACL": user.UnknownACLs(),

			"bookings": bookings,

			"today":   now.Format("2006-01-02"),
			"now":     now.Round(5 * time.Minute).Format("15:04"),
			"nowplus": now.Round(5 * time.Minute).Add(45 * time.Minute).Format("15:04"),
		}

		date, err := time.Parse("2006-01-02", c.PostForm("date"))
		if err != nil {
			return g.HTML(http.StatusFound, "search.html", pageVars)
		}

		rooms, err := t.GetRooms(ctx, date)
		if err != nil {
			return http.StatusBadGateway, "listing rooms: " + err.Error()
		}

		fuzzy := time.Minute // TODO: test
		rooms, conflicting, dicks := tahvel.FilterRooms(db, rooms, user.Roles,
			c.PostForm("needsPiano") == "needsPiano",
			c.PostForm("startTime"), c.PostForm("stopTime"), fuzzy,
		)

		equipment, err := tahvel.GetEquipment(ctx)
		if err != nil {
			return http.StatusBadGateway, "listing equipment: " + err.Error()
		}
		// equipment = tahvel.FilterEquipmentReferenced(equipment, rooms)

		var hasCrowdsource bool
		for i, r := range rooms {
			var resolvedEquipment []string

			for _, e := range r.FlatEquipment() {
				resolvedEquipment = append(resolvedEquipment, strings.TrimPrefix(equipment[e], "_"))
			}
			rooms[i].ResolvedEquipmnet = strings.Join(resolvedEquipment, ", ")

			if r.MissingACL {
				hasCrowdsource = true
			}
		}

		maps.Copy(pageVars, gin.H{
			"hasCrowdsource": hasCrowdsource,
			"rooms":          rooms,

			"bookStart": c.PostForm("startTime"),
			"bookStop":  c.PostForm("stopTime"),

			"conflicting": conflicting,

			"dicks": dicks,
		})

		return g.HTML(http.StatusFound, "search.html", pageVars)
	})
}

func bookingHandlers(gctx context.Context, r *gin.Engine) {
	r.GET("/book", ginutil.HandlerWithErr(func(c *gin.Context, g *ginutil.Context) (int, string) {
		if !authed(c) {
			return g.Redirect(http.StatusTemporaryRedirect, "/")
		}

		t := tahvel.Tahvel{Session: g.Cookie("session")}
		ctx, cancel := context.WithTimeout(gctx, 5*time.Second)
		defer cancel()

		todayDateS := time.Now().In(TIMEZONE).Format("2006-01-02")
		startT, _ := time.Parse("2006-01-02 15:04", todayDateS+" "+c.Query("start"))
		stopT, _ := time.Parse("2006-01-02 15:04", todayDateS+" "+c.Query("stop"))
		id, _ := strconv.Atoi(c.Query("id"))

		if err := t.CreateBooking(ctx, id, startT, stopT); err != nil {
			return http.StatusBadGateway, err.Error()
		}

		return g.Redirect(http.StatusTemporaryRedirect, "/")
	}))

	r.GET("/cancel", ginutil.HandlerWithErr(func(c *gin.Context, g *ginutil.Context) (int, string) {
		if !authed(c) {
			return g.Redirect(http.StatusTemporaryRedirect, "/")
		}

		t := tahvel.Tahvel{Session: g.Cookie("session")}
		ctx, cancel := context.WithTimeout(gctx, 5*time.Second)
		defer cancel()

		if err := t.CancelBooking(ctx, c.Query("id")); err != nil {
			return http.StatusBadGateway, err.Error()
		}

		return g.Redirect(http.StatusTemporaryRedirect, "/")
	}))
}
