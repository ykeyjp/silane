package middleware

import "github.com/ykeyjp/silane"

func JsonStrategy(c *silane.Context, next silane.NextFunc) {
	next(c)
	c.Response.Header.Set("Content-Type", "application/json")
}
