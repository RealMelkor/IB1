package web

import (
	"github.com/gin-gonic/gin"
	"IB1/config"
)

type session map[string]any
var sessions = map[string]session{}

func getID(c *gin.Context) (string, error) {
	v, err := c.Cookie("id")
	if v == "" || err != nil {
		token, err := newToken()
		if err != nil { return "", err }
		c.SetCookie("id", token, 0, "/", config.Cfg.Web.Domain,
				false, true)
	}
	_, ok := sessions[v]
	if !ok { sessions[v] = session{} }
	return v, nil
}

func get(c *gin.Context) func(string)any {
	id, err := getID(c)
	if err != nil { return func(string)any { return nil } }
	return func(param string)any {
		v, ok := sessions[id][param]
		if !ok { return nil }
		return v
	}
}

func set(c *gin.Context) func(string, any) any {
	id, err := getID(c)
	if err != nil { return func(string, any) any { return "" } }
	return func(param string, value any) any {
		_, ok := sessions[id]
		if !ok { sessions[id] = session{} }
		sessions[id][param] = value
		return nil
	}
}

func once(c *gin.Context) func(string)any {
	id, err := getID(c)
	if err != nil { return func(string)any { return nil } }
	return func(param string)any {
		v, ok := sessions[id][param]
		if !ok { return nil }
		delete(sessions[id], param)
		return v
	}
}

func has(c *gin.Context) func(string)bool {
	id, err := getID(c)
	if err != nil { return func(string)bool { return false } }
	return func(param string)bool {
		_, ok := sessions[id][param]
		return ok
	}
}
