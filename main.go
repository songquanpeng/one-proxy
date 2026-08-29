package main

import (
	"embed"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"log"
	"one-proxy/common"
	"one-proxy/middleware"
	"one-proxy/model"
	"one-proxy/router"
	"os"
	"strconv"
)

//go:embed web/build
var buildFS embed.FS

//go:embed web/build/index.html
var indexPage []byte

type cookieMaxAgeStore interface {
	MaxAge(int)
}

type redisMaxAgeStore interface {
	SetMaxAge(int)
}

func configureSessionStore(store sessions.Store) {
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   common.SessionMaxAge,
		HttpOnly: true,
	})

	// Both stores also apply MaxAge to their secure-cookie codecs. Updating
	// only the browser cookie would leave the codec's shorter default expiry in
	// place and reject otherwise-valid sessions early.
	if cookieStore, ok := store.(cookieMaxAgeStore); ok {
		cookieStore.MaxAge(common.SessionMaxAge)
	}
	if redisStore, ok := store.(redisMaxAgeStore); ok {
		redisStore.SetMaxAge(common.SessionMaxAge)
	}
}

func main() {
	common.ParseCommandLine()
	common.SetupGinLog()
	common.SysLog("One Proxy " + common.Version + " started")
	if os.Getenv("GIN_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}
	// Initialize SQL Database
	err := model.InitDB()
	if err != nil {
		common.FatalLog(err)
	}
	if err = model.InitSessionSecret(); err != nil {
		common.FatalLog(err)
	}
	defer func() {
		err := model.CloseDB()
		if err != nil {
			common.FatalLog(err)
		}
	}()

	// Initialize Redis
	err = common.InitRedisClient()
	if err != nil {
		common.FatalLog(err)
	}

	// Initialize options
	model.InitOptionMap()

	// Initialize HTTP server
	server := gin.Default()
	server.Use(middleware.CORS())

	// Initialize session store
	if common.RedisEnabled {
		opt := common.ParseRedisOption()
		store, storeErr := redis.NewStore(opt.MinIdleConns, opt.Network, opt.Addr, opt.Password, []byte(common.SessionSecret))
		if storeErr != nil {
			common.FatalLog(storeErr)
		}
		configureSessionStore(store)
		server.Use(sessions.Sessions("session", store))
	} else {
		store := cookie.NewStore([]byte(common.SessionSecret))
		configureSessionStore(store)
		server.Use(sessions.Sessions("session", store))
	}

	router.SetRouter(server, buildFS, indexPage)
	var port = os.Getenv("PORT")
	if port == "" {
		port = strconv.Itoa(*common.Port)
	}
	err = server.Run(":" + port)
	if err != nil {
		log.Println(err)
	}
}
