package router

import "github.com/auttaja/gommand"

// Router is used to define the command router.
var Router = gommand.NewRouter(&gommand.RouterConfig{
	PrefixCheck: gommand.MentionPrefix,
})
